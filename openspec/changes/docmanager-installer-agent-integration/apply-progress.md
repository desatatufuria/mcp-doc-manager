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

## Work Unit 3 — Release Trust (re-sliced tasks 2.5–2.6)

Completed tasks: 2.5, 2.6.

### TDD Cycle Evidence

| Task | Test File | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|---|---|---|---|---|---|---|---|
| 2.5 | `internal/adapters/release/{trust,extract}_test.go` | Unit/Integration | N/A (new package) | `go test ./internal/adapters/release -run 'Trust|Extract'` — build failed: trust/extraction symbols undefined | Same command — exit 0; PASS | 19 subcases span valid and invalid trust, download, and archive paths | `gofmt`; focused, package, and full-suite tests remained PASS |
| 2.6 | `internal/adapters/release/{trust,download,extract}.go` | Unit/Integration | New production files; RED coverage above | RED tests from 2.5 preceded every production primitive | `go test ./internal/adapters/release` — exit 0; PASS | Valid artifact download/extract versus host, redirect, size, digest, traversal, link, duplicate, overflow, and unexpected cases | Added explicit primary-key policy and isolated extraction directory; tests remained PASS |

### Work Unit Evidence

| Evidence | Result |
|---|---|
| Focused test command and exact result | `go test ./internal/adapters/release -run 'Trust|Extract'` — exit 0; `TestTrust` and `TestExtractTarGz` PASS. |
| Release package test command and exact result | `go test ./internal/adapters/release` — exit 0; trust, download, and extraction tests PASS. |
| Regression command and exact result | `go test ./...` — exit 0; `assets` has no tests; all tested packages PASS. |
| Runtime harness command/scenario and exact result | `go test -v ./internal/adapters/release -run 'TestTrust|TestDownloadRejectsUntrustedResponses|TestExtractTarGz'` — exit 0; ephemeral in-memory Ed25519 manifest checks, `httptest.NewTLSServer` host/redirect/size/digest rejection, and `t.TempDir()` extraction rejection all PASS while preserving the sentinel target. |
| Formatting and diff checks | `gofmt -l internal/adapters/release` — no output; `git diff --check` — exit 0. |
| Rollback boundary | Revert only `internal/adapters/release/{trust,download,extract}.go`, their two tests, and the 2.5–2.6 task/progress entries. No lifecycle, probe, install/upgrade, scripts, assets, workflow, agent, or documentation behavior is included. |
| Cleanup/process evidence | `httptest` servers close via `defer`; every archive path uses `t.TempDir()` and failed extraction removes its generated directory. No shell execution, subprocess, persistent target replacement, or background process occurs. |

### Work Unit 3 Apply Result Contract

- status: success
- executive_summary: Added manifest trust verification, bounded allowlisted HTTPS artifact download, and isolated tar.gz extraction primitives with closed deterministic errors. No binary lifecycle or release publication behavior was introduced.
- artifacts: `internal/adapters/release/{trust,download,extract}.go`; `internal/adapters/release/{trust,extract}_test.go`; `openspec/changes/docmanager-installer-agent-integration/{tasks,apply-progress}.md`
- next_recommended: sdd-apply for explicitly assigned tasks 2.7–2.8 only
- risks: The primitives intentionally do not install, replace, probe, or roll back binaries; those lifecycle safeguards remain Work Unit 4 scope.
- skill_resolution: paths-injected — sdd-apply, go-testing, work-unit-commits, chained-pr
- implementation_mode: Strict TDD
- workload_boundary: approved feature-branch-chain Work Unit 3 child slice on `feature/docmanager-installer-release`, based on tracker `feature/docmanager-installer-agent-integration` after Work Unit 2; no PR, commit, push, merge, or release action.
- total_changed_lines_including_preexisting_resliced_task_evidence: 525
- native_budget: 525 < 800 changed lines
- native_settlement_result: complete
- evidence_revision: `sha256:6089b6821222f9462fb67b070a0c3934ccb09fa37d9f174d78712c6d2d2fc893` — ordered SHA-256 manifest of Work Unit 3 code, tests, and `tasks.md`; excludes this self-describing progress file.

### Aggregate Completion Status

10/21 tasks complete. `tasks.md` confirms ten checked tasks: 1.1–1.4 and 2.1–2.6. Ready for the next assigned batch; not ready for verification.

## Work Unit 4 — Release Lifecycle (tasks 2.7–2.8)

Completed tasks: 2.7, 2.8.

### TDD Cycle Evidence

