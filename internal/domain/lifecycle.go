package domain

import "errors"

var (
	ErrLifecycle                = errors.New("lifecycle_failure")
	ErrAuthorizationInvalidated = errors.New("authorization_invalidated")
	ErrIdempotencyConflict      = errors.New("idempotency_conflict")
)

type LifecycleState string

const (
	RadiographyDraft     LifecycleState = "radiography_draft"
	RadiographyConfirmed LifecycleState = "radiography_confirmed"
	RadiographyRejected  LifecycleState = "radiography_rejected"
	PlanProposed         LifecycleState = "plan_proposed"
	PlanApproved         LifecycleState = "plan_approved"
	PlanRefused          LifecycleState = "plan_refused"
	PlanDrifted          LifecycleState = "plan_drifted"
	BatchProposed        LifecycleState = "batch_proposed"
	BatchAuthorized      LifecycleState = "batch_authorized"
	BatchReported        LifecycleState = "batch_reported"
	BatchVerified        LifecycleState = "batch_verified"
	BatchRejected        LifecycleState = "batch_rejected"
)

func CanTransition(from, to LifecycleState) bool {
	return (from == RadiographyDraft && (to == RadiographyConfirmed || to == RadiographyRejected)) ||
		(from == PlanProposed && (to == PlanApproved || to == PlanRefused)) ||
		(from == PlanApproved && to == PlanDrifted) ||
		(from == BatchProposed && (to == BatchAuthorized || to == BatchRejected)) ||
		(from == BatchAuthorized && (to == BatchReported || to == BatchVerified)) || (from == BatchReported && to == BatchVerified)
}

type ActionKind string

const (
	ActionCreate ActionKind = "create"
	ActionUpdate ActionKind = "update"
	ActionMove   ActionKind = "move"
	ActionDelete ActionKind = "delete"
)

type PlannedAction struct {
	Path       string
	Kind       ActionKind
	Structural bool
}
type Authorization struct {
	BatchID, PlanRevision, PolicyRevision string
	Scope                                 Scope
	Baseline, Evidence                    string
	Allowed                               []PlannedAction
	Active                                bool
}

func (a Authorization) Validate(current Authorization) error {
	if !a.Active || !current.Active || a.PlanRevision != current.PlanRevision || a.PolicyRevision != current.PolicyRevision || a.Scope != current.Scope || a.Baseline != current.Baseline || a.Evidence != current.Evidence {
		return ErrAuthorizationInvalidated
	}
	return nil
}

type Provenance struct{ DeclaredActor, Interaction, Context string }

func (p Provenance) Validate() error {
	if p.DeclaredActor == "" || p.Interaction == "" || p.Context == "" {
		return ErrLifecycle
	}
	return nil
}

type CatalogEntry struct {
	Path, Purpose, Audience, Owner, Evidence string
	RelatedAreas                             []string
	State                                    LifecycleState
	LastVerification                         string
	PendingActions                           []string
}
type Audit struct {
	State               LifecycleState
	Evidence, Rationale string
}
type Verification struct {
	State    LifecycleState
	Evidence string
}
