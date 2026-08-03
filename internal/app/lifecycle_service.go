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

var ErrPlanningDenied = errors.New("planning_denied")

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
