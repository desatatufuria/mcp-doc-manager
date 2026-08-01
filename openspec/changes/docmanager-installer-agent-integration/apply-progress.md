# Apply Progress: Docmanager Installer and Agent Integration

## Work Unit 1 — Contract (PR #1)

Completed tasks: 1.1, 1.2, 1.3, 1.4.

### TDD Cycle Evidence

| Task | Test File | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|---|---|---|---|---|---|---|---|
| 1.1 | `internal/app/identity_test.go`, `internal/app/contract_test.go` | Unit | `go test ./internal/app ./cmd/docmanager` — pass before changes | `go test ./internal/app -run 'TestValidateIdentity|TestValidateRequestRejectsClosedOperationInputsBeforeExecution|TestExecuteReturnsHeadlessJSONUnionAndDryRunPlanWithoutChanges'` — build failed: undefined contract symbols | Same command — PASS | Canonical/legacy/unknown identities; four invalid operation cases; dry-run plan case | gofmt; focused tests remained PASS |
| 1.2 | `internal/app/identity_test.go`, `internal/app/contract_test.go` | Unit | New contract files; existing app baseline above | Contract tests written in 1.1 before identity/contract implementation | `go test ./internal/app -run 'TestValidateIdentity|TestValidateRequestRejectsClosedOperationInputsBeforeExecution|TestExecuteReturnsHeadlessJSONUnionAndDryRunPlanWithoutChanges'` — PASS | Module identity plus typed result/error and closed-input paths | gofmt; focused tests remained PASS |
| 1.3 | `cmd/docmanager/main_test.go` | Integration | `go test ./internal/app ./cmd/docmanager` — pass before changes | `go test ./cmd/docmanager -run 'TestCLIHeadlessHierarchyReturnsJSONResult|TestCLIAliasesAreDeprecatedOnlyInHumanOutput'` — FAIL: unsupported_request and absent deprecation text | Same command — PASS | Workspace JSON and human/machine alias outputs | gofmt; focused tests remained PASS |
| 1.4 | `cmd/docmanager/main_test.go` | Integration | Existing CLI baseline above | CLI tests written in 1.3 before routing implementation | `go test ./cmd/docmanager -run 'TestCLIHeadlessHierarchyReturnsJSONResult|TestCLIAliasesAreDeprecatedOnlyInHumanOutput'` — PASS | Hierarchical route and JSON/non-JSON alias branches | gofmt; focused tests remained PASS |

### Work Unit Evidence

| Evidence | Result |
|---|---|
| Focused test command and exact result | `go test ./internal/app ./cmd/docmanager` — exit 0; both packages PASS. |
| Regression command and exact result | `go test ./...` — exit 0; `assets` has no tests; `cmd/docmanager`, `internal/adapters/{git,mcp,sqlite}`, `internal/app`, and `internal/domain` PASS. |
| Runtime harness command/scenario and exact result | `tmp="$(mktemp -d)" && bin="/tmp/opencode/docmanager-work-unit-1" && git -C "$tmp" init && go build -o "$bin" ./cmd/docmanager && env -C "$tmp" "$bin" workspace status --json && test -z "$(git -C "$tmp" status --porcelain)" && rm -rf "$tmp" "$bin" && test ! -e "$tmp" && test ! -e "$bin"` — exit 0; isolated Git repository returned `workspace.status`, outcome `status`, state/hook `absent`; no worktree write; temporary repository and binary removed. |
| Rollback boundary | Revert `go.mod`, canonical Go imports, `internal/app/{identity,contract}.go`, CLI routing/tests, and this Work Unit's task/progress entries. MCP adapter, receipt implementation, and SQLite ledger behavior remain otherwise unchanged. |
| Cleanup/process evidence | Harness created no background process; all invoked `git`, `go`, and CLI processes exited before the command returned. Explicit removal assertions passed for the temporary repository and `/tmp/opencode/docmanager-work-unit-1`. |

