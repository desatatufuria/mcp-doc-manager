package app

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	gitadapter "github.com/desatatufuria/mcp-doc-manager/internal/adapters/git"
	"github.com/desatatufuria/mcp-doc-manager/internal/domain"
)

func TestPlanningServiceBuildsBoundedPlanFromConfirmedRadiography(t *testing.T) {
	radiography := gitadapter.Radiography{
		Root:     "/repo",
		Identity: "sha256:radiography",
		Documentation: []gitadapter.DocumentationEvidence{
			{Path: "docs/guide.md", Classification: "uncertain", Evidence: "repository_evidence"},
		},
		Ownership: gitadapter.OwnershipUncertainty{Uncertain: true, Reason: "owner_confirmation_required"},
	}
	service := PlanningService{Radiography: fakeRadiography{report: radiography}, Clock: fixedPlanningClock}
	result := service.Plan(context.Background(), PlanningRequest{
		Repository: "/repo", Confirmed: true, VisibleStorage: "docs", Audience: "contributors", Language: "en", Owner: "docs-team",
		Policy: ApprovalRequired, PlanApproved: true, BatchApproved: true, PolicyApproved: true,
		Actions: []domain.PlannedAction{{Path: "docs/guide.md", Kind: domain.ActionUpdate}},
		Actor:   "local-user", Interaction: "mcp:plan", Request: "request-1", IdempotencyKey: "key-1",
	})
	if result.Err != nil {
		t.Fatal(result.Err)
	}
	if result.Plan.State != domain.PlanApproved || result.Plan.Radiography != radiography.Identity || result.Plan.VisibleStorage != "docs" || !reflect.DeepEqual(result.Plan.ExistingDocuments, []ExistingDocumentTreatment{{Path: "docs/guide.md", Treatment: InPlaceInventory}}) {
		t.Fatalf("plan = %#v", result.Plan)
	}
	if result.Batch.State != domain.BatchAuthorized || result.Authorization == nil || !reflect.DeepEqual(result.Authorization.Allowed, result.Plan.Actions) {
		t.Fatalf("batch/authorization = %#v %#v", result.Batch, result.Authorization)
	}
	if result.Provenance.Authority != domain.DeclaredLocalProvenance || result.Provenance.Validate() != nil {
		t.Fatalf("provenance = %#v", result.Provenance)
	}
}

func TestPlanningServiceTreatsExistingDocumentsInPlaceWithoutContentAction(t *testing.T) {
	radiography := gitadapter.Radiography{
		Root:     "/repo",
		Identity: "sha256:radiography",
		Documentation: []gitadapter.DocumentationEvidence{
			{Path: "docs/guide.md", Classification: "uncertain", Evidence: "repository_evidence"},
		},
	}
	service := PlanningService{Radiography: fakeRadiography{report: radiography}, Clock: fixedPlanningClock}
	result := service.Plan(context.Background(), PlanningRequest{
		Repository: "/repo", Confirmed: true, VisibleStorage: "docs", Audience: "contributors", Language: "en", Owner: "docs-team",
		Policy: ApprovalRequired, PlanApproved: true, BatchApproved: true, PolicyApproved: true,
		Actor: "local-user", Interaction: "mcp:plan", Request: "request-1", IdempotencyKey: "key-1",
	})
	if result.Err != nil {
		t.Fatal(result.Err)
	}
	want := []ExistingDocumentTreatment{{Path: "docs/guide.md", Treatment: InPlaceInventory}}
	if !reflect.DeepEqual(result.Plan.ExistingDocuments, want) || len(result.Plan.Actions) != 0 {
		t.Fatalf("plan = %#v", result.Plan)
	}
}

func TestPlanningServiceDeniesUnconfirmedOrUnapprovedPlans(t *testing.T) {
	service := PlanningService{Radiography: fakeRadiography{report: gitadapter.Radiography{Root: "/repo", Identity: "sha256:radiography"}}, Clock: fixedPlanningClock}
	for _, request := range []PlanningRequest{
		{Repository: "/repo", Policy: ApprovalRequired},
		{Repository: "/repo", Confirmed: true, VisibleStorage: "docs", Audience: "contributors", Language: "en", Owner: "docs-team", Policy: ApprovalRequired, PlanApproved: true, PolicyApproved: true, Actions: []domain.PlannedAction{{Path: "docs/guide.md", Kind: domain.ActionUpdate}}},
	} {
		result := service.Plan(context.Background(), request)
		if !errors.Is(result.Err, ErrPlanningDenied) || result.Authorization != nil {
			t.Fatalf("result = %#v", result)
		}
	}
}