| Task | Test File | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|---|---|---|---|---|---|---|---|
| 2.7 | `internal/adapters/release/lifecycle_test.go` | Unit/Integration | `go test ./internal/adapters/release ./internal/app -run 'Lifecycle|Probe'` — exit 0 before lifecycle tests existed; no matching tests | Same command after adding tests — build failed with undefined `Lifecycle`, `ProbeResult`, and lifecycle errors | Same command — exit 0; all lifecycle/probe cases PASS | Literal `--version` argv; process failure, wrong binary, bounded timeout; symlink/drift/lock refusal; absent status; install/upgrade/doctor/rollback; failed health restoration; real subprocess harness | Added small state/ownership helpers; focused tests remained PASS |
| 2.8 | `internal/adapters/release/lifecycle.go`, `internal/app/release.go` | Unit/Integration | `go test ./internal/app -run 'ReleaseService'` — no matching test before RED | Same command after `release_test.go` — build failed: undefined `ReleaseService` | `go test ./internal/adapters/release ./internal/app -run 'Lifecycle|Probe|ReleaseService'` — exit 0 | Closed read-only input refusal and absent release status; adapter lifecycle cases exercise mutation results | `gofmt`; package, full-suite, and harness tests remained PASS |

### Work Unit Evidence

| Evidence | Result |
|---|---|
| Focused test command and exact result | `go test ./internal/adapters/release ./internal/app -run 'Lifecycle|Probe'` — exit 0; release lifecycle/probe tests PASS and app package has no matching lifecycle/probe test. |
| Package and regression commands | `go test ./internal/adapters/release ./internal/app` — exit 0; both packages PASS. `go test ./...` — exit 0; `assets` has no tests and every tested package PASS. |
| Runtime harness command/scenario and exact result | `go test -v ./internal/adapters/release -run 'TestLifecycleOwnedBinaryHarness'` — exit 0; a `t.TempDir()` owned executable was installed, upgraded, and atomically rolled back through the real bounded argv-only subprocess runner. |
| Formatting and diff checks | `gofmt -l internal/adapters/release internal/app` — no output; `git diff --check` — exit 0. |
| Rollback boundary | Revert only `internal/adapters/release/{lifecycle,lifecycle_test}.go`, `internal/app/{release,release_test}.go`, and the 2.7–2.8 task/progress entries. Trust/download/extraction, MCP, receipt, ledger, workspace, bootstrap, assets, workflows, agents, acceptance, and documentation remain outside this boundary. |
| Cleanup/process evidence | All lifecycle tests use `t.TempDir()` with explicit temporary state roots. The runtime harness starts one foreground owned-binary probe and it exits before the test returns; no shell interpolation, persistent installation, background process, or target outside the temporary directory is used. |

### Work Unit 4 Apply Result Contract

- status: success
- executive_summary: Added an ownership-aware, lock-protected release lifecycle over extracted candidates, with argv-only bounded health probes, atomic replacement, prior-binary restoration, rollback, and read-only status/doctor. The app adapter maps those outcomes into the existing closed request/result/stable-error contract without CLI, MCP, receipt, ledger, workspace, bootstrap, asset, workflow, agent, or documentation changes.
- artifacts: `internal/adapters/release/{lifecycle,lifecycle_test}.go`; `internal/app/{release,release_test}.go`; `openspec/changes/docmanager-installer-agent-integration/{tasks,apply-progress}.md`; Engram topic `sdd/docmanager-installer-agent-integration/apply-progress`.
- next_recommended: sdd-apply for explicitly assigned tasks 2.9–2.10 or the next approved work-unit slice.
- risks: Installation is intentionally not wired to manifest retrieval, archive extraction, or CLI yet; `ReleaseService` accepts only the already-verified/extracted lifecycle boundary. No real installation was performed.
- skill_resolution: paths-injected — sdd-apply, go-testing, work-unit-commits, chained-pr.
- implementation_mode: Strict TDD.
- workload_boundary: approved feature-branch-chain Work Unit 4 child slice from `feature/docmanager-installer-agent-integration` after Work Unit 3; no commit, push, merge, PR, bootstrap, scripts, assets, workflow, agents, acceptance, or documentation action.
- native_token: retained by parent.
- changed_line_evidence: 617 code/test changed lines + 41 OpenSpec evidence lines = 658 total; `658 < 800`.
- native_settlement_result: complete
- evidence_revision: `sha256:6f3229cc6f46ff5ca3428a6bf69efbc60414b72619582852aae5d7af5fc174fc` — SHA-256 of the ordered per-file SHA-256 manifest for Work Unit 4 code, tests, and `tasks.md`; excludes this self-describing progress file.

### Aggregate Completion Status

12/21 tasks complete. `tasks.md` confirms tasks 1.1–1.4 and 2.1–2.8 checked. Ready for the next assigned batch; not ready for verification.

## Work Unit 5 — Release Packaging (tasks 2.9–2.10)

Completed tasks: 2.9, 2.10.

### TDD Cycle Evidence

