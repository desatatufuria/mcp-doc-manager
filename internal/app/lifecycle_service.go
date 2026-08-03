package app

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"time"

	gitadapter "github.com/desatatufuria/mcp-doc-manager/internal/adapters/git"
	"github.com/desatatufuria/mcp-doc-manager/internal/domain"
)

var (
	ErrPlanningDenied          = errors.New("planning_denied")
	ErrCatalogEvidenceRequired = errors.New("catalog_evidence_required")
)

type LifecyclePolicy string

const (
	ApprovalRequired           LifecyclePolicy = "approval-required"
	AutomaticAfterApprovedPlan LifecyclePolicy = "automatic-after-approved-plan"
)

type RadiographyService interface {
	Radiograph(context.Context, string) (gitadapter.Radiography, error)
}

type PlanningRequest struct {
	Repository, VisibleStorage, Audience, Language, Owner  string
	Actor, Interaction, Request, IdempotencyKey            string
	Confirmed, PlanApproved, BatchApproved, PolicyApproved bool
	Policy                                                 LifecyclePolicy
	Actions                                                []domain.PlannedAction
}

type Plan struct {
	State                                                            domain.LifecycleState
	Revision, Radiography, VisibleStorage, Audience, Language, Owner string
	ExistingDocuments                                                []ExistingDocumentTreatment
	Actions                                                          []domain.PlannedAction
}

type ExistingDocumentTreatment struct {
	Path      string
	Treatment ExistingDocumentTreatmentKind
}

type ExistingDocumentTreatmentKind string

const InPlaceInventory ExistingDocumentTreatmentKind = "in_place_inventory"

type Batch struct {
	ID      string
	State   domain.LifecycleState
	Actions []domain.PlannedAction
}

type Policy struct {
	Mode     LifecyclePolicy
	Revision string
	Approved bool
}

type PlanningResult struct {
	Radiography   gitadapter.Radiography
	Plan          Plan
	Batch         Batch
	Policy        Policy
	Authorization *domain.Authorization
	Provenance    domain.Provenance
	Err           error
}

type PlanningService struct {
	Radiography RadiographyService
	Clock       func() time.Time
}

func (s PlanningService) Plan(ctx context.Context, request PlanningRequest) PlanningResult {
	if s.Radiography == nil || !request.Confirmed || !completeContext(request) || !validPolicy(request.Policy) || !validActions(request.Actions) {
		return PlanningResult{Err: ErrPlanningDenied}
	}
	radiography, err := s.Radiography.Radiograph(ctx, request.Repository)
	if err != nil {
		return PlanningResult{Err: err}
	}
	if radiography.Identity == "" || radiography.Root == "" {
		return PlanningResult{Radiography: radiography, Err: ErrPlanningDenied}
	}
	plan := Plan{
		State:             domain.PlanProposed,
		Revision:          radiography.Identity,
		Radiography:       radiography.Identity,
		VisibleStorage:    request.VisibleStorage,
		Audience:          request.Audience,
		Language:          request.Language,
		Owner:             request.Owner,
		ExistingDocuments: existingDocuments(radiography),
		Actions:           append([]domain.PlannedAction(nil), request.Actions...),
	}
	policy := Policy{Mode: request.Policy, Revision: string(request.Policy), Approved: request.PolicyApproved}
	batch := Batch{ID: "batch:" + radiography.Identity, State: domain.BatchProposed, Actions: append([]domain.PlannedAction(nil), request.Actions...)}
	result := PlanningResult{Radiography: radiography, Plan: plan, Batch: batch, Policy: policy}
	if !request.PlanApproved || !request.PolicyApproved || !batchApproved(request) {
		result.Err = ErrPlanningDenied
		return result
	}
	result.Plan.State = domain.PlanApproved
	result.Batch.State = domain.BatchAuthorized
	result.Authorization = &domain.Authorization{
		BatchID: batch.ID, PlanRevision: plan.Revision, PolicyRevision: policy.Revision,
		Scope: domain.Scope{Kind: domain.ScopeWorktree}, Baseline: radiography.Identity, Evidence: radiography.Identity,
		Allowed: append([]domain.PlannedAction(nil), request.Actions...), Active: true,
	}
	result.Provenance = planningProvenance(request, result, s.now())
	if err := result.Provenance.Validate(); err != nil {
		result.Authorization = nil
		result.Err = ErrPlanningDenied
	}
	return result
}

func completeContext(request PlanningRequest) bool {
	return request.Repository != "" && request.VisibleStorage != "" && request.Audience != "" && request.Language != "" && request.Owner != ""
}

func validPolicy(policy LifecyclePolicy) bool {
	return policy == ApprovalRequired || policy == AutomaticAfterApprovedPlan
}

func validActions(actions []domain.PlannedAction) bool {
	for _, action := range actions {
		if action.Path == "" || filepath.IsAbs(action.Path) || filepath.Clean(action.Path) != action.Path || strings.HasPrefix(action.Path, ".."+string(filepath.Separator)) {
			return false
		}
		if action.Kind != domain.ActionCreate && action.Kind != domain.ActionUpdate && action.Kind != domain.ActionMove && action.Kind != domain.ActionDelete {
			return false
		}
	}
	return true
}

