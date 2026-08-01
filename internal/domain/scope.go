package domain

import (
	"errors"
	"strings"
)

type ScopeKind string

const (
	ScopeRange    ScopeKind = "range"
	ScopeInitial  ScopeKind = "initial"
	ScopeStaged   ScopeKind = "staged"
	ScopeWorktree ScopeKind = "worktree"
)

var (
	ErrInvalidScope       = errors.New("invalid_scope")
	ErrEmptyScope         = errors.New("empty_scope")
	ErrNotRepository      = errors.New("not_repository")
	ErrOutsideRepository  = errors.New("outside_repository")
	ErrGitUnavailable     = errors.New("git_unavailable")
	ErrOutputLimit        = errors.New("bounded_output")
	ErrInvalidRevision    = errors.New("invalid_revision")
	ErrContentRead        = errors.New("content_read_failed")
	ErrUnsupportedRequest = errors.New("unsupported_request")
	ErrLedgerFailure      = errors.New("ledger_failure")
	ErrReceiptMismatch    = errors.New("receipt_mismatch")
	ErrInvalidTarget      = errors.New("invalid_target")
)

type Scope struct {
	Kind  ScopeKind `json:"kind"`
	Range string    `json:"range"`
}

func (s Scope) Validate() error {
	switch s.Kind {
	case ScopeRange, ScopeInitial:
		if strings.TrimSpace(s.Range) == "" {
			return ErrInvalidScope
		}
	case ScopeStaged, ScopeWorktree:
		if s.Range != "" {
			return ErrInvalidScope
		}
	default:
		return ErrUnsupportedRequest
	}
	return nil
}