| Task | Test File | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|---|---|---|---|---|---|---|---|
| 2.9 | `internal/adapters/release/bootstrap_test.go` | Integration/static | `go test ./internal/adapters/release -run Bootstrap` — exit 0, no matching tests before RED | Focused command — FAIL: missing `assets/release/trust.json`; later FAIL: missing macOS `shasum -a 256` fallback | `go test ./internal/adapters/release -run Bootstrap -count=1` — exit 0; 2 bootstrap tests PASS | Four Linux/darwin × amd64/arm64 matrix entries; valid public key/key ID; non-evaluating/no-secret static checks; isolated failed-download harness | Added macOS digest fallback and privilege/profile refusal assertions; focused tests remained PASS |
| 2.10 | `scripts/install.sh`, `assets/release/`, `.github/workflows/release.yml` | Integration/static | New production assets; task 2.9 RED suite above | Bootstrap and package tests were written before every asset/workflow behavior; shell syntax and workflow assertions fail if required safety/build settings disappear | `bash -n scripts/install.sh`; static CGo-free/trimpath/secret-reference/no-private-key assertions — exit 0 | Four package targets plus bootstrap failure-before-install fixture; exact expected release archive template and package-set assertions | Extracted portable `sha256` helper; no further refactor needed |

### Work Unit Evidence

| Evidence | Result |
|---|---|
| Focused test command and exact result | `go test ./internal/adapters/release -run Bootstrap -count=1` — exit 0; 2 tests PASS. |
| Release package and full Go tests | `go test ./internal/adapters/release` — exit 0; package PASS. `go test ./...` — exit 0; all tested packages PASS; `assets` reports no test files. |
| Runtime harness command/scenario and exact result | `TestBootstrapFixtureFailsClosedWithoutInstalling` invokes `sh scripts/install.sh` with a temporary PATH whose `curl` fails, an allowlisted HTTPS base, a temporary install directory, and no network. Exit was nonzero with `unable to fetch trusted manifest`; `install/docmanager` remained absent. |
| Shell/static checks | `bash -n scripts/install.sh && git diff --check` — exit 0. `shellcheck scripts/install.sh` — unavailable (command absent). `actionlint .github/workflows/release.yml` — unavailable (command absent). Static assertions for `CGO_ENABLED=0`, `-trimpath -buildvcs=false`, signing secret reference, and absence of private-key filenames — exit 0. |
| Rollback boundary | Revert only `internal/adapters/release/bootstrap_test.go`, `scripts/install.sh`, `assets/release/{trust.json,public-key.pem}`, `.github/workflows/release.yml`, and this Work Unit's task/progress entries. Trust/download/extraction/lifecycle, CLI, agents, acceptance regressions, and documentation remain outside this boundary. |
| Cleanup/process evidence | The fixture uses `t.TempDir()` and a foreground shell process; its temporary PATH, install directory, and failed download are removed when the test returns. The script traps and removes its `mktemp -d` directory. No network request succeeds, no real install occurs, and no background process is created. |

### Work Unit 5 Apply Result Contract

- status: success
- executive_summary: Added a fail-closed signed-manifest bootstrap, deterministic public trust metadata, and a GitHub release workflow for exactly Linux/darwin amd64/arm64 CGo-free archives. The bootstrap uses quoted data/argv, bounded HTTPS downloads, detached public-key signature verification, digest verification, temporary extraction, and atomic target replacement without sudo or shell-profile mutation.
- artifacts: `internal/adapters/release/bootstrap_test.go`; `scripts/install.sh`; `assets/release/{trust.json,public-key.pem}`; `.github/workflows/release.yml`; `openspec/changes/docmanager-installer-agent-integration/{tasks,apply-progress}.md`; Engram topic `sdd/docmanager-installer-agent-integration/apply-progress`.
- next_recommended: sdd-apply for explicitly assigned agent work only, or acceptance work after that dependency.
- risks: Release publishing is unexecuted in this local work unit; maintainers must configure `DOCMANAGER_RELEASE_SIGNING_KEY` to match the committed public key. The separately recorded macOS lexical-root and Windows native-package smoke regressions remain acceptance tasks 4.1–4.2 and were not changed.
- skill_resolution: paths-injected — sdd-apply, go-testing, work-unit-commits, chained-pr.
- implementation_mode: Strict TDD.
- workload_boundary: approved feature-branch-chain Work Unit 5 child slice based on the tracker after Work Unit 4; no commit, push, merge, PR, publication, agent, acceptance, or native-regression remediation action.
- native_settlement_result: complete
- changed_line_evidence: 245 product/test/script/asset/workflow lines + 42 OpenSpec task/progress evidence lines = 287 total; `287 < 800` native work-unit budget.
- evidence_revision: `sha256:a1f271e603dbd2d645a8b8f290e45d4dc8793dae4d9fb539d8332f9725402cfd` — SHA-256 of the ordered per-file SHA-256 manifest for Work Unit 5 code, bootstrap test, public metadata, workflow, and `tasks.md`; excludes this self-describing progress file.

### Aggregate Completion Status

14/21 tasks complete. `tasks.md` confirms tasks 1.1–1.4 and 2.1–2.10 checked. Agent tasks 3.1–3.4 and acceptance/documentation tasks 4.1–4.3 remain pending. Ready for the next assigned batch; not ready for verification.

## Work Unit 6 — Agent core/OpenCode (tasks 3.1–3.2)

Completed tasks: 3.1, 3.2.

### TDD Cycle Evidence