func TestPlanningServiceAppliesPolicyAndActionBounds(t *testing.T) {
	base := PlanningRequest{
		Repository: "/repo", Confirmed: true, VisibleStorage: "docs", Audience: "contributors", Language: "en", Owner: "docs-team",
		Policy: AutomaticAfterApprovedPlan, PlanApproved: true, PolicyApproved: true,
		Actor: "local-user", Interaction: "mcp:plan", Request: "request-2", IdempotencyKey: "key-2",
	}
	service := PlanningService{Radiography: fakeRadiography{report: gitadapter.Radiography{Root: "/repo", Identity: "sha256:radiography"}}, Clock: fixedPlanningClock}
	for _, test := range []struct {
		name    string
		actions []domain.PlannedAction
		approve bool
		denied  bool
	}{
		{"automatic in-plan content actions", []domain.PlannedAction{{Path: "docs/guide.md", Kind: domain.ActionUpdate}, {Path: "docs/new.md", Kind: domain.ActionCreate}}, false, false},
		{"unapproved structural action", []domain.PlannedAction{{Path: "docs/guide.md", Kind: domain.ActionMove, Structural: true}}, false, true},
		{"approved structural action", []domain.PlannedAction{{Path: "docs/guide.md", Kind: domain.ActionMove, Structural: true}}, true, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := base
			request.Actions, request.BatchApproved = test.actions, test.approve
			result := service.Plan(context.Background(), request)
			if test.denied {
				if !errors.Is(result.Err, ErrPlanningDenied) || result.Authorization != nil {
					t.Fatalf("result = %#v", result)
				}
				return
			}
			if result.Err != nil || result.Authorization == nil || !reflect.DeepEqual(result.Authorization.Allowed, test.actions) {
				t.Fatalf("result = %#v", result)
			}
		})
	}
}

func TestCatalogServiceImportsApprovedInPlaceDocumentsWithLocalProvenance(t *testing.T) {
	service := CatalogService{Clock: fixedPlanningClock}
	plan := Plan{
		State: domain.PlanApproved, Revision: "sha256:plan", VisibleStorage: "docs", Audience: "contributors", Owner: "docs-team",
		ExistingDocuments: []ExistingDocumentTreatment{{Path: "docs/guide.md", Treatment: InPlaceInventory}},
	}
	result := service.Import(CatalogImportRequest{
		Plan: plan, Actor: "local-user", Interaction: "mcp:import-catalog", Request: "request-import", IdempotencyKey: "key-import", Evidence: "sha256:repository",
	})
	if result.Err != nil {
		t.Fatal(result.Err)
	}
	want := domain.CatalogEntry{
		Path: "docs/guide.md", Purpose: "existing_documentation", Audience: "contributors", Owner: "docs-team", RelatedAreas: []string{"docs"},
		State: domain.CatalogImported, Evidence: "sha256:repository",
	}
	if !reflect.DeepEqual(result.Entries, []domain.CatalogEntry{want}) {
		t.Fatalf("entries = %#v", result.Entries)
	}
	if len(result.Provenance) != 1 || result.Provenance[0].Operation != "catalog_import" || result.Provenance[0].Authority != domain.DeclaredLocalProvenance || result.Provenance[0].Validate() != nil {
		t.Fatalf("provenance = %#v", result.Provenance)
	}
	if result.Entries[0].LastVerification != "" {
		t.Fatalf("last verification = %q", result.Entries[0].LastVerification)
	}
}

func TestCatalogServiceRejectsImportsMissingPlanProvenance(t *testing.T) {
	service := CatalogService{Clock: fixedPlanningClock}
	plan := Plan{
		State: domain.PlanApproved, Revision: "sha256:plan", VisibleStorage: "docs", Audience: "contributors", Owner: "docs-team",
		ExistingDocuments: []ExistingDocumentTreatment{{Path: "docs/guide.md", Treatment: InPlaceInventory}},
	}
	for _, field := range []string{"Revision", "Audience", "Owner", "VisibleStorage"} {
		t.Run(field, func(t *testing.T) {
			incomplete := plan
			switch field {
			case "Revision":
				incomplete.Revision = ""
			case "Audience":
				incomplete.Audience = ""
			case "Owner":
				incomplete.Owner = ""
			case "VisibleStorage":
				incomplete.VisibleStorage = ""
			}
			result := service.Import(CatalogImportRequest{
				Plan: incomplete, Actor: "local-user", Interaction: "mcp:import-catalog", Request: "request-import", IdempotencyKey: "key-import", Evidence: "sha256:repository",
			})
			if !errors.Is(result.Err, ErrCatalogEvidenceRequired) || len(result.Entries) != 0 || len(result.Provenance) != 0 {
				t.Fatalf("result = %#v", result)
			}
		})
	}
}

