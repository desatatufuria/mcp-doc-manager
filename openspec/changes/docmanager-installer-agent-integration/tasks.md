# Tasks: Docmanager Installer and Agent Integration

## Review Workload Forecast

| Field | Value |
|---|---|
| Estimated changed lines | 2,400–3,300 authored; fixtures/assets excluded |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Suggested split | Contract → workspace → trust → lifecycle → packaging → agent core → Codex/Claude → Copilot/Pi → orchestration → acceptance; tracker last |
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
| 6 | Agent core/OpenCode ≤800 | `feature/docmanager-installer-agent-core`, tracker after U5 | `go test ./internal/adapters/agent -run OpenCode` | isolated XDG JSON/JSONC | core/OpenCode only |
| 7 | Codex/Claude ≤800 | tracker after U6 | `go test ./internal/adapters/agent -run 'Codex|Claude'` | temp HOME, TOML/0600 JSON | Codex/Claude only |
| 8 | Copilot/Pi ≤800 | tracker after U7 | `go test ./internal/adapters/agent -run 'Copilot|Pi'` | isolated VS Code/Pi roots | Copilot/Pi only |
| 9 | Agent orchestration ≤800 | tracker after U8 | `go test ./internal/app -run Agent` | fake adapters, temp roots | app orchestration only |
| 10 | Acceptance | tracker after U9 | `go test ./...` | build + temp CLI matrix | acceptance tests and docs |

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

- [x] 3.1 RED: `internal/adapters/agent/core_opencode_test.go`, versioned `testdata/agent/opencode/` JSON/JSONC fixtures: registry/status, unknown/malformed/drift/symlink/route/probe no-write.
- [x] 3.2 GREEN: add `internal/adapters/agent/{registry,route,merge,opencode}.go` lock/ownership/atomic merge/guidance/probe abstractions and OpenCode adapter.
- [x] 3.3 RED: `internal/adapters/agent/codex_claude_test.go`, versioned TOML and `~/.claude.json` fixtures: exact MCP/guidance merge/unmerge, unrelated preservation, 0600 mode, drift refusal.
- [x] 3.4 GREEN: add `internal/adapters/agent/{codex,claude}.go` managed merge/unmerge adapters.
- [ ] 3.5 RED: `internal/adapters/agent/copilot_pi_test.go`: VS Code User OS paths, Pi MCP prerequisite/adapter, fixtures, merge/unmerge, guidance/status.
- [ ] 3.6 GREEN: add `internal/adapters/agent/{copilot,pi}.go` routes, adapters, and managed guidance.
- [ ] 3.7 RED: `internal/app/agent_test.go`: detect/configure/unconfigure/status/doctor for all agents; independent dimensions, dry-run/JSON, stable errors.
- [ ] 3.8 GREEN: add `internal/app/agent.go` orchestration across all named adapters.

## Phase 4: Acceptance and Documentation

- [ ] 4.1 RED: `cmd/docmanager/acceptance_test.go`: scenarios/JSON/dry-run/status/doctor/MCP/receipt/SQLite; macOS lexical `/var`→physical `/private/var`; reject symlink target/escape; Windows `docmanager.exe` build/PowerShell smoke.
- [ ] 4.2 GREEN: fix `internal/app/lifecycle.go`, `.github/workflows/ci.yml`; verify Linux/macOS/Windows native package smoke, `go test ./...`, and `go build ./cmd/docmanager`.
- [ ] 4.3 Update `README.md`, `docs/{installers,agents,compatibility,recovery}.md`; verify bootstrap, recovery, aliases, consent, fixtures, rollback.