| Task | Test File | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|---|---|---|---|---|---|---|---|
| 3.1 | `internal/adapters/agent/core_opencode_test.go` | Unit/integration | N/A (new package) | `go test ./internal/adapters/agent -run OpenCode -count=1` — build failed: `NewOpenCode`, options, and safe-boundary errors undefined | Same command — exit 0; all OpenCode tests PASS | JSON and JSONC routes; installed/supported/configured/healthy states; malformed, unknown, ownership, drift, symlink, route, lock, literal, timeout, and failed-probe branches | Extracted route, merge, registry, and adapter boundaries; focused tests remained PASS |
| 3.2 | `internal/adapters/agent/{registry,route,merge,opencode}.go` | Unit/integration | RED suite from 3.1 preceded production code | 3.1 covered every new adapter boundary before production files existed | `go test ./internal/adapters/agent -count=1` — exit 0; package PASS | Idempotent configure/unconfigure and unrelated JSON/JSONC preservation exercise independent mutation paths | `gofmt -w internal/adapters/agent`; focused/package/full-suite checks remained PASS |

### Work Unit Evidence

| Evidence | Result |
|---|---|
| Focused test command and exact result | `go test ./internal/adapters/agent -run OpenCode -count=1` — exit 0; OpenCode focused tests PASS. |
| Package and regression commands | `go test ./internal/adapters/agent -count=1` — exit 0; package PASS. `go test ./...` — exit 0; all tested packages PASS; `assets` has no test files. |
| Runtime harness command/scenario and exact result | `go test -v ./internal/adapters/agent -run 'TestOpenCodeConfigure(StatusAndUnconfigureJSONC|AndUnconfigureJSON)$' -count=1` — exit 0; isolated `t.TempDir()` XDG JSONC and JSON roots configured, reported status, unconfigured, preserved unrelated content/comments, and removed only managed MCP/guidance. |
| Formatting and diff checks | `gofmt -w internal/adapters/agent`; `test -z "$(gofmt -l cmd internal spike)"`; `git diff --check` — exit 0. |
| Rollback boundary | Revert only `internal/adapters/agent/{registry,route,merge,opencode}.go`, `core_opencode_test.go`, `testdata/agent/opencode/`, and the 3.1–3.2 task/progress entries. Codex, Claude, Copilot, Pi, app/CLI orchestration, acceptance, MCP, receipt, ledger, and CI regression work remain outside this boundary. |
| Cleanup/process evidence | All tests use `t.TempDir()` and injected discovery/runner functions; no real HOME, XDG configuration, user config, shell, background process, or external subprocess is touched. Lock files are released by deferred cleanup and atomic temporary files are deferred for removal. |

### Work Unit 6 Apply Result Contract

- status: success
- executive_summary: Added fixture-pinned OpenCode discovery, route validation, ownership-aware JSON/JSONC managed MCP/guidance mutation, locking, atomic writes, and bounded argv-only health probes. Detection is read-only and reports installed, supported, configured, and healthy independently.
- artifacts: `internal/adapters/agent/{registry,route,merge,opencode}.go`; `internal/adapters/agent/core_opencode_test.go`; `testdata/agent/opencode/{registry-v1,valid}.{json,jsonc}`; `openspec/changes/docmanager-installer-agent-integration/{tasks,apply-progress}.md`; Engram topic `sdd/docmanager-installer-agent-integration/apply-progress`.
- next_recommended: sdd-apply for explicitly assigned tasks 3.3–3.4 only.
- risks: The OpenCode reference adapter is not wired into app/CLI orchestration; Codex, Claude, Copilot, Pi, and acceptance/CI regressions remain pending by design.
- skill_resolution: paths-injected — sdd-apply, work-unit-commits, go-testing, chained-pr.
- implementation_mode: Strict TDD.
- workload_boundary: approved feature-branch-chain Work Unit 6 on `feature/docmanager-installer-agent-core`, based on tracker after packaging; no commit, push, merge, PR, Codex, Claude, Copilot, Pi, app/CLI orchestration, acceptance, or CI-regression action.
- native_token: retained by parent.
- native_settlement_result: complete
- changed_line_evidence: 564 product/test/fixture lines + 42 task/progress evidence lines = 606 total; `606 < 800` native work-unit budget.
- evidence_revision: `sha256:37b153941536ecfe9d7f6c7c2beaa25259b3f78f128b23f253e1433d63631fad` — SHA-256 of the ordered per-file SHA-256 manifest for Work Unit 6 agent code, tests, fixtures, and `tasks.md`; excludes this self-describing progress file.

### Aggregate Completion Status

16/25 tasks complete. `tasks.md` confirms 1.1–1.4, 2.1–2.10, and 3.1–3.2 checked. Tasks 3.3–3.8 and 4.1–4.3 remain pending. Ready for the next assigned batch; not ready for verification.

## Work Unit 7 — Codex/Claude (tasks 3.3–3.4)

Completed tasks: 3.3, 3.4.

### TDD Cycle Evidence

