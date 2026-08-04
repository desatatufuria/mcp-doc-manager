// Package agent provides safe, fixture-pinned agent configuration adapters.
package agent

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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
	Guidance     GuidanceIdentity
}

type GuidanceIdentity struct{ Path, Version, Digest string }

func NewGuidanceIdentity(path, version string, content []byte) GuidanceIdentity {
	digest := sha256.Sum256(content)
	return GuidanceIdentity{Path: path, Version: version, Digest: hex.EncodeToString(digest[:])}
}
