package domain

import (
	"errors"
	"testing"
)

func TestLifecycleTransitionsAndAuthorizationBounds(t *testing.T) {
	for _, tt := range []struct {
		from, to LifecycleState
		want     bool
	}{
		{RadiographyDraft, RadiographyConfirmed, true}, {PlanProposed, PlanApproved, true},
		{BatchAuthorized, BatchReported, true}, {BatchReported, BatchVerified, true},
		{BatchAuthorized, BatchVerified, false}, {PlanApproved, PlanProposed, false},
	} {
		t.Run(string(tt.from)+"_"+string(tt.to), func(t *testing.T) {
			if got := CanTransition(tt.from, tt.to); got != tt.want {
				t.Fatalf("CanTransition() = %v", got)
			}
		})
	}
	base := Authorization{BatchID: "batch-1", Active: true, PlanRevision: "plan-1", PolicyRevision: "policy-1", Scope: Scope{Kind: ScopeRange, Range: "a..b"}, Baseline: "base-1", Evidence: "evidence-1", Allowed: []PlannedAction{{Path: "README.md", Kind: ActionUpdate}, {Path: "docs/a.md", Kind: ActionCreate}}}
	for _, tt := range []struct {
		name   string
		actual Authorization
	}{
		{"batch", bind(base, func(a *Authorization) { a.BatchID = "batch-2" })},
		{"allowed ordered identity", bind(base, func(a *Authorization) { a.Allowed = []PlannedAction{a.Allowed[1], a.Allowed[0]} })},
		{"plan", bind(base, func(a *Authorization) { a.PlanRevision = "plan-2" })},
		{"policy", bind(base, func(a *Authorization) { a.PolicyRevision = "policy-2" })},
		{"scope", bind(base, func(a *Authorization) { a.Scope = Scope{Kind: ScopeStaged} })},
		{"baseline", bind(base, func(a *Authorization) { a.Baseline = "base-2" })},
		{"evidence", bind(base, func(a *Authorization) { a.Evidence = "evidence-2" })},
		{"inactive", bind(base, func(a *Authorization) { a.Active = false })},
		{"completed", bind(base, func(a *Authorization) { a.Completed = true })},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if err := base.Validate(tt.actual); !errors.Is(err, ErrAuthorizationInvalidated) {
				t.Fatalf("Validate() = %v", err)
			}
		})
	}
}

func bind(a Authorization, mutate func(*Authorization)) Authorization {
	a.Allowed = append([]PlannedAction(nil), a.Allowed...)
	mutate(&a)
	return a
}

func TestLifecycleProvenanceDeclaresLocalContextWithoutAuthenticationClaim(t *testing.T) {
	p := Provenance{Authority: DeclaredLocalProvenance, DeclaredActor: "local-user", Interaction: "mcp:approve-plan", Context: "request-7", Operation: "approve-plan", Request: "request-7", IdempotencyKey: "key-7", Time: "2026-08-02T00:00:00Z", Approval: "approved", Evidence: "evidence-7", Input: "input-7", Result: "result-7", Versions: "plan-1/policy-1"}
	if err := p.Validate(); err != nil {
		t.Fatal(err)
	}
	if got := p.Authority; got != DeclaredLocalProvenance {
		t.Fatalf("authority = %q, want declared local provenance", got)
	}
	catalog := CatalogEntry{State: CatalogArchived}
	audit := Audit{State: AuditReported, Result: AuditConflict}
	pending := PendingAction{State: PendingCompleted}
	verification := Verification{State: VerificationMismatch}
	if catalog.State != CatalogArchived || audit.State != AuditReported || audit.Result != AuditConflict || pending.State != PendingCompleted || verification.State != VerificationMismatch {
		t.Fatal("typed state was not retained")
	}
}

func TestLifecycleProvenanceRequiresEveryField(t *testing.T) {
	p := Provenance{Authority: DeclaredLocalProvenance, DeclaredActor: "actor", Interaction: "interaction", Context: "context", Operation: "operation", Request: "request", IdempotencyKey: "key", Time: "time", Approval: "approval", Evidence: "evidence", Input: "input", Result: "result", Versions: "versions"}
	for name, clear := range map[string]func(*Provenance){
		"authority": func(p *Provenance) { p.Authority = "" }, "actor": func(p *Provenance) { p.DeclaredActor = "" }, "interaction": func(p *Provenance) { p.Interaction = "" }, "context": func(p *Provenance) { p.Context = "" }, "operation": func(p *Provenance) { p.Operation = "" }, "request": func(p *Provenance) { p.Request = "" }, "key": func(p *Provenance) { p.IdempotencyKey = "" }, "time": func(p *Provenance) { p.Time = "" }, "approval": func(p *Provenance) { p.Approval = "" }, "evidence": func(p *Provenance) { p.Evidence = "" }, "input": func(p *Provenance) { p.Input = "" }, "result": func(p *Provenance) { p.Result = "" }, "versions": func(p *Provenance) { p.Versions = "" },
	} {
		t.Run(name, func(t *testing.T) {
			bad := p
			clear(&bad)
			if !errors.Is(bad.Validate(), ErrLifecycle) {
				t.Fatalf("Validate() = %v", bad.Validate())
			}
		})
	}
}

func TestIdempotencyReplayStateContract(t *testing.T) {
	if IdempotencyAvailable != "available" {
		t.Fatalf("available replay state = %q", IdempotencyAvailable)
	}
	if IdempotencyLegacyUnavailable != "legacy_unavailable" {
		t.Fatalf("legacy-unavailable replay state = %q", IdempotencyLegacyUnavailable)
	}
	if ErrLegacyIdempotencyReplayUnavailable.Error() != "legacy_idempotency_replay_unavailable" {
		t.Fatalf("legacy replay error = %q", ErrLegacyIdempotencyReplayUnavailable)
	}
}