| Task | Test File | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|---|---|---|---|---|---|---|---|
| 3.3 | `internal/adapters/agent/codex_claude_test.go` | Unit/integration | `go test ./internal/adapters/agent -run 'Codex|Claude' -count=1` — package built before the new test | Same focused command — build FAILED: `NewCodex`, `CodexOptions`, `NewClaude`, and `ClaudeOptions` undefined | Same command — exit 0; all Codex/Claude tests PASS | Configure/unconfigure, unrelated TOML/OAuth preservation, idempotence, 0600, malformed/conflict/drift/symlink/lock/route/probe no-write cases | Extracted adapter-local routes and marker checks; focused tests remained PASS |
| 3.4 | `internal/adapters/agent/{codex,claude}.go` | Unit/integration | RED suite from 3.3 preceded production code | 3.3 referenced every adapter constructor before implementation | `go test ./internal/adapters/agent -count=1` — exit 0; package PASS | Codex TOML versus Claude JSON/config-mode routes exercise distinct parsing and mutation paths | `gofmt -w internal/adapters/agent`; focused/package/full-suite checks remained PASS |

### Work Unit Evidence

| Evidence | Result |
|---|---|
| Focused test command and exact result | `go test ./internal/adapters/agent -run 'Codex|Claude' -count=1` — exit 0; Codex and Claude focused tests PASS. |
| Package and regression commands | `go test ./internal/adapters/agent -count=1` — exit 0; package PASS. `go test ./...` — exit 0; all tested packages PASS; `assets` has no test files. |
| Runtime harness command/scenario and exact result | `go test -v ./internal/adapters/agent -run 'Test(CodexConfigureStatusAndUnconfigure|ClaudeConfigureStatusAndUnconfigurePreservesModeAndOAuth)$' -count=1` — exit 0; isolated `t.TempDir()` HOME roots configured, reported status, unconfigured, preserved unrelated TOML/OAuth content, and verified Claude mode `0600`. |
| Formatting and diff checks | `gofmt -w internal/adapters/agent`; `test -z "$(gofmt -l cmd internal spike)"`; `git diff --check` — exit 0. |
| Rollback boundary | Revert only `internal/adapters/agent/{codex,claude}.go`, `codex_claude_test.go`, `testdata/agent/{codex,claude}/`, and the 3.3–3.4 task/progress entries. OpenCode core, Copilot, Pi, app/CLI orchestration, acceptance, MCP, receipt, and ledger behavior remain outside this boundary. |
| Cleanup/process evidence | Tests use only `t.TempDir()` HOME roots and injected runners. No real HOME/config is read or written; no shell/background process occurs; lock files and temporary atomic-write files are removed by deferred cleanup. |

### Work Unit 7 Apply Result Contract

- status: success
- executive_summary: Added fixture-pinned Codex TOML and Claude sensitive JSON adapters with owned MCP/guidance lifecycles, independent status dimensions, absolute argv-only bounded probes, locks, atomic writes, drift/conflict/malformed/symlink/route refusal, and Claude `0600` enforcement.
- artifacts: `internal/adapters/agent/{codex,claude}.go`; `internal/adapters/agent/codex_claude_test.go`; `testdata/agent/{codex,claude}/`; `openspec/changes/docmanager-installer-agent-integration/{tasks,apply-progress}.md`; Engram topic `sdd/docmanager-installer-agent-integration/apply-progress` remains preserved from Work Unit 6.
- next_recommended: sdd-apply for explicitly assigned tasks 3.5–3.6 only.
- risks: TOML support deliberately changes only the owned `[mcp_servers.docmanager]` block and preserves unrelated source bytes/tables; no generic TOML rewrite, no actual user config, and no app/CLI wiring was introduced.
- skill_resolution: paths-injected — sdd-apply, work-unit-commits, go-testing, chained-pr.
- implementation_mode: Strict TDD.
- workload_boundary: approved feature-branch-chain Work Unit 7 based on tracker after Work Unit 6; no commit, push, merge, PR, Copilot, Pi, app/CLI orchestration, acceptance, or CI-regression action.
- native_token: retained by parent.
- changed_line_evidence: 568 product/test/fixture lines + 43 OpenSpec task/progress evidence lines = 611 total; `611 < 800` native work-unit budget.
- native_settlement_result: complete
- evidence_revision: `sha256:cfb3bf997c7f1ad686de667c090cbdc24cd854d89f45e8e10262c4c4ec934ce1` — SHA-256 of the ordered per-file SHA-256 manifest for Work Unit 7 code, tests, fixtures, and `tasks.md`; excludes this self-describing progress file.

### Aggregate Completion Status

18/25 tasks complete. `tasks.md` confirms 1.1–1.4, 2.1–2.10, and 3.1–3.4 checked. Tasks 3.5–3.8 and 4.1–4.3 remain pending. Ready for the next assigned batch; not ready for verification.

## Work Unit 8 — Copilot/Pi (tasks 3.5–3.6)

Completed tasks: 3.5, 3.6.

### TDD Cycle Evidence

