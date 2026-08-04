# Tasks: OpenCode Daily DocManager Guidance

## Review Workload Forecast

Estimated authored changed lines: 1,150–1,450
Delivery strategy: single-pr; required decision: approve `size:exception` before apply.

Decision needed before apply: Yes
Chained PRs recommended: Yes
Chain strategy: size-exception
400-line budget risk: High

### Work Units

| Unit | Goal | Test command | Harness | Rollback boundary |
|---|---|---|---|---|
| 1 | Guidance/pair | `go test ./internal/app ./internal/adapters/agent` | `go test ./internal/adapters/agent -run TestOpenCode -v`: temp-Git pair/no-op/drift | Reverse-order only: revert Unit 2, then delete asset and Unit 1 hunks/tests. |
| 2 | CLI/receipt | `go test ./internal/adapters/mcp ./cmd/docmanager` | `go test ./cmd/docmanager -run TestAcceptance -v`: configure/stage/review/verify; probe success/failure/timeout | Revert Unit 2 command and install/status/metadata/receipt hunks/tests. |
| 3 | Docs/proof | `go test ./...` | `go build -o /tmp/docmanager ./cmd/docmanager && /tmp/docmanager --help`: no mutation | Revert only OpenCode sections/fixtures. |

## Phase 1: Guidance Asset and Owned Configuration

- [x] 1.1 RED: `lifecycle_test.go` requires version/readability, explicit scopes, one analysis, review then one unchanged verify; prohibits unrelated questions, routine coding, inferred/ambiguous scope, auto-editing, unavailable evidence, MCP lifecycle, workspace/hook changes, hook auto-enable, other agents, generic agent/release composition, historical evidence, and user-instruction/`AGENTS.md` mutation.
- [x] 1.2 RED: tests cover fresh/older-owned upgrade and moved/edited/missing/substituted drift refusal; preserve `.docmanager`/user files.
- [x] 1.3 GREEN: create `assets/opencode.md`; register catalog/identity/snapshots in `assets/assets.go` and `internal/app/lifecycle.go`.
- [x] 1.4 RED: `core_opencode_test.go` covers absent/`[]string`, preserved order, unsupported/conflict/incomplete no-write, no-op, owned upgrade, drift, and exact paired unconfigure.
- [x] 1.5 GREEN: implement atomic MCP/path configuration and ownership checks in `internal/adapters/agent/{opencode,registry}.go`.

## Phase 2: OpenCode Lifecycle and Inspection

- [x] 2.1 RED: `install_test.go` asserts workspace failure reports failed/previous state, zero configure/probe, unchanged config; separate preflight/write failures retain workspace/existing pair/original bytes and make zero probe calls.
- [x] 2.2 RED: bounded probe success/failure/timeout; failure removes only a newly-created unchanged pair, retains pre-existing owned pair, and refuses drifted-pair rollback; workspace survives each.
- [x] 2.3 GREEN: orchestrate install, preflight/configure, bounded probe, and compare-and-write rollback in `install.go`/`lifecycle.go`.
- [x] 2.4 RED: OpenCode status/doctor tests cover healthy plus unsupported/unreadable/moved/missing/drifted, probe success/failure/timeout, byte-identical no-repair inspection.
- [x] 2.5 GREEN: add only `docmanager opencode status`/`doctor`; wire `OpenCode.Status/Inspect`, never generic agent/release composition.

## Phase 3: Bounded MCP Workflow

- [x] 3.1 RED: `mcp_test.go` covers explicit worktree/index-staged/two-dot range, one-call/read-only limits, every guidance prohibition, and review.
- [x] 3.2 GREEN: update `internal/adapters/mcp/mcp.go`; preserve index-versus-unstaged resolver semantics.
- [x] 3.3 RED: `internal/app/*_test.go` table tests: changed scope/considered bytes or consumed receipt refuse/no verify; unrelated bytes remain eligible.
- [x] 3.4 GREEN: implement byte provider, invalidators, re-analysis snapshot, and review gate in receipt flow.

## Phase 4: Acceptance, Documentation, and Cleanup

- [x] 4.1 RED: acceptance transcripts cover config/no-op/upgrade/drift, OpenCode-only status/doctor, exact unconfigure, staged/range, every negative no-call, review, invalidation/re-analysis, and one unchanged verify.
- [x] 4.2 GREEN: wire fakes proving unrelated config survives and every call/non-call matches transcripts.
- [x] 4.3 Update only OpenCode sections in `README.md`, `docs/agents.md`, `ONBOARDING.md`: scopes, review, rollback, non-goals.
- [x] 4.4 Run `gofmt -w cmd internal`, focused tests, `go test ./...`, `go build ./cmd/docmanager`; record runtime/rollback evidence; verify no hooks, user instructions, other agents, releases, or installer-foundation changes.
