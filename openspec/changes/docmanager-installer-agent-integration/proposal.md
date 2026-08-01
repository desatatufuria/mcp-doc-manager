# Proposal: Docmanager Installer and Agent Integration

## Intent

Make `github.com/desatatufuria/mcp-doc-manager` installable and safely configurable without changing MCP tools or ledger semantics. Correct its Go module/import identity before release.

## Scope

### In Scope
- Linux/macOS amd64/arm64: first-release signed manifests; install, upgrade, status, rollback, and doctor.
- Headless `release`, `agent`, and `workspace` hierarchy: dry-run, JSON, stable errors, locks, ownership, backups, atomic writes, unconfigure, and one-major-version aliases.
- Managed MCP/guidance configuration and fixtures for OpenCode, Codex, Claude Code, GitHub Copilot, and Pi.
- Release-trust policy: trusted key IDs, signed-manifest expiry, rotation overlap, compromise revocation, and offline/manual recovery.
- Services for a future TUI and optional Gentle AI community-tool orchestration analogous to CodeGraph, without runtime or ownership dependency.

### Out of Scope
- TUI, Windows distribution, unmanaged agent configuration, or automatic repository hooks.
- Changes to read-only MCP tools, receipt behavior, or repository-local ledger location.

## Capabilities

### New Capabilities
- `project-identity-migration`: Canonical Go module/import identity.
- `release-management`: Trusted distribution, updates, recovery, and status.
- `agent-integration`: Safe managed MCP/guidance lifecycle and health.
- `headless-orchestration-contract`: Stable plans/results, JSON, dry-run, and errors.
- `workspace-wiring`: Workspace lifecycle and compatible aliases; hooks are opt-in.

### Modified Capabilities
None; `openspec/specs/` has no existing capabilities.

## Approach

Use services behind `release`, `agent`, and `workspace`; preserve `install --target`, `doctor --target`, and `uninstall --target` as deprecated human-output aliases through the next major version. Isolate release, filesystem, and agent adapters; configure only namespaced owned entries and restore exact snapshots. Reject unknown or externally changed state with versioned fixtures.

## Affected Areas

| Area | Impact | Description |
|---|---|---|
| `go.mod`, Go imports | Modified | Canonical module identity. |
| `cmd/docmanager/`, `internal/app/` | Modified | Headless services and CLI rendering. |
| `internal/adapters/{agent,release,filesystem}/` | New | Safe integration and release boundaries. |
| `assets/`, `.github/workflows/` | Modified/New | Managed guidance and signed release assets. |

## Risks

| Risk | Likelihood | Mitigation |
|---|---|---|
| Agent schemas drift | High | Versioned fixtures; reject unknown shapes. |
| Trust-key compromise/rotation | Med | Key IDs, expiry, overlap, revocation, recovery policy. |
| Unsafe writes or binary replacement | Med | Ownership checks, locks, backups, atomic rollback. |

## Rollback Plan

Withdraw publication; retain owned binary/state until health succeeds; restore snapshots. Aliases preserve workspace wiring.

## Dependencies

- Signing keys, manifest publication, and verified agent/version fixtures.
- Phased/chained implementation is required because the forecast exceeds the 800-line review budget.

## Success Criteria

- [ ] Platforms install verified artifacts and safely upgrade, doctor, and roll back.
- [ ] Each named agent safely supports detect/configure/status/unconfigure with fixtures and stable JSON/dry-run results.
- [ ] Existing MCP and repository ledger behavior remains compatible; hooks are never silently enabled.