| Task | Test File | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|---|---|---|---|---|---|---|---|
| 3.5 | `internal/adapters/agent/copilot_pi_test.go` | Unit/integration | `go test ./internal/adapters/agent -count=1` — exit 0 before the new test file | `go test ./internal/adapters/agent -run 'Copilot|Pi' -count=1` — build failed: Copilot/Pi constructors, routes, options, and prerequisite error undefined | Same focused command — exit 0; Copilot/Pi tests PASS | Linux/darwin/windows Copilot routes; Pi prerequisite absent/present; configure/status/unconfigure; idempotence; unrelated preservation; malformed/conflict/drift/symlink/route/lock/probe no-write branches | Extracted shared JSON ownership/mutation helper; focused, package, and full-suite tests remained PASS |
| 3.6 | `internal/adapters/agent/{copilot,pi,json_agent}.go` | Unit/integration | RED suite from 3.5 preceded production code | 3.5 referenced all new constructors/routes before implementation | `go test ./internal/adapters/agent -count=1` — exit 0; package PASS | Copilot `servers` array command and Pi `mcpServers` command/args entries exercise distinct routes and managed shapes | `gofmt -w internal/adapters/agent`; focused/package/full-suite checks remained PASS |

### Work Unit Evidence

| Evidence | Result |
|---|---|
| Focused test command and exact result | `go test ./internal/adapters/agent -run 'Copilot|Pi' -count=1` — exit 0; all focused Copilot/Pi tests PASS. |
| Package and regression commands | `go test ./internal/adapters/agent -count=1` — exit 0; package PASS. `go test ./...` — exit 0; all tested packages PASS; `assets` has no test files. |
| Runtime harness command/scenario and exact result | `go test -v ./internal/adapters/agent -run 'Test(CopilotConfigureStatusAndUnconfigurePreservesServers|PiPrerequisiteConfigureStatusAndUnconfigure)$' -count=1` — exit 0; isolated `t.TempDir()` VS Code and Pi roots configured, reported healthy status, unconfigured, preserved unrelated MCP entries, and rejected the unavailable Pi adapter without writing. |
| Formatting and diff checks | `test -z "$(gofmt -l cmd internal spike)"`; `git diff --check` — exit 0. |
| Rollback boundary | Revert only `internal/adapters/agent/{copilot,pi,json_agent}.go`, `registry.go` prerequisite error, `copilot_pi_test.go`, `testdata/agent/{copilot,pi}/`, and the 3.5–3.6 task/progress entries. OpenCode/Codex/Claude, app/CLI orchestration, acceptance, MCP, receipt, and ledger behavior remain outside this boundary. |
| Cleanup/process evidence | Every test and harness uses `t.TempDir()` roots plus injected discovery/runner functions. No user HOME, XDG, APPDATA, npm install, shell, background process, or external subprocess is used; locks and atomic temporary files are removed before return. |

### Work Unit 8 Apply Result Contract

- status: success
- executive_summary: Added fixture-pinned GitHub Copilot VS Code User and Pi MCP adapters. Copilot owns only `servers.docmanager`; Pi owns only `mcpServers.docmanager`, requires an already-available Pi adapter, and never installs npm packages. Both adapters maintain independent detection dimensions, managed guidance, locks, atomic writes, drift refusal, and bounded argv-only probes.
- artifacts: `internal/adapters/agent/{copilot,pi,json_agent}.go`; `registry.go`; `copilot_pi_test.go`; `testdata/agent/{copilot,pi}/`; `openspec/changes/docmanager-installer-agent-integration/{tasks,apply-progress}.md`
- next_recommended: sdd-apply for explicitly assigned tasks 3.7–3.8 only.
- risks: OS route selection is tested through isolated injected roots; no actual VS Code, Pi, npm, user configuration, or app/CLI orchestration was invoked.
- skill_resolution: paths-injected — sdd-apply, work-unit-commits, go-testing, chained-pr.
- implementation_mode: Strict TDD.
- workload_boundary: approved feature-branch-chain Work Unit 8 child slice on `feature/docmanager-installer-copilot-pi`, based on the tracker after Work Unit 7; no commit, push, merge, PR, app orchestration, acceptance, or CI-regression action.
- native_token: retained by parent.
- native_settlement_result: complete
- changed_line_evidence: 680 additions + 9 deletions = 689 total including task/progress evidence; `689 < 800` native work-unit budget.
- evidence_revision: `sha256:e36da53488d89cf6920db777fb3b9c226275773796271fb7e66dcb7cda8d904a` — SHA-256 of the ordered per-file SHA-256 manifest for Work Unit 8 code, tests, fixtures, `registry.go`, and `tasks.md`; excludes this self-describing progress file.

### Aggregate Completion Status

20/25 tasks complete. `tasks.md` confirms 1.1–1.4, 2.1–2.10, and 3.1–3.6 checked. Tasks 3.7–3.8 and 4.1–4.3 remain pending. Ready for the next assigned batch; not ready for verification.

## Work Unit 9 — Agent orchestration (tasks 3.7–3.8)

Completed tasks: 3.7, 3.8.

### TDD Cycle Evidence

