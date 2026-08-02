package domain

import (
	"errors"
	"reflect"
)

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
	return (from == RadiographyDraft && (to == RadiographyConfirmed || to == RadiographyRejected)) || (from == PlanProposed && (to == PlanApproved || to == PlanRefused)) || (from == PlanApproved && to == PlanDrifted) || (from == BatchProposed && (to == BatchAuthorized || to == BatchRejected)) || (from == BatchAuthorized && to == BatchReported) || (from == BatchReported && to == BatchVerified)
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
	Active, Completed                     bool
}

func (a Authorization) Validate(current Authorization) error {
	if !a.Active || a.Completed || !current.Active || current.Completed || a.BatchID != current.BatchID || a.PlanRevision != current.PlanRevision || a.PolicyRevision != current.PolicyRevision || a.Scope != current.Scope || a.Baseline != current.Baseline || a.Evidence != current.Evidence || !reflect.DeepEqual(a.Allowed, current.Allowed) {
		return ErrAuthorizationInvalidated
	}
	return nil
}

type ProvenanceAuthority string

const DeclaredLocalProvenance ProvenanceAuthority = "declared_local_provenance"

type Provenance struct {
	Authority                                                                                                                  ProvenanceAuthority
	DeclaredActor, Interaction, Context, Operation, Request, IdempotencyKey, Time, Approval, Evidence, Input, Result, Versions string
}

func (p Provenance) Validate() error {
	if p.Authority != DeclaredLocalProvenance || p.DeclaredActor == "" || p.Interaction == "" || p.Context == "" || p.Operation == "" || p.Request == "" || p.IdempotencyKey == "" || p.Time == "" || p.Approval == "" || p.Evidence == "" || p.Input == "" || p.Result == "" || p.Versions == "" {
		return ErrLifecycle
	}
	return nil
}

type CatalogState string

const (
	CatalogImported  CatalogState = "imported"
	CatalogActive    CatalogState = "active"
	CatalogStale     CatalogState = "stale"
	CatalogUncertain CatalogState = "uncertain"
	CatalogOrphan    CatalogState = "orphan"
	CatalogArchived  CatalogState = "archived"
)

type AuditState string

const (
	AuditRunning  AuditState = "running"
	AuditReported AuditState = "reported"
)

type AuditResult string

const (
	AuditUpdate   AuditResult = "update"
	AuditReview   AuditResult = "review"
	AuditOrphan   AuditResult = "orphan"
	AuditConflict AuditResult = "conflict"
	AuditNoAction AuditResult = "no_action"
)

type PendingActionState string

const (
	PendingOpen      PendingActionState = "open"
	PendingApproved  PendingActionState = "approved"
	PendingRefused   PendingActionState = "refused"
	PendingCompleted PendingActionState = "completed"
	PendingCancelled PendingActionState = "cancelled"
)

type VerificationState string

const (
	VerificationPending  VerificationState = "pending"
	VerificationVerified VerificationState = "verified"
	VerificationMismatch VerificationState = "mismatch"
	VerificationStale    VerificationState = "stale"
)

type CatalogEntry struct {
	Path, Purpose, Audience, Owner, Evidence string
	RelatedAreas                             []string
	State                                    CatalogState
	LastVerification                         string
	PendingActions                           []PendingAction
}
type Audit struct {
	State               AuditState
	Result              AuditResult
	Evidence, Rationale string
}
type PendingAction struct {
	State    PendingActionState
	Evidence string
}
type Verification struct {
	State    VerificationState
	Evidence string
}
