// Package agent provides safe, fixture-pinned agent configuration adapters.
package agent

import (
	"context"
	"errors"
	"time"
)

var (
	ErrLocked              = errors.New("agent configuration is locked")
	ErrDrift               = errors.New("managed agent configuration drifted")
	ErrOwnership           = errors.New("agent configuration is not owned")
	ErrMalformedConfig     = errors.New("malformed agent configuration")
	ErrUnsupportedConfig   = errors.New("unsupported agent configuration")
	ErrUnsafeRoute         = errors.New("unsafe agent configuration route")
	ErrProbeFailed         = errors.New("agent probe failed")
	ErrPrerequisiteMissing = errors.New("agent prerequisite missing")
)

const fixtureProvenanceV1 = "docmanager/agent-fixture/v1"

// Status deliberately keeps discovery, support, managed configuration, and health independent.
type Status struct{ Installed, Supported, Configured, Healthy bool }

type OpenCodeOptions struct {
	Root         string
	Binary       string
	Installed    func() bool
	Run          func(ctx context.Context, argv ...string) error
	ProbeTimeout time.Duration
	Provenance   string
}