| Task | Test File | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|---|---|---|---|---|---|---|---|
| 3.7 | `internal/app/agent_test.go` | Unit/integration | `go test ./internal/app -count=1` — exit 0 before app agent tests | `go test ./internal/app -run Agent -count=1` — build FAILED: `AgentService` and `AgentPort` undefined | `go test ./internal/app -run Agent -count=1` — exit 0; app agent tests PASS | All/detected/explicit selections; five independent dimensions; read-only detect/status/doctor; dry-run; preflight; unsupported Pi/unknown/drift; partial rollback plus rollback-failure detail | Extracted selection, inspection, plan, rollback, and stable-error helpers; focused tests remained PASS |
| 3.8 | `internal/app/agent.go`, `internal/app/contract.go` | Unit/integration | Existing app baseline above | 3.7 referenced every service/port before production code existed | `go test ./internal/app ./internal/adapters/agent -count=1` — exit 0; both packages PASS | Isolated real OpenCode temporary-root configure/status/unconfigure plus five-port fake configure/dry-run/status/unconfigure/rollback harness | `gofmt`; focused, package, full-suite, and harness tests remained PASS |

### Work Unit Evidence

| Evidence | Result |
|---|---|
| Focused test command and exact result | `go test ./internal/app -run Agent -count=1` — exit 0; 5 AgentService tests PASS. |
| App and agent package command and exact result | `go test ./internal/app ./internal/adapters/agent -count=1` — exit 0; both packages PASS. |
| Full Go suite command and exact result | `go test ./...` — exit 0; all tested packages PASS; `assets` has no test files. |
| Runtime harness command/scenario and exact result | `go test -v ./internal/app -run 'TestAgentService(MultiAgentHarness|PlansBeforeWritesAndRollsBackPartialConfigure|UsesRealAdapterAtTemporaryRoot)$' -count=1` — exit 0; fake five-agent dry-run/configure/status/unconfigure and rollback scenarios plus an isolated real OpenCode temporary-root lifecycle PASS. |
| Formatting and diff checks | `test -z "$(gofmt -l cmd internal spike)"`; `git diff --check` — exit 0. |
| Rollback boundary | Revert only `internal/app/{agent,agent_test}.go`, the additive `Status.Agents` field in `internal/app/contract.go`, and the 3.7–3.8 task/progress entries. Adapters, CLI/TUI wiring, acceptance/native CI fixes, docs, MCP, receipt, and ledger behavior remain outside this boundary. |
| Cleanup/process evidence | All tests use `t.TempDir()` roots or in-memory ports. The harness invokes no shell, package installer, user HOME/XDG, background process, or external subprocess; temporary files are removed by Go test cleanup. |

### Work Unit 9 Apply Result Contract

- status: success
- executive_summary: Added headless application orchestration over injected adapter ports. It deterministically aggregates per-agent status, preflights every selected adapter before writes, returns JSON-ready plans, delegates all configuration semantics to adapters, and compensates prior mutations on later failure while preserving the original stable error.
- artifacts: `internal/app/{agent,agent_test}.go`; `internal/app/contract.go`; `openspec/changes/docmanager-installer-agent-integration/{tasks,apply-progress}.md`
- next_recommended: sdd-apply for explicitly assigned tasks 4.1–4.3 only.
- risks: No CLI registration, TUI, Gentle AI runtime/ownership dependency, acceptance/native CI fix, documentation, commit, push, or merge was added. Composition must inject the five approved adapter ports in a later boundary.
- skill_resolution: paths-injected — sdd-apply, work-unit-commits, go-testing, chained-pr.
- implementation_mode: Strict TDD.
- workload_boundary: approved feature-branch-chain Work Unit 9 child slice on `feature/docmanager-installer-agent-orchestration`, based on the tracker after Work Unit 8; no commit, push, merge, PR, acceptance, native CI, or documentation action.
- native_token: retained by parent.
- native_settlement_result: complete
- changed_line_evidence: 375 product/test/contract/task lines plus 42 OpenSpec evidence lines = 417 total; `417 < 800` native work-unit budget.
- evidence_revision: `sha256:13610f7161aa1a8bfa9596af39f62dd7d135929456c8a75594908d9cc55fb22b` — SHA-256 of the ordered per-file SHA-256 manifest for Work Unit 9 code, tests, contract addition, and `tasks.md`; excludes this self-describing progress file.

### Aggregate Completion Status

22/25 tasks complete. `tasks.md` confirms tasks 1.1–1.4, 2.1–2.10, and 3.1–3.8 are checked. Tasks 4.1–4.3 remain pending. Ready for the next assigned batch; not ready for verification.

## Work Unit 10 — Final Runtime Acceptance (tasks 4.1–4.2)

Completed tasks: 4.1, 4.2. Work Units 1–9 and their evidence above remain unchanged.

### TDD Cycle Evidence