### Delivery Boundary

- Strategy: feature-branch-chain; PR #1 targets `feature/docmanager-installer-agent-integration`.
- Included: canonical identity migration, typed no-write contract, CLI routing, and compatibility aliases.
- Excluded: Phase 2+ workspace transactions/hooks, releases, agents, assets, workflows, and documentation.

## Apply Result Contract

### Implementation Mode

Strict TDD.

### Files Changed

| File or path | Action | Purpose |
|---|---|---|
| `go.mod` | Modified | Published the canonical module identity. |
| `cmd/docmanager/main.go` | Modified | Added headless `release`, `agent`, and `workspace` routing plus compatibility aliases. |
| `cmd/docmanager/main_test.go` | Modified | Added hierarchy, JSON, and human-only alias coverage. |
| `internal/app/identity.go` | Created | Added canonical and legacy identity classification. |
| `internal/app/identity_test.go` | Created | Added identity contract tests. |
| `internal/app/contract.go` | Created | Added typed operations, requests, results, stable errors, and no-write contract boundary. |
| `internal/app/contract_test.go` | Created | Added closed-input and dry-run contract tests. |
| `internal/app/{document_change,hook,lifecycle,verify_receipt}.go` and tests | Modified | Migrated internal imports to the canonical module identity without changing behavior. |
| `internal/adapters/{git,mcp,sqlite}/*.go` and tests | Modified | Migrated internal imports while preserving MCP, receipt, and SQLite ledger behavior. |
| `tasks.md` | Modified | Marked only Phase 1 tasks 1.1–1.4 complete. |
| `apply-progress.md` | Created/Modified | Recorded TDD, work-unit, and apply result evidence. |

### Deviations from Design

None — implementation matches the Phase 1 identity, typed-contract, routing, and compatibility-alias design. Phase 2+ behavior remains explicitly unimplemented.

### Issues Found

None. The initial work-unit diff exceeds the nominal 400-line review budget because canonical import migration touches existing Go sources; it remains the authorized PR #1 chained slice.

### Strict-TDD Test Summary

- Total tests written: unavailable; recorded output does not include an exact test count.
- Total tests passing: unavailable; recorded output reports package-level PASS results without an exact count.
- Layers used: Unit and Integration; exact per-layer counts unavailable.
- Approval tests: unavailable; no exact count was captured.
- Pure functions created: unavailable; no exact count was captured.

### Remaining Tasks

- [ ] 2.1–2.8 Transactional workspace and release RED/GREEN tasks.
- [ ] 3.1–3.4 Managed agent integration RED/GREEN tasks.
- [ ] 4.1–4.3 Acceptance tests, final verification, and documentation.

### Aggregate Status

4/19 tasks complete. Ready for the next batch; not ready for verification.

### Workload / PR Boundary

