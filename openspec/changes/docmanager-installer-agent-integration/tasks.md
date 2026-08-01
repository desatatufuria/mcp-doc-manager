# Tasks: Docmanager Installer and Agent Integration

## Review Workload Forecast

| Field | Value |
|---|---|
| Estimated changed lines | 2,400–3,300 authored; fixtures/assets excluded |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Suggested split | Contract → workspace → trust → lifecycle → packaging → agents → acceptance; tracker last |
| Delivery strategy | ask-on-risk |
| Chain strategy | feature-branch-chain |

Decision needed before apply: No
Chained PRs recommended: Yes
Chain strategy: feature-branch-chain
400-line budget risk: High

### Suggested Work Units

| Unit | Goal | Work branch/base | Focused test command | Runtime harness | Rollback boundary |
|---|---|---|---|---|---|
| 1 | Contract | tracker `feature/docmanager-installer-agent-integration` | `go test ./internal/app ./cmd/docmanager` | temp Git: `workspace status --json` | `go.mod`, contract, CLI |
| 2 | Workspace | tracker after U1 | `go test ./internal/adapters/filesystem ./internal/app` | temp Git install/status/uninstall | filesystem, workspace, hooks |
| 3 | Release trust ≤800 | `feature/docmanager-installer-release`, tracker after U2 | `go test ./internal/adapters/release -run 'Trust|Extract'` | `httptest` manifest/archive | trust/download/extraction only |
| 4 | Release lifecycle ≤800 | tracker after U3 | `go test ./internal/adapters/release ./internal/app -run 'Lifecycle|Probe'` | temp owned binary upgrade/rollback | lifecycle/probe only |
| 5 | Release packaging ≤800 | tracker after U4 | `go test ./internal/adapters/release -run Bootstrap` | archive/bootstrap fixture | script/assets/workflow only |
| 6 | Agents | tracker after U5 | `go test ./internal/adapters/agent ./internal/app` | isolated XDG/HOME/VS Code | adapters, fixtures, assets |
| 7 | Acceptance | tracker after U6 | `go test ./...` | build + temp CLI matrix | acceptance tests and docs |

Each branch starts from the tracker after its predecessor merges; only the tracker merges to main.

## Phase 1: Identity and Typed Contract

- [x] 1.1 RED: table-test `internal/app/{identity,contract}_test.go`: canonical/legacy; MCP/receipt/ledger unchanged; closed-operation invalid input pre-write; headless, JSON union, dry-run no-write.
- [x] 1.2 GREEN: migrate `go.mod`/imports; add `internal/app/{identity,contract}.go` typed context results, unions, errors, ports; preserve `internal/adapters/{mcp,sqlite}`.
- [x] 1.3 RED: test hierarchy, JSON, and deprecated human-only aliases in `cmd/docmanager/main_test.go`.
- [x] 1.4 GREEN: route `release|agent|workspace` and `install|doctor|uninstall --target` aliases in `cmd/docmanager/main.go`.

## Phase 2: Transactional Workspace and Release

- [x] 2.1 RED: `internal/adapters/filesystem/transaction_test.go`: lock, backup digest/mode, fsync-rename, drift, ownership, symlink, route escape; no mutation.
- [x] 2.2 GREEN: add `internal/adapters/filesystem/transaction.go`, `internal/app/workspace.go`, external state, and workspace install/uninstall/status/doctor.
- [x] 2.3 RED: `internal/app/workspace_test.go`: default absent, opt-in, drift refusal, non-root Git selectors.
- [x] 2.4 GREEN: implement explicit hooks in `internal/app/workspace.go` and `cmd/docmanager/main.go`.
- [x] 2.5 RED: `internal/adapters/release/{trust,extract}_test.go`: signature/platform/expiry/revocation/rotation/recovery; HTTPS host/redirect/size/digest; traversal/absolute/link/duplicate/decompression/unexpected archive rejection without replacement.
- [x] 2.6 GREEN: add `internal/adapters/release/{trust,download,extract}.go` manifest-authorized, isolated download/extraction primitives only.
- [x] 2.7 RED: `internal/adapters/release/lifecycle_test.go`: argv literal/timeout/failure/wrong binary; ownership/symlink refusal; install/upgrade/status/doctor/rollback; failed health restores prior binary.
- [x] 2.8 GREEN: add `internal/adapters/release/lifecycle.go` and `internal/app/release.go` around Unit 3 primitives only.
- [x] 2.9 RED: test signed Linux/darwin amd64/arm64 archives, non-evaluating bootstrap, and no committed secret in `internal/adapters/release/bootstrap_test.go`.
- [x] 2.10 GREEN: add `scripts/install.sh`, `assets/release/`, `.github/workflows/release.yml` trusted metadata and platform packages without a private key.

## Phase 3: Managed Agent Integration

- [ ] 3.1 RED: fixtures `testdata/agent/{opencode,codex,claude,copilot,pi}/`; healthy OpenCode; unsupported/malformed preserve; Codex configure; Claude owned-only removal; drift/lock/route/probe no-write.
- [ ] 3.2 GREEN: add pinned `internal/adapters/agent/` discovery/route/merge/ownership/guidance/launcher/probe and `assets/` guidance.
- [ ] 3.3 RED: test independent operation states in `internal/app/agent_test.go`.
- [ ] 3.4 GREEN: add `internal/app/agent.go` status orchestration.

## Phase 4: Acceptance and Documentation

- [ ] 4.1 RED: `cmd/docmanager/acceptance_test.go`: scenarios/JSON/dry-run/status/doctor/MCP/receipt/SQLite; macOS lexical `/var`→physical `/private/var`; reject symlink target/escape; Windows `docmanager.exe` build/PowerShell smoke.
- [ ] 4.2 GREEN: fix `internal/app/lifecycle.go`, `.github/workflows/ci.yml`; verify Linux/macOS/Windows native package smoke, `go test ./...`, and `go build ./cmd/docmanager`.
- [ ] 4.3 Update `README.md`, `docs/{installers,agents,compatibility,recovery}.md`; verify bootstrap, recovery, aliases, consent, fixtures, rollback.
