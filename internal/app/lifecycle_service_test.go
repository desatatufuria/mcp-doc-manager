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

type fakeRadiography struct {
	report gitadapter.Radiography
	err    error
}

func (f fakeRadiography) Radiograph(context.Context, string) (gitadapter.Radiography, error) {
	return f.report, f.err
}

func fixedPlanningClock() time.Time { return time.Date(2026, time.August, 3, 0, 0, 0, 0, time.UTC) }
