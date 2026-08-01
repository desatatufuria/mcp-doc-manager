# Tasks: Docmanager Installer and Agent Integration

## Review Workload Forecast

| Field | Value |
|---|---|
| Estimated changed lines | 2,400–3,300 authored; fixtures/assets excluded |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Suggested split | PR #1 → PR #2 → PR #3 → PR #4 → PR #5; tracker merges to main last |
| Delivery strategy | ask-on-risk |
| Chain strategy | feature-branch-chain |

Decision needed before apply: No
Chained PRs recommended: Yes
Chain strategy: feature-branch-chain
400-line budget risk: High

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|---|---|---|---|---|---|
| 1 | Contract | PR #1 base: `feature/docmanager-installer-agent-integration` | `go test ./internal/app ./cmd/docmanager` | temp Git: `workspace status --json` | `go.mod`, contract, CLI |
| 2 | Workspace | PR #2 base: PR #1 branch | `go test ./internal/adapters/filesystem ./internal/app` | temp Git install/status/uninstall | filesystem, workspace, hooks |
| 3 | Releases | PR #3 base: PR #2 branch | `go test ./internal/adapters/release ./internal/app` | `httptest` install/upgrade/rollback | release, bootstrap, workflow |
| 4 | Agents | PR #4 base: PR #3 branch | `go test ./internal/adapters/agent ./internal/app` | isolated XDG/HOME/VS Code | adapters, fixtures, assets |
| 5 | Acceptance | PR #5 base: PR #4 branch | `go test ./...` | build + temp CLI matrix | acceptance tests and docs |

## Phase 1: Identity and Typed Contract

- [x] 1.1 RED: table-test `internal/app/{identity,contract}_test.go`: canonical/legacy; MCP/receipt/ledger unchanged; closed-operation invalid input pre-write; headless, JSON union, dry-run no-write.
- [x] 1.2 GREEN: migrate `go.mod`/imports; add `internal/app/{identity,contract}.go` typed context results, unions, errors, ports; preserve `internal/adapters/{mcp,sqlite}`.
- [x] 1.3 RED: test hierarchy, JSON, and deprecated human-only aliases in `cmd/docmanager/main_test.go`.
- [x] 1.4 GREEN: route `release|agent|workspace` and `install|doctor|uninstall --target` aliases in `cmd/docmanager/main.go`.

## Phase 2: Transactional Workspace and Release

- [ ] 2.1 RED: `internal/adapters/filesystem/transaction_test.go`: lock, backup digest/mode, fsync-rename, drift, ownership, symlink, route escape; no mutation.
- [ ] 2.2 GREEN: add `internal/adapters/filesystem/transaction.go`, `internal/app/workspace.go`, external state, and workspace install/uninstall/status/doctor.
- [ ] 2.3 RED: `internal/app/workspace_test.go`: default absent, opt-in, drift refusal, non-root Git selectors.
- [ ] 2.4 GREEN: implement explicit hooks in `internal/app/workspace.go` and `cmd/docmanager/main.go`.
- [ ] 2.5 RED: `internal/adapters/release/release_test.go`: valid Linux install, absent status; Windows; signature/expiry/revocation/rotation; HTTPS/redirect/size/digest; traversal/absolute/link/duplicate/decompression/unexpected entries; probe metacharacter/timeout/wrong binary; symlink/non-owned; health rollback.
- [ ] 2.6 GREEN: add `internal/adapters/release/` and `internal/app/release.go`: trust, extraction, argv probe, atomic lifecycle.
- [ ] 2.7 RED: test signed platform archives and non-evaluating bootstrap in `internal/adapters/release/bootstrap_test.go`.
- [ ] 2.8 GREEN: add `scripts/install.sh`, `assets/release/`, `.github/workflows/release.yml` for Linux/darwin amd64/arm64.

## Phase 3: Managed Agent Integration

- [ ] 3.1 RED: fixtures `testdata/agent/{opencode,codex,claude,copilot,pi}/`; test healthy OpenCode, unknown/unsupported and malformed preservation, Codex configure, Claude owned-only removal, drift, lock, route escape/symlink, literal/timeout/failing probe no-write.
- [ ] 3.2 GREEN: add `internal/adapters/agent/` pinned discovery/route/merge/ownership/guidance/launcher/probe adapters and `assets/` managed guidance.
- [ ] 3.3 RED: test `detect|configure|unconfigure|status|doctor` independent states in `internal/app/agent_test.go`.
- [ ] 3.4 GREEN: add `internal/app/agent.go` status orchestration.

## Phase 4: Acceptance and Documentation

- [ ] 4.1 RED: add `cmd/docmanager/acceptance_test.go` temp-repo matrix for every scenario, JSON/dry-run/status/doctor, MCP/receipt/SQLite.
- [ ] 4.2 GREEN: resolve acceptance failures; run `go test ./...` and `go build ./cmd/docmanager`.
- [ ] 4.3 Update `README.md` and `docs/{installers,agents,compatibility,recovery}.md`; verify clean bootstrap, recovery, aliases, consent, fixtures, rollback.