| Task | Test File | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|---|---|---|---|---|---|---|---|
| 4.1 | `cmd/docmanager/acceptance_test.go` | Integration/static | `go test ./internal/app ./cmd/docmanager -count=1` — exit 0 before changes | `go test ./cmd/docmanager -run 'TestAcceptance' -count=1` — FAIL: Windows workflow did not build `docmanager.exe` | Same command — exit 0; six acceptance tests PASS | Headless plan/status/doctor; symlink/traversal rejection; MCP+receipt+SQLite; adapter composition; macOS lexical path; Windows build/invocation | `gofmt`; focused tests remained PASS |
| 4.2 | `internal/app/lifecycle.go`, `.github/workflows/ci.yml` | Integration/static | Existing lifecycle and CLI baseline above | 4.1 acceptance regression preceded canonical-path and workflow production changes | Focused acceptance and lifecycle commands — exit 0 | Physical Git-root equivalence versus final symlink/traversal rejection; matrix-selected Unix/Windows outputs | Minimal `EvalSymlinks` comparison and explicit output matrix; tests remained PASS |

### Work Unit Evidence

| Evidence | Result |
|---|---|
| Focused test command and exact result | `go test ./cmd/docmanager -run 'TestAcceptance' -count=1` — exit 0; six acceptance tests PASS. `go test ./internal/app -run 'TestLifecycle|TestWorkspace' -count=1` — exit 0. |
| Runtime harness command/scenario and exact result | Isolated `mktemp` repository plus isolated `HOME`, `XDG_STATE_HOME`, and `GIT_CONFIG_*`; native Linux binary ran workspace status, JSON dry-run, doctor, install, doctor, and uninstall. Exit 0; state was absent after dry-run and uninstall; temporary root was removed and confirmed absent. MCP acceptance used a temporary Git repo and foreground stdio server; it created/verified a receipt in the repository-local SQLite ledger without changing the staged Git state. |
| Regression and quality commands | `go test ./... -count=1`, `go test -race ./... -count=1`, `go vet ./...`, `go build ./cmd/docmanager`, `test -z "$(gofmt -l cmd internal spike)"`, and `git diff --check` — all exit 0. The direct build artifact was removed afterward. |
| Cross-build checks | CGo-free, trimpath, buildvcs-disabled Linux/amd64, Darwin/arm64, and Windows/amd64 (`.exe`) builds — exit 0; every temporary artifact was nonempty and cleanup passed. |
| Native settlement | Linux package smoke: complete. Darwin and Windows native execution: unavailable on this Linux executor; no emulation was attempted. Pending external evidence: rerun `native-package-smoke` on `macos-latest` and `windows-latest` CI. |
| Rollback boundary | Revert `cmd/docmanager/acceptance_test.go`, the canonical physical-root comparison in `internal/app/lifecycle.go`, the `native-package-smoke` output matrix in `.github/workflows/ci.yml`, and only the 4.1–4.2 task/progress entries. MCP tools, receipt format, SQLite ledger behavior, agent adapters, and documentation remain otherwise unchanged. |
| Cleanup/process evidence | Every harness used `t.TempDir()` or `mktemp -d /tmp/opencode/...`; isolated HOME/XDG paths were never user paths. MCP and CLI subprocesses were foreground and closed before return. Temporary repositories, binaries, cross-build directories, locks, and direct-build output were removed. |

### Work Unit 10 Apply Result Contract

- status: success
- executive_summary: Added behavior-first final acceptance coverage for headless CLI contracts, managed-agent composition, MCP receipt/SQLite preservation, path safety, macOS lexical/physical Git roots, and Windows package smoke agreement. Canonical root comparison now compares resolved physical paths only after rejecting a final symlink target or traversal; CI selects the executable name from its OS matrix.
- artifacts: `cmd/docmanager/acceptance_test.go`; `internal/app/lifecycle.go`; `.github/workflows/ci.yml`; `openspec/changes/docmanager-installer-agent-integration/{tasks,apply-progress}.md`
- next_recommended: sdd-apply for task 4.3 only, then sdd-verify after its completion
- risks: Native macOS and Windows runtime execution is pending CI evidence; local cross-build and static workflow agreement are not a substitute for those runners.
- skill_resolution: paths-injected — sdd-apply, work-unit-commits, go-testing, chained-pr.
- implementation_mode: Strict TDD.
- workload_boundary: approved feature-branch-chain Work Unit 10 child slice based on tracker after Unit 9; no commit, push, merge, publication, documentation, or user configuration mutation.
- native_token: retained by parent.
- native_settlement_result: partial_external_evidence_pending
- changed_line_evidence: 195 new acceptance-test lines + 11 production/workflow changes + 4 task checkbox changes + this evidence entry; under the 800-line work-unit budget.
- evidence_revision: `sha256:208e4bde8f5f277a48c0cfc5c299e45f25095c4fc18c1bc1aa11fab12a7d6c9c` — SHA-256 of the ordered per-file SHA-256 manifest for the acceptance test, lifecycle/workflow fixes, and `tasks.md`; excludes this self-describing progress file.

### Aggregate Completion Status

24/25 tasks complete. `tasks.md` confirms tasks 1.1–1.4, 2.1–2.10, 3.1–3.8, and 4.1–4.2 are checked. Task 4.3 documentation remains pending. Ready for the final documentation batch; not ready for verification.
