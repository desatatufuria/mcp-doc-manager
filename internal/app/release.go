package app

import (
	"context"
	"errors"

	releaseadapter "github.com/desatatufuria/mcp-doc-manager/internal/adapters/release"
)

// ReleaseService adapts the closed app request/result contract to an already
// extracted, manifest-authorized binary. Manifest retrieval remains a caller
// concern until the release CLI wiring work unit.
type ReleaseService struct{ Lifecycle releaseadapter.Lifecycle }

func (s ReleaseService) Execute(ctx context.Context, request Request) Result {
	if stable := ValidateRequest(request); stable != nil {
		return failure(request, stable)
	}
	if request.DryRun {
		return Result{Operation: request.Operation, Target: request.InstallDir, Outcome: OutcomePlanned, Changes: []Change{}}
	}
	var (
		status releaseadapter.LifecycleStatus
		err    error
	)
	switch request.Operation {
	case OperationReleaseStatus:
		status, err = s.Lifecycle.Status(request.InstallDir)
	case OperationReleaseDoctor:
		status, err = s.Lifecycle.Doctor(ctx, request.InstallDir)
	case OperationReleaseRollback:
		status, err = s.Lifecycle.Rollback(ctx, request.InstallDir)
	default:
		return failure(request, &StableError{Code: "unsupported_operation", Classification: "unsupported_operation", Detail: "release replacement requires a verified extracted binary"})
	}
	if err != nil {
		return failure(request, stableReleaseError(err))
	}
	outcome := OutcomeSuccess
	if request.Operation == OperationReleaseStatus || request.Operation == OperationReleaseDoctor {
		outcome = OutcomeStatus
	}
	return Result{Operation: request.Operation, Target: request.InstallDir, Outcome: outcome, Changes: []Change{}, Status: &Status{Release: &ReleaseStatus{Installed: status.Installed, Healthy: status.Healthy, Version: status.Version, Platform: status.Platform}}}
}

func stableReleaseError(err error) *StableError {
	code := "ownership"
	switch {
	case errors.Is(err, releaseadapter.ErrUnsupportedPlatform):
		code = "unsupported_platform"
	case errors.Is(err, releaseadapter.ErrManifestVerification), errors.Is(err, releaseadapter.ErrRecoveryRequired):
		code = "manifest_verification"
	case errors.Is(err, releaseadapter.ErrProbeFailed):
		code = "probe_failed"
	case errors.Is(err, releaseadapter.ErrLocked):
		code = "locked"
	case errors.Is(err, releaseadapter.ErrRollback):
		code = "rollback_failed"
	}
	return &StableError{Code: code, Classification: code, Detail: err.Error()}
}