- Mode: chained PR slice using `feature-branch-chain`.
- Current work unit: Work Unit 1 — Contract (PR #1).
- PR #1 base/tracker: `feature/docmanager-installer-agent-integration`.
- Boundary: canonical identity, typed no-write contract, CLI routing, compatibility aliases, and their evidence; Phase 2+ is out of scope.
- Changed-line evidence: 161 tracked additions and 26 tracked deletions; 287 lines in newly created product/test files; 34 lines in the original progress artifact evidence.
- Estimated review-budget impact: at least 474 authored product changes (161 additions + 26 deletions + 287 new-file lines), exceeding the 400-line budget by at least 74 changes. This is the authorized PR #1 feature-branch-chain slice; no size exception is recorded.

### Return Envelope

- status: success
- executive_summary: Work Unit 1 completed with canonical identity migration, typed orchestration contract, headless CLI routes, and compatibility aliases. Existing recorded tests and isolated Git runtime harness passed.
- artifacts: `openspec/changes/docmanager-installer-agent-integration/tasks.md`; `openspec/changes/docmanager-installer-agent-integration/apply-progress.md`
- next_recommended: sdd-apply for the next assigned batch
- risks: PR #1 exceeds the nominal review budget due to mandatory import migration; continue only as the declared feature-branch-chain slice. No product behavior beyond Phase 1 was implemented.
- skill_resolution: paths-injected — sdd-apply, go-testing, work-unit-commits, chained-pr
- native_settlement_result: complete

### Evidence Revision

`sha256:2580059afd44210776e094a68fc268d2b83289a6a49556bad78eff73d196aa12` — SHA-256 of the ordered, per-file SHA-256 manifest for the Work Unit code, tests, canonical imports, and `tasks.md` checkbox evidence; excludes this self-describing progress file.

## Work Unit 2 — Workspace (PR #2)

Completed tasks: 2.1, 2.2, 2.3, 2.4.

### TDD Cycle Evidence

| Task | Test File | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|---|---|---|---|---|---|---|---|
| 2.1 | `internal/adapters/filesystem/transaction_test.go` | Unit | N/A (new package) | `go test ./internal/adapters/filesystem -run 'TestTransaction'` — build failed: `Open` and transaction errors undefined | Same command — PASS | Lock contention; creation/replacement digest+mode; drift, symlink, and route escape no-mutation cases | `gofmt`; focused package PASS |
| 2.2 | `internal/adapters/filesystem/transaction.go`, `internal/app/workspace.go` | Unit/Integration | Existing `go test ./internal/app` — PASS before CLI edits | Transaction RED in 2.1 and workspace service RED in 2.3 preceded production code | `go test ./internal/adapters/filesystem ./internal/app` — PASS | External XDG state, default absent hook, opted-in hook, drift refusal, and root validation | Extracted ownership-aware transaction and workspace state helpers; focused tests PASS |
| 2.3 | `internal/app/workspace_test.go` | Integration | `go test ./internal/app` — PASS before workspace integration | `go test ./internal/app -run 'TestWorkspace'` — build failed: workspace functions/errors undefined | Same command — PASS | Default/opt-in/drift/non-root Git selector cases | `gofmt`; focused tests PASS |
| 2.4 | `cmd/docmanager/workspace_test.go` | Integration | `go test ./cmd/docmanager` — PASS before CLI routing edit | `go test ./cmd/docmanager -run 'TestCLIWorkspaceInstallHonorsExplicitHookOptIn'` — FAIL: `pre-push` absent after `--enable-hook` | `go test ./internal/adapters/filesystem ./internal/app ./cmd/docmanager` — PASS | CLI explicit opt-in plus app default/opt-in and drift branches | `gofmt`; focused tests PASS |

### Work Unit Evidence

| Evidence | Result |
|---|---|
| Focused test command and exact result | `go test ./internal/adapters/filesystem ./internal/app` — exit 0; both packages PASS. |
| Regression command and exact result | `go test ./...` — exit 0; `assets` has no tests; all tested packages PASS. |
| Runtime harness command/scenario and exact result | `tmp="$(mktemp -d)" && home="$tmp/home" && state="$tmp/state" && bin="/tmp/opencode/docmanager-work-unit-2" && env HOME="$home" XDG_STATE_HOME="$state" GIT_CONFIG_NOSYSTEM=1 GIT_CONFIG_GLOBAL=/dev/null git init "$tmp" >/dev/null && go build -o "$bin" ./cmd/docmanager && env HOME="$home" XDG_STATE_HOME="$state" GIT_CONFIG_NOSYSTEM=1 GIT_CONFIG_GLOBAL=/dev/null "$bin" workspace install --target "$tmp" --json && test ! -e "$tmp/.git/hooks/pre-push" && env HOME="$home" XDG_STATE_HOME="$state" "$bin" workspace status --target "$tmp" --json && env HOME="$home" XDG_STATE_HOME="$state" "$bin" workspace install --target "$tmp" --enable-hook --json && env HOME="$home" XDG_STATE_HOME="$state" "$bin" workspace status --target "$tmp" --json && env HOME="$home" XDG_STATE_HOME="$state" "$bin" workspace uninstall --target "$tmp" --json && test ! -e "$tmp/.docmanager" && test ! -e "$tmp/.git/hooks/pre-push" && test -z "$(git -C "$tmp" status --porcelain)" && rm -rf "$tmp" "$bin" && test ! -e "$tmp" && test ! -e "$bin"` — exit 0; isolated install reported absent hook by default, explicit opt-in reported `opted-in`, status tracked each state, uninstall removed only managed workspace artifacts, and cleanup assertions passed. |
| Rollback boundary | Revert `internal/adapters/filesystem/{transaction,transaction_test}.go`, `internal/app/{workspace,workspace_test}.go`, `cmd/docmanager/{main.go,workspace_test.go}`, and this Work Unit's task/progress entries. Existing `.docmanager` lifecycle, MCP tools, receipt behavior, and SQLite ledger remain outside this boundary. |
| Cleanup/process evidence | The harness used only a new temporary Git root plus isolated `HOME`/`XDG_STATE_HOME`; all `git`, `go`, and CLI child processes exited before return. The temporary repository and binary were explicitly removed and confirmed absent. |

### Delivery Boundary

- Strategy: feature-branch-chain; Work Unit 2 is the child slice based on the completed Work Unit 1 boundary.
- Included: ownership-aware atomic filesystem transactions, external XDG workspace hook ownership state, workspace lifecycle/status/doctor, and explicit CLI hook opt-in.
- Excluded: release downloading/lifecycle, agent adapters/configuration, assets/workflows, and tasks 2.5+.

### Work Unit 2 Files Changed

| File or path | Action | Purpose |
|---|---|---|
| `internal/adapters/filesystem/transaction.go` | Created | Added locks, ownership digest/mode checks, path rejection, fsynced atomic replacement. |
| `internal/adapters/filesystem/transaction_test.go` | Created | Added lock, ownership, atomicity, drift, symlink, and route-escape tests. |
| `internal/app/workspace.go` | Created | Added exact-root workspace lifecycle and isolated XDG hook ownership state. |
| `internal/app/workspace_test.go` | Created | Added default absence, explicit consent, drift, and selector coverage. |
| `cmd/docmanager/main.go` | Modified | Routed workspace operations through the workspace service with stable drift/ownership errors. |
| `cmd/docmanager/workspace_test.go` | Created | Proved `workspace install --enable-hook` creates the managed hook. |
| `tasks.md` | Modified | Marked only tasks 2.1–2.4 complete. |
| `apply-progress.md` | Modified | Added cumulative Work Unit 2 evidence while retaining Work Unit 1 evidence. |

### Work Unit 2 Result

- Deviations from design: None — hooks stay absent unless `--enable-hook` is explicitly supplied; binary/agent configuration cannot enable hooks.
- Issues found: None.
- Changed-line evidence: 637 additions and 9 deletions in product/test files, plus task/progress evidence; 646 authored product/test changes are below the 800-line runtime budget but exceed the nominal 400-line review budget, so this remains the approved feature-branch-chain slice.
- Aggregate status: 8/19 tasks complete. Ready for the next assigned batch; not ready for verification.
- Evidence revision: `sha256:1b1a05dae3b99bbbc0cbb35951762b2d0725768bc21095e78f63f4e9a057b0b8` — SHA-256 of the ordered per-file SHA-256 manifest for Work Unit 2 code, tests, CLI route, and `tasks.md`; excludes this self-describing progress file.

## Work Unit 2 Apply Result Contract

### Status

success

### Executive Summary

Work Unit 2 completed transactional filesystem ownership checks, external XDG workspace state, workspace lifecycle/status/doctor behavior, and explicit CLI hook opt-in. Existing repository-local `.docmanager` lifecycle semantics remain intact, and hooks remain absent unless the caller supplies `--enable-hook`.

### Artifacts

- `openspec/changes/docmanager-installer-agent-integration/tasks.md` — tasks 2.1–2.4 marked complete.
- `openspec/changes/docmanager-installer-agent-integration/apply-progress.md` — cumulative Work Unit 1 and Work Unit 2 evidence plus this result contract.

### Next Recommended

sdd-apply for the next explicitly assigned batch.

### Risks

None. The Work Unit 2 product/test diff exceeds the nominal 400-line review budget, but it is the approved feature-branch-chain slice and remains within the native 800-line budget.

### Skill Resolution

paths-injected — sdd-apply, go-testing, work-unit-commits, chained-pr.

### Implementation Mode

Strict TDD.

### Files Inventory

| File or path | Action | Purpose |
|---|---|---|
| `internal/adapters/filesystem/transaction.go` | Created | Ownership-aware locks, safe paths, digest/mode validation, and fsynced atomic replacement. |
| `internal/adapters/filesystem/transaction_test.go` | Created | Lock, digest/mode, atomicity, drift, symlink, and route-escape coverage. |
| `internal/app/workspace.go` | Created | Exact Git-root workspace lifecycle with XDG-owned hook state. |
| `internal/app/workspace_test.go` | Created | Default absence, explicit consent, drift refusal, and non-root selector coverage. |
| `cmd/docmanager/main.go` | Modified | Workspace service routing and stable workspace errors. |
| `cmd/docmanager/workspace_test.go` | Created | Explicit CLI hook opt-in coverage. |
| `openspec/changes/docmanager-installer-agent-integration/tasks.md` | Modified | Completed only tasks 2.1–2.4. |
| `openspec/changes/docmanager-installer-agent-integration/apply-progress.md` | Modified | Cumulative implementation evidence and Work Unit 2 result contract. |

### Deviations from Design

None — implementation matches the approved workspace routing, ownership, locking, atomicity, drift, and explicit-consent decisions.

### Issues Found

None.

### Strict-TDD Test Summary

- Total tests written: 8 named Work Unit 2 test functions (3 unit, 5 integration); subtest count is unavailable because the recorded commands did not emit per-test output.
- Total tests passing: unavailable as an exact count; the recorded focused, package, and full-suite commands exited 0.
- Layers used: Unit (3 named test functions), Integration (5 named test functions).
- Approval tests: None — this work introduced new workspace/transaction behavior rather than refactoring existing behavior.
- Pure functions created: unavailable; no separate count was recorded.

### Remaining Tasks

- [ ] 2.5–2.8 Release-management RED/GREEN tasks.
- [ ] 3.1–3.4 Managed agent integration tasks.
- [ ] 4.1–4.3 Acceptance, verification, and documentation tasks.

### Aggregate Completion Status

8/19 tasks complete. Only tasks 1.1–1.4 and 2.1–2.4 are checked in `tasks.md`. Ready for the next assigned batch; not ready for verification.

### Workload and Branch Boundary

- Mode: approved feature-branch-chain slice.
- Current work unit: Work Unit 2 — Workspace (PR #2 child slice).
- Base/tracker boundary: completed Work Unit 1 boundary on `feature/docmanager-installer-agent-integration`; no branch, commit, push, merge, or PR action was performed here.
- Included: filesystem transaction, workspace lifecycle, XDG ownership state, and explicit hooks.
- Excluded: tasks 2.5+, releases, agent adapters, assets, workflows, and documentation.
- Product/test changed lines: 646.
- Total changed lines including `tasks.md` and `apply-progress.md` evidence: 703.
- Native budget: 800 changed lines; Work Unit 2 is within budget.

### Native Settlement

- native_settlement_result: complete
- Evidence revision: `sha256:1b1a05dae3b99bbbc0cbb35951762b2d0725768bc21095e78f63f4e9a057b0b8`