func batchApproved(request PlanningRequest) bool {
	for _, action := range request.Actions {
		if action.Structural || action.Kind == domain.ActionMove || action.Kind == domain.ActionDelete {
			return request.BatchApproved
		}
	}
	return request.Policy == AutomaticAfterApprovedPlan || request.BatchApproved
}

func existingDocuments(radiography gitadapter.Radiography) []ExistingDocumentTreatment {
	paths := make([]ExistingDocumentTreatment, 0, len(radiography.Documentation))
	for _, document := range radiography.Documentation {
		paths = append(paths, ExistingDocumentTreatment{Path: document.Path, Treatment: InPlaceInventory})
	}
	return paths
}

func (s PlanningService) now() time.Time {
	if s.Clock != nil {
		return s.Clock()
	}
	return time.Now().UTC()
}

func planningProvenance(request PlanningRequest, result PlanningResult, now time.Time) domain.Provenance {
	return domain.Provenance{
		Authority: domain.DeclaredLocalProvenance, DeclaredActor: request.Actor, Interaction: request.Interaction,
		Context: result.Radiography.Root, Operation: "plan", Request: request.Request, IdempotencyKey: request.IdempotencyKey,
		Time: now.Format(time.RFC3339Nano), Approval: "approved", Evidence: result.Radiography.Identity,
		Input: result.Plan.Revision, Result: result.Batch.ID, Versions: result.Plan.Revision + "/" + result.Policy.Revision,
	}
}

type CatalogImportRequest struct {
	Plan                                                  Plan
	Actor, Interaction, Request, IdempotencyKey, Evidence string
}

type CatalogImportResult struct {
	Entries    []domain.CatalogEntry
	Provenance []domain.Provenance
	Err        error
}

type CatalogAssessment struct {
	Entry                                   domain.CatalogEntry
	Evidence, Rationale                     string
	Related, Uncertain, Orphan, Conflicting bool
}

type CatalogAssessmentResult struct {
	Entry domain.CatalogEntry
	Audit domain.Audit
	Err   error
}

// CatalogService derives in-memory catalog evidence; it never writes visible documents.
type CatalogService struct{ Clock func() time.Time }

func (s CatalogService) Import(request CatalogImportRequest) CatalogImportResult {
	if request.Plan.State != domain.PlanApproved || !completeCatalogPlan(request.Plan) || request.Evidence == "" || request.Actor == "" || request.Interaction == "" || request.Request == "" || request.IdempotencyKey == "" {
		return CatalogImportResult{Err: ErrCatalogEvidenceRequired}
	}
	result := CatalogImportResult{}
	for _, document := range request.Plan.ExistingDocuments {
		if document.Path == "" || document.Treatment != InPlaceInventory {
			return CatalogImportResult{Err: ErrCatalogEvidenceRequired}
		}
		entry := domain.CatalogEntry{
			Path: document.Path, Purpose: "existing_documentation", Audience: request.Plan.Audience, Owner: request.Plan.Owner,
			RelatedAreas: []string{request.Plan.VisibleStorage}, State: domain.CatalogImported, Evidence: request.Evidence,
		}
		result.Entries = append(result.Entries, entry)
		result.Provenance = append(result.Provenance, domain.Provenance{
			Authority: domain.DeclaredLocalProvenance, DeclaredActor: request.Actor, Interaction: request.Interaction,
			Context: request.Plan.Revision, Operation: "catalog_import", Request: request.Request, IdempotencyKey: request.IdempotencyKey,
			Time: s.now().Format(time.RFC3339Nano), Approval: "approved", Evidence: request.Evidence,
			Input: document.Path, Result: string(domain.CatalogImported), Versions: request.Plan.Revision,
		})
	}
	return result
}

func completeCatalogPlan(plan Plan) bool {
	return plan.Revision != "" && plan.Audience != "" && plan.Owner != "" && plan.VisibleStorage != ""
}

func (s CatalogService) Assess(assessment CatalogAssessment) CatalogAssessmentResult {
	if assessment.Entry.Path == "" || assessment.Evidence == "" || assessment.Rationale == "" {
		return CatalogAssessmentResult{Err: ErrCatalogEvidenceRequired}
	}
	result := CatalogAssessmentResult{Entry: assessment.Entry, Audit: domain.Audit{State: domain.AuditReported, Evidence: assessment.Evidence, Rationale: assessment.Rationale}}
	switch {
	case assessment.Conflicting:
		result.Entry.State, result.Audit.Result = domain.CatalogUncertain, domain.AuditConflict
	case assessment.Orphan:
		result.Entry.State, result.Audit.Result = domain.CatalogOrphan, domain.AuditOrphan
		result.Entry.PendingActions = append(result.Entry.PendingActions, domain.PendingAction{State: domain.PendingOpen, Evidence: assessment.Evidence})
	case assessment.Uncertain:
		result.Entry.State, result.Audit.Result = domain.CatalogUncertain, domain.AuditReview
	case assessment.Related:
		result.Entry.State, result.Audit.Result = domain.CatalogStale, domain.AuditUpdate
	default:
		result.Audit.Result = domain.AuditNoAction
	}
	return result
}

func (s CatalogService) now() time.Time {
	if s.Clock != nil {
		return s.Clock()
	}
	return time.Now().UTC()
}