func TestCatalogServiceAssessesStaleUncertainAndOrphanEvidenceWithoutMutation(t *testing.T) {
	service := CatalogService{}
	entry := domain.CatalogEntry{Path: "docs/guide.md", State: domain.CatalogActive, Evidence: "sha256:old"}
	for _, tt := range []struct {
		name    string
		input   CatalogAssessment
		state   domain.CatalogState
		result  domain.AuditResult
		pending bool
	}{
		{"stale", CatalogAssessment{Entry: entry, Evidence: "sha256:new", Rationale: "repository evidence changed", Related: true}, domain.CatalogStale, domain.AuditUpdate, false},
		{"uncertain", CatalogAssessment{Entry: entry, Evidence: "sha256:new", Rationale: "mapping is incomplete", Uncertain: true}, domain.CatalogUncertain, domain.AuditReview, false},
		{"orphan", CatalogAssessment{Entry: entry, Evidence: "sha256:new", Rationale: "path is absent", Orphan: true}, domain.CatalogOrphan, domain.AuditOrphan, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := service.Assess(tt.input)
			if got.Err != nil || got.Entry.State != tt.state || got.Audit.Result != tt.result || got.Audit.Evidence != tt.input.Evidence || got.Audit.Rationale != tt.input.Rationale {
				t.Fatalf("assessment = %#v", got)
			}
			if tt.pending && (len(got.Entry.PendingActions) != 1 || got.Entry.PendingActions[0].State != domain.PendingOpen || got.Entry.PendingActions[0].Evidence != tt.input.Evidence) {
				t.Fatalf("pending actions = %#v", got.Entry.PendingActions)
			}
		})
	}
}

func TestCatalogServiceReportsOnlyEvidencedAuditTreatments(t *testing.T) {
	service := CatalogService{}
	entry := domain.CatalogEntry{Path: "docs/guide.md", State: domain.CatalogActive, Evidence: "sha256:old"}
	for _, tt := range []struct {
		name  string
		input CatalogAssessment
		want  domain.AuditResult
	}{
		{"update", CatalogAssessment{Entry: entry, Evidence: "sha256:new", Rationale: "related code changed", Related: true}, domain.AuditUpdate},
		{"review", CatalogAssessment{Entry: entry, Evidence: "sha256:new", Rationale: "mapping is uncertain", Uncertain: true}, domain.AuditReview},
		{"orphan", CatalogAssessment{Entry: entry, Evidence: "sha256:new", Rationale: "document path is absent", Orphan: true}, domain.AuditOrphan},
		{"conflict", CatalogAssessment{Entry: entry, Evidence: "sha256:new", Rationale: "evidence conflicts", Conflicting: true}, domain.AuditConflict},
		{"no action", CatalogAssessment{Entry: entry, Evidence: "sha256:old", Rationale: "no related change"}, domain.AuditNoAction},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := service.Assess(tt.input)
			if got.Err != nil || got.Audit.State != domain.AuditReported || got.Audit.Result != tt.want || got.Audit.Evidence != tt.input.Evidence || got.Audit.Rationale != tt.input.Rationale {
				t.Fatalf("assessment = %#v", got)
			}
		})
	}
	if got := service.Assess(CatalogAssessment{Entry: entry}); !errors.Is(got.Err, ErrCatalogEvidenceRequired) {
		t.Fatalf("missing evidence = %#v", got)
	}
}

type fakeRadiography struct {
	report gitadapter.Radiography
	err    error
}

func (f fakeRadiography) Radiograph(context.Context, string) (gitadapter.Radiography, error) {
	return f.report, f.err
}

func fixedPlanningClock() time.Time { return time.Date(2026, time.August, 3, 0, 0, 0, 0, time.UTC) }
