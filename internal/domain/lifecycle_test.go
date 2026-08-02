package domain

import (
	"errors"
	"strings"
	"testing"
)

func TestLifecycleTransitionsAndAuthorizationBounds(t *testing.T) {
	for _, tt := range []struct {
		from, to LifecycleState
		want     bool
	}{
		{RadiographyDraft, RadiographyConfirmed, true}, {PlanProposed, PlanApproved, true},
		{BatchAuthorized, BatchVerified, true}, {PlanApproved, PlanProposed, false},
	} {
		t.Run(string(tt.from)+"_"+string(tt.to), func(t *testing.T) {
			if got := CanTransition(tt.from, tt.to); got != tt.want {
				t.Fatalf("CanTransition() = %v", got)
			}
		})
	}
	base := Authorization{Active: true, PlanRevision: "plan-1", PolicyRevision: "policy-1", Scope: Scope{Kind: ScopeRange, Range: "a..b"}, Baseline: "base-1", Evidence: "evidence-1"}
	for _, tt := range []struct {
		name   string
		actual Authorization
	}{
		{"completion", Authorization{}}, {"plan", Authorization{Active: true, PlanRevision: "plan-2", PolicyRevision: "policy-1", Scope: base.Scope, Baseline: "base-1", Evidence: "evidence-1"}},
		{"policy", Authorization{Active: true, PlanRevision: "plan-1", PolicyRevision: "policy-2", Scope: base.Scope, Baseline: "base-1", Evidence: "evidence-1"}},
		{"scope", Authorization{Active: true, PlanRevision: "plan-1", PolicyRevision: "policy-1", Scope: Scope{Kind: ScopeStaged}, Baseline: "base-1", Evidence: "evidence-1"}},
		{"baseline", Authorization{Active: true, PlanRevision: "plan-1", PolicyRevision: "policy-1", Scope: base.Scope, Baseline: "base-2", Evidence: "evidence-1"}},
		{"evidence", Authorization{Active: true, PlanRevision: "plan-1", PolicyRevision: "policy-1", Scope: base.Scope, Baseline: "base-1", Evidence: "evidence-2"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if err := base.Validate(tt.actual); !errors.Is(err, ErrAuthorizationInvalidated) {
				t.Fatalf("Validate() = %v", err)
			}
		})
	}
}

func TestLifecycleProvenanceDeclaresLocalContextWithoutAuthenticationClaim(t *testing.T) {
	p := Provenance{DeclaredActor: "local-user", Interaction: "mcp:approve-plan", Context: "request-7"}
	if err := p.Validate(); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.ToLower(p.DeclaredActor+p.Interaction+p.Context), "auth") {
		t.Fatal("provenance claimed authentication")
	}
}
