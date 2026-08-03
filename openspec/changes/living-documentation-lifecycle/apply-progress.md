# Apply Progress: Living Documentation Lifecycle

## Result Contract

- Outcome: passed
- Evidence revision: `sha256:48b14af4d24eb684bf78d40af054a097c53e52bfcd457460eb805bc2947c3daf`
- Runtime acquire token: `sha256:6c86f3c533bc4a75f0be92d7a96f45ccf9ecb312226792c67e479a66b505ec5d`
- Diagnosis: Unit 1 domain transitions and transactional lifecycle persistence passed all required evidence.
- Harness disposition: reused
- Cleanup evidence: no processes started; temporary SQLite workspaces were created by `t.TempDir()` and cleaned by Go tests.
- Process evidence: focused packages, runtime lifecycle harness, and `go test ./...` exited 0.

## Strict-TDD Cycle Evidence

| Task | Test file | Layer | Safety net | RED | GREEN | Triangulate | Refactor |
|---|---|---|---|---|---|---|---|
| 1.1 | `internal/domain/lifecycle_test.go` | Unit | N/A (new) | build failure: undefined lifecycle API | pass: 2 tests | transition + six invalidation causes | gofmt; pass |
| 1.2 | `internal/adapters/sqlite/lifecycle_test.go` | Integration | N/A (new) | build failure: undefined lifecycle adapter | pass: 2 tests | migration, rollback, replay, divergence | gofmt; pass |
| 1.3 | `internal/domain/lifecycle_test.go` | Unit | N/A (new) | lifecycle API absent | pass | typed valid/invalid transitions | gofmt; pass |
| 1.4 | `internal/adapters/sqlite/lifecycle_test.go` | Integration | N/A (new) | lifecycle adapter absent | pass | transaction failure and idempotency branches | gofmt; pass |

## Work Unit Evidence

| Evidence | Exact result |
|---|---|
| Focused test | `go test ./internal/domain ./internal/adapters/sqlite` — exit 0; 2 packages passed. |
| Runtime harness | `go test ./internal/adapters/sqlite -run Lifecycle -count=1 -v` — exit 0; 2 lifecycle tests passed. |
| Full suite | `go test ./...` — exit 0; 9 tested packages passed, 1 package had no test files. |
| RED domain | `go test ./internal/domain -run Lifecycle -count=1` — exit 1; undefined lifecycle API. |
| RED SQLite | `go test ./internal/adapters/sqlite -run Lifecycle -count=1` — exit 1; undefined lifecycle adapter/domain API. |
| Format/check | `gofmt -d` on four touched Go files and `git diff --check` — exit 0, no output. |
| Rollback boundary | Remove only `internal/domain/lifecycle*.go` and `internal/adapters/sqlite/lifecycle*.go`; no existing ledger behavior changes. |

## Delivery and Risk

- Chain: feature-branch-chain; PR 1 base is `feat/living-documentation-lifecycle-tracker`; the tracker targets `develop`.
- Boundary: Unit 1 only — typed domain lifecycle and `.docmanager/lifecycle.db`; Units 2 and 3 are untouched.
- Changed-line accounting: 329 new Go lines + 8 task checkbox lines + 45 progress lines = 382 authored additions/deletions, within the 400-line ceiling.
- Risks: `Lifecycle` is intentionally persistence-only; application/MCP integration remains for later units. `ledger.go` is unchanged.
- Skill resolution: paths-injected — sdd-apply, strict-tdd, go-testing, work-unit-commits, chained-pr, shared phase protocol, and OpenSpec convention.

## Unit 1 Hardening Final Registration (native evidence)

This final registration preserves all prior attempts, corrections, interruptions, and hardening history. It records evidence only; the parent owns settlement.

- Native final-registration acquire state: `proceed`.
- Runtime acquire token: `sha256:cf0a14d59f10a4977ab21db1b357cb6f93e91065e5374990c081c06c30f8521d`.
- Work unit: `unit-1-hardening-final-registration`.
- Functional change in this objective: 0 Go source/test lines; no functional code or tests were changed.
- Native hardening attempt: 66 changed lines, exceeding its 60-line forecast; the prior 50-line actor claim is not authoritative.
- Delivery: PR 1b correction child only, with maintainer-approved `size:exception`; no Unit 2/3 work is included.

### Artifact and Task Reconciliation

- Tasks 1.1–1.8 are checked in `tasks.md`; Unit 2 (2.1–2.6) and Unit 3 (3.1–3.5) remain unchecked.
- `tasks.md` was not edited because no checkbox was objectively wrong.
- This objective edited only this evidence-registration section in `apply-progress.md`.

### Unchanged Source/Test Identities

| File | Git blob hash before verification | Git blob hash after verification |
|---|---|---|
| `internal/domain/lifecycle.go` | `8c29da58b4657463b61d3aa9b314d341ee1f0e89` | `8c29da58b4657463b61d3aa9b314d341ee1f0e89` |
| `internal/domain/lifecycle_test.go` | `4af5265e7e95a1c599bf4ffa01d7e946ae3c7c1f` | `4af5265e7e95a1c599bf4ffa01d7e946ae3c7c1f` |
| `internal/adapters/sqlite/lifecycle.go` | `772f5128a1a0bcf27012038fcd86bf6c3de51e1b` | `772f5128a1a0bcf27012038fcd86bf6c3de51e1b` |
| `internal/adapters/sqlite/lifecycle_test.go` | `fb84692dea455a3d7dad07463d221599350b77f0` | `fb84692dea455a3d7dad07463d221599350b77f0` |

Identities match before and after the bounded verification set.

### Bounded Verification Set

| Command | Exact result |
|---|---|
| `go test ./internal/domain ./internal/adapters/sqlite` | exit 0; 2 packages passed. |
| `go test ./internal/adapters/sqlite -run Lifecycle -count=1 -v` | exit 0; 4 lifecycle tests passed, including 3 unsafe-path subtests. |
| `go test ./...` | exit 0; 9 tested packages passed; `assets` had no test files. |
| `gofmt -d internal/domain/lifecycle.go internal/domain/lifecycle_test.go internal/adapters/sqlite/lifecycle.go internal/adapters/sqlite/lifecycle_test.go` | exit 0; no output. |
| `git diff --check` | exit 0; no output. |
| `git diff --numstat origin/feat/living-documentation-lifecycle-01-domain-db` | exit 0; 402 additions + 77 deletions = 479 changed lines: `81/22`, `113/5`, `72/14`, `43/11`, `59/0`, `34/25` by listed diff order. |

### Result Contract

- Outcome: passed.
- Evidence revision: `sha256:ca6a6d8c98794dc1b68f478ef89cbec5d0b8e49fc647fbab658823b1a02d2c26` (canonical final-registration evidence payload).
- Diagnosis: unchanged hardened Unit 1 candidate passed the bounded native verification set; native/OpenSpec line evidence is reconciled without repeating the incorrect 50-line claim.
- Harness disposition: reused; no process was started.
- Cleanup evidence: Go test temporary workspaces are test-owned `t.TempDir()` resources and were cleaned by the test runtime; no durable state was created by this objective.
- Process evidence: all six bounded commands exited 0; source/test blob identities remained identical.
- Current-attempt changed-line count: 0 functional Go source/test lines; this evidence-only registration is within the 100-line ceiling.
- Settlement: not claimed; parent settles once.

### Remaining Scope

- [ ] Unit 2 remains unimplemented.
- [ ] Unit 3 remains unimplemented.

## Unit 1 Hardening (PR 1b; Strict TDD)
- Native correction baseline: 370; hardening incremental candidate: 50; native candidate: 420.
- Evidence revision: `sha256:7a4fb92c651edb6517bc4a27507cf8a9ba299d715c914fdb0ed5b0eb0576977f`.
### TDD Cycle Evidence
| Task | RED | GREEN | Refactor |
|---|---|---|---|
| 1.5 | empty Operation failed | domain pass | gofmt pass |
| 1.6 | key mismatch failed | NUL collision pass | gofmt pass |
| 1.7 | 1.5/1.6 RED | minimal validation/encoding | gofmt pass |
| 1.8 | N/A (evidence-only) | evidence below | N/A |
### Work Unit Evidence
| Evidence | Exact result |
|---|---|
| Focused | `go test ./internal/domain ./internal/adapters/sqlite` — exit 0; 2 packages. |
| Runtime | `go test ./internal/adapters/sqlite -run Lifecycle -count=1 -v` — exit 0; 4 tests, 3 unsafe-path subtests. |
| Full/check | `go test ./...`; `gofmt -d` touched Go files; `git diff --check` — exit 0. |
| Rollback | Revert only `internal/domain/lifecycle.go`, `internal/domain/lifecycle_test.go`, `internal/adapters/sqlite/lifecycle.go`, `internal/adapters/sqlite/lifecycle_test.go`, `openspec/changes/living-documentation-lifecycle/tasks.md`, and the complete cumulative `openspec/changes/living-documentation-lifecycle/apply-progress.md` artifact; no Unit 2/3 behavior. |
- `size:exception` remains maintainer-approved only for PR 1b; its boundary is provenance/idempotency hardening.

## Remaining Tasks

- [ ] 2.4–2.6 Unit 2a2 Git identity/read-only safety.
- [ ] 3.1–3.5 Catalog, audit, verification, and final integration evidence.
## Unit 2a1 Final Line-Neutral Correction: Repository Inventory & Classification (Strict TDD)
- Outcome: passed; prior correction receipt `git diff --numstat 5b962489a08d0b301e0ee21c61cfea5afb0ec8d0 2b6354dd2cbe73adb90c106e8bf0f39ea48914c2` = 215 lines under explicit 220-line authority; final work unit `unit-2a1-final-line-neutral-correction` is 120/120 lines from baseline `2b6354dd2cbe73adb90c106e8bf0f39ea48914c2`; identity binds emitted order/every semantic field and 2a2 remains excluded.
| Task | Test file/layer | Safety net | RED | GREEN / triangulate | Refactor |
|---|---|---|---|---|---|
| 2.1 | `resolver_test.go` / integration | `go test ./internal/adapters/git -count=1` — exit 0 | exact order assertions: exit 1 | focused exit 0; 4 top-level tests, 21 subtests | gofmt exit 0 |
| 2.2 | `resolver.go` / integration | same | same RED | focused exit 0; order plus all fields bind | compact serialization; gofmt exit 0 |
| 2.3 | OpenSpec evidence | N/A | N/A | resolver/compatibility/full/check/accounting exit 0 | N/A |
### Work Unit Evidence
| Evidence | Exact result |
|---|---|
| Focused / runtime | `go test ./internal/adapters/git -run 'TestResolverRadiographsDocumentationWithExplicitExclusions|TestResolverRadiographSkipsIgnoredUntrackedDocumentation|TestRadiographyIdentityBindsSemanticFields|TestRadiographyExclusionPrecedenceAndSafety' -count=1 -v` — exit 0; 4 top-level tests, 21 subtests; first two use real temporary Git repositories. |
| Resolver / compatibility | `go test ./internal/adapters/git -count=1`; `go test ./internal/app ./internal/adapters/mcp -count=1` — all exit 0; resolver, document-change/pre-push, MCP receipt and stdio compatibility packages passed. |
| Full / check-only | `go test ./...` — exit 0; 9 packages passed and `assets` had no test files. `gofmt -l internal/adapters/git/resolver.go internal/adapters/git/resolver_test.go` and `git diff --check` — exit 0/no output. |
| Rollback | Revert only Unit 2a1 changes in `internal/adapters/git/resolver.go`, `internal/adapters/git/resolver_test.go`, `openspec/changes/living-documentation-lifecycle/tasks.md`, and this cumulative `apply-progress.md`; document-change, receipt, pre-push, 2a2, and later units remain intact. |
- Chain: feature-branch-chain; `feat/living-documentation-lifecycle-02-radiography` targets PR #6 `a051667`; 2a2 targets this branch.
- No `size:exception`; `git diff --numstat 2b6354dd2cbe73adb90c106e8bf0f39ea48914c2` = 58 additions + 62 deletions = 120, while `git diff --numstat a051667` = 365 additions + 30 deletions = 395 final PR lines.
## PR 1d Replay Integrity (Strict TDD)
### Result Contract

- Outcome: passed.
- Runtime acquire token: `sha256:12a3860a88530930f0871019c771528ac73e00463ebb99341cc2d31d48d2cee4` (parent-owned; no reset, acquisition, or settlement was performed).
- Work unit: `unit-1d-legacy-replay-integrity`; maximum actor delta: 350 lines.
- Diagnosis: migrated legacy keys are refused by a typed domain error before provenance validation and `BEGIN IMMEDIATE`; fresh v2 keys retain exact non-mutating replay.
- Harness disposition: reused; no process was started.
- Cleanup evidence: SQLite workspaces are test-owned `t.TempDir()` resources; no durable test state was created.

### Strict-TDD Cycle Evidence

| Task | Test file | Layer | Safety net | RED | GREEN | Triangulate | Refactor |
|---|---|---|---|---|---|---|---|
| 1.13 | `internal/domain/lifecycle_test.go`, `internal/adapters/sqlite/lifecycle_test.go` | Unit + integration | `go test ./internal/domain ./internal/adapters/sqlite` — exit 0; 2 packages | focused command — exit 1; undefined typed state/error symbols | focused command — exit 0; domain 1 test, SQLite 2 tests | both typed states; both migrated keys; repeated, malformed, and mismatched provenance | gofmt; focused command exit 0 |
| 1.14 | `internal/adapters/sqlite/lifecycle_test.go` | Integration | same 2-package baseline | same missing-symbol RED gate | focused command — exit 0; exact v2 result, non-null record link, available state, no replay mutation | legacy refusal snapshots tables/schemas/counts/values/version; v2 replay snapshot | snapshot helper; lifecycle suite exit 0 |
| 1.15 | `internal/domain/lifecycle.go`, `internal/adapters/sqlite/lifecycle.go` | Integration | same 2-package baseline | same missing-symbol RED gate | focused command — exit 0 | valid v2 replay and divergent existing behavior remain covered by lifecycle suite | gofmt; lifecycle suite exit 0 |
| 1.16 | OpenSpec artifacts | Evidence | N/A | N/A | verification/accounting below | N/A | N/A |

### Work Unit Evidence

| Evidence | Exact result |
|---|---|
| Focused test | `go test ./internal/domain ./internal/adapters/sqlite -run 'Test(IdempotencyReplayStateContract|LifecycleRefusesEveryMigratedLegacyKeyWithoutMutation|LifecycleReplaysAvailableV2KeyExactlyOnce)' -count=1 -v` — exit 0; 1 domain test and 2 SQLite tests passed; legacy test ran 5 refusal cases. |
| Runtime harness | `go test ./internal/adapters/sqlite -run Lifecycle -count=1 -v` — exit 0; 8 lifecycle tests passed, including migrated v1 fixture refusal/replay paths. |
| Domain/SQLite | `go test ./internal/domain ./internal/adapters/sqlite` — exit 0; 2 packages passed. |
| Full suite | `go test ./...` — exit 0; 9 tested packages passed; `assets` had no test files. |
| Check-only | `gofmt -l internal/domain/lifecycle.go internal/domain/lifecycle_test.go internal/adapters/sqlite/lifecycle.go internal/adapters/sqlite/lifecycle_test.go` — exit 0/no output; `git diff --check` — exit 0/no output. |
| Rollback boundary | Revert only PR 1d changes in `internal/domain/lifecycle.go`, `internal/domain/lifecycle_test.go`, `internal/adapters/sqlite/lifecycle.go`, `internal/adapters/sqlite/lifecycle_test.go`, and this unit's `tasks.md`/cumulative `apply-progress.md` updates. This removes typed replay refusal and its proof, while preserving PR 1c's migration/rollback foundation and leaving Units 2/3 untouched. |

### Delivery and Remaining Scope

- Chain: feature-branch-chain; PR 1d targets `fix/living-documentation-lifecycle-01c-v1-v2-migration`, never `develop`.
- Boundary: typed replay-state/error contract, pre-validation/pre-transaction legacy refusal, and v2 replay proof only.
- Git accounting against PR 1c base `a467731`: 184 additions + 9 deletions = 193 changed lines (`13/2` SQLite adapter, `105/0` SQLite tests, `11/3` domain, `12/0` domain tests, `39/0` progress, `4/4` tasks); within the 350-line actor delta and 400-line PR ceilings.
- No size exception. Units 2 and 3 remain pending; this is not verification of the entire SDD change.
- Skill resolution: direct executor reads of `sdd-apply`, `strict-tdd`, `go-testing`, `work-unit-commits`, and `chained-pr`; CodeGraph was unavailable because this worktree is unindexed.

## PR 1c v1→v2 Migration (Strict TDD)
- Result Contract: outcome passed; native token `sha256:fab48346d46330902677709d44a18105275f32c23e5921170709d58868ec1bfe` is parent-settled only.
- Scope: migration foundation only; no production RED occurred. PR 1c is publishable only as a chained review slice, not mergeable/deployable to `develop`; PR 1d replay integrity remains required and unchecked.
| Task | Test file | Layer | Safety net | RED | GREEN | Triangulate | REFACTOR |
|---|---|---|---|---|---|---|---|
| 1.9 | `internal/adapters/sqlite/lifecycle_test.go` | Integration | `go test ./internal/adapters/sqlite -run Lifecycle -count=1 -v` exit 0; 6 tests | Inherited candidate behavior; new exact fixture assertions passed before code re-scope | focused migration test exit 0; 2 tests | two provenance/idempotency rows | snapshot helper |
| 1.10 | same | Integration | same | Characterization: initial test build failure was test-only; production candidate passed after correction | focused migration test exit 0; 2 tests | all four schemas and every original row | snapshot helper |
| 1.11 | `internal/adapters/sqlite/lifecycle.go` | Integration | same | Inherited transaction already green; no fabricated RED | lifecycle suite exit 0; 6 tests | fresh v2 and forced rollback paths | no further refactor |
| 1.12 | artifacts | Evidence | N/A | N/A | full/check/accounting passed | N/A | N/A |
| Evidence | Exact result |
|---|---|
| Focused test | `go test ./internal/adapters/sqlite -run 'TestLifecycle(MigratesCompleteHistoricalV1\|V2MigrationRollsBackHistoricalSchemaAndData)' -count=1 -v` — exit 0; 2 tests passed. |
| Runtime harness | `go test ./internal/adapters/sqlite -run Lifecycle -count=1 -v` — exit 0; exact v1 fixture→migration and injected rollback, 6 tests passed. |
| Domain/SQLite and full | `go test ./internal/domain ./internal/adapters/sqlite`; `go test ./...` — exit 0; 2 packages, then 9 tested packages passed and `assets` had no test files. |
| Check/accounting | `gofmt -d` touched SQLite files; `git diff --check` — exit 0/no output; base diff is 313 additions + 75 deletions = 388 lines. |
| Rollback boundary | Revert only `internal/adapters/sqlite/lifecycle.go`, `internal/adapters/sqlite/lifecycle_test.go`, `openspec/changes/living-documentation-lifecycle/apply-progress.md`, `openspec/changes/living-documentation-lifecycle/design.md`, `openspec/changes/living-documentation-lifecycle/specs/documentation-catalog-lifecycle/spec.md`, and `openspec/changes/living-documentation-lifecycle/tasks.md`; no domain replay/error behavior or Unit 2/3 work. |

## Unit 1 Conformance-Correction (Strict TDD; attempt 1)

This section supplements and preserves the original Unit 1 attempt and settlement evidence above; it does not settle this correction.

### Result Contract

- Outcome: passed
- Evidence revision: `sha256:9d3fd816989f932b8aecd8c3e650970e72e02a5493fcbe8bf7f7167e27fcdf7d`
- Original runtime reference: `sha256:6c86f3c533bc4a75f0be92d7a96f45ccf9ecb312226792c67e479a66b505ec5d`
- Correction acquire token: `sha256:81150f8688c974213bc9a3adea39eae2063271406463563ac40c3e39083e4c36` (`state: proceed`)
- Diagnosis: corrected Unit 1 authorization binding, typed lifecycle foundations, SQLite safety/migrations, complete declared-local provenance, and idempotency replay semantics.
- Harness disposition: reused; no processes started.
- Cleanup evidence: `t.TempDir()` SQLite workspaces were test-owned and cleaned by Go; no durable workspace state was created.
- Process evidence: focused, runtime, full, format, and diff checks exited 0.

### Strict-TDD Cycle Evidence

| Task | Test file | Layer | Safety net | RED | GREEN | Triangulate | Refactor |
|---|---|---|---|---|---|---|---|
| 1.1 / 1.3 | `internal/domain/lifecycle_test.go` | Unit | `go test ./internal/domain ./internal/adapters/sqlite` — 2 packages exit 0 | `go test ./internal/domain -run Lifecycle -count=1` — exit 1; missing completion/provenance/typed-state API | same command — exit 0; 2 lifecycle cases | batch, ordered actions, six drift bindings, inactive/completed states, typed states/results | gofmt; focused test exit 0 |
| 1.2 / 1.4 | `internal/adapters/sqlite/lifecycle_test.go` | Integration | same focused baseline — exit 0 | `go test ./internal/domain ./internal/adapters/sqlite -run Lifecycle -count=1` — exit 1; missing provenance read/API | same command — exit 0; 4 lifecycle cases | symlink/nonregular/unsafe modes; v1→v2 migration and rollback; replay/divergence | gofmt; runtime harness exit 0 |

### Work Unit Evidence

| Evidence | Exact result |
|---|---|
| Focused test | `go test ./internal/domain ./internal/adapters/sqlite` — exit 0; 2 packages passed. |
| Runtime harness | `go test ./internal/adapters/sqlite -run Lifecycle -count=1 -v` — exit 0; 4 lifecycle tests passed, including 3 unsafe-path subtests. |
| Full suite | `go test ./...` — exit 0; 9 tested packages passed and `assets` reported no test files. |
| Final check-only | `gofmt -d` on the four touched Go files and `git diff --check` — exit 0 with no output. |
| Rollback boundary | Revert only `internal/domain/lifecycle.go`, `internal/domain/lifecycle_test.go`, `internal/adapters/sqlite/lifecycle.go`, `internal/adapters/sqlite/lifecycle_test.go`, `openspec/changes/living-documentation-lifecycle/tasks.md`, and the complete cumulative `openspec/changes/living-documentation-lifecycle/apply-progress.md` artifact; `ledger.go` and all Unit 2/3 paths remain untouched. |

### Delivery and Remaining Risk

- PR boundary: feature-branch-chain correction child, base `feat/living-documentation-lifecycle-01-domain-db`; Unit 2 must base on this correction.
- Incremental candidate changed-line count: 329 Go lines plus 40 Unit 1 OpenSpec evidence lines = 369, below the 400-line ceiling.
- Ownership boundary: portability permits proof of non-symlink directories and no group/world write bits; this implementation deliberately does not claim UID ownership validation.
- Remaining risk: no Unit 2/3 use case, MCP tool, catalog behavior, audit, radiography, planning, or `VerifyOutcome` wiring is implemented.
- Skill resolution: paths-injected — sdd-apply, strict-tdd, go-testing, work-unit-commits, chained-pr, shared phase protocol, and OpenSpec convention.

## Unit 2a2.1a Safe Git Execution & Root Binding (Strict TDD)
### Result Contract
- Outcome: passed; native token `sha256:fb224e6a938a2c1e6b8f9342ad689d597fb377d95afac8d3b0f8a15608377e49` was neither acquired, reset, nor settled.
- Work unit: `unit-2a2-1a-safe-git-root-binding`; correction actor limit: 180 lines; historical actor receipt from baseline tree `3aeb41751f67bbaf3b8227b01f426a10f991cc91` to receipt tree `fed031ae79c9d66d058e4ec8ad5fe7d8bb08fe45` is 271 changed lines.
- Portable correction churn from receipt tree `fed031ae79c9d66d058e4ec8ad5fe7d8bb08fe45` to pre-evidence-correction code/test candidate tree `15166a7275f20ea2eda5d9fefd3fb10317c408e6` is 62 additions + 25 deletions = 87 changed lines.
- Diagnosis: every Resolver/Radiograph Git invocation crosses `run`, which clears inherited environment and fixes safe config/arguments; `Lstat` plus `SameFile` revalidates the bound root before/after commands and rejects a replacement symlink to the original inode. A swap-and-restore wholly during one Git command is explicitly outside this portable local-tool model.
### Strict-TDD Cycle Evidence
| Task | Test file/layer | Safety net | RED | GREEN / triangulate | Refactor |
|---|---|---|---|---|---|
| 2.4 | `resolver_test.go` / real-Git integration | `go test ./internal/adapters/git -count=1` — exit 0 | symlink replacement during `ls-files` returned nil instead of `ErrOutsideRepository` | exit 0: relative, traversal, dot, subdir, symlink, renamed-root, and replacement-symlink failures | root binding helper; gofmt exit 0 |
| 2.5 | `resolver_test.go` / boundary integration | same | proxy test failed: credential/helper-safe `-c` arguments were absent | exit 0: range, initial, staged, worktree, inventory, and digest operations record exact args and isolated env | one centralized boundary; gofmt exit 0 |
| 2.6 | OpenSpec evidence | N/A | N/A | focused, resolver, compatibility, full, format/check, and accounting pass | N/A |
### Work Unit Evidence
| Evidence | Exact result |
|---|---|
| Focused test / runtime harness | `go test ./internal/adapters/git -run 'TestResolver(RadiographUsesReadOnlyGitCommands|RejectsRootReplacementBeforeEvidence)$' -count=1 -v` — exit 0; 2 top-level tests. The proxy covers range, initial, staged, worktree, inventory, and digest calls; replacement-symlink returns no evidence. |
| Resolver / compatibility | `go test ./internal/adapters/git -count=1`; `go test ./internal/app ./internal/adapters/mcp -count=1` — exit 0; all three packages passed. |
| Full / check-only | `go test ./...` — exit 0; 9 tested packages passed and `assets` had no test files. `gofmt -l internal/adapters/git/resolver.go internal/adapters/git/resolver_test.go` and `git diff --check` — exit 0/no output. |
| Safe-boundary proof | Exact proxy records the fixed `-c` list and complete controlled environment; poisoned injected config/Git directory variables are absent. Repository-local included fsmonitor/filter helpers do not execute; used plumbing has no hook invocation path. |
| Rollback | Revert only `internal/adapters/git/resolver.go`, `internal/adapters/git/resolver_test.go`, `openspec/changes/living-documentation-lifecycle/design.md`, `openspec/changes/living-documentation-lifecycle/tasks.md`, and this cumulative `apply-progress.md` section; this removes safe Git/root binding without changing Unit 2a1 inventory/classification. |
### Delivery and Deferred Scope
- Feature-chain slice: `fix/living-documentation-lifecycle-02a2-git-safety` → `feat/living-documentation-lifecycle-02-radiography`; no `size:exception`; pre-evidence-correction code/test candidate tree `15166a7275f20ea2eda5d9fefd3fb10317c408e6` against `878aa1f` is 275 additions + 83 deletions = 358 changes across the final five-file rollback boundary, within the 400-line maximum. Growth from the old 321-line publication receipt to the corrected 358-line receipt is 37 lines of net publication receipt growth, not actor or correction churn.
- Tasks 2.7–2.9 retain linked-worktree path resolution only; all snapshot/before-after guarantees are deferred to 2a2.1c.
- Skill resolution: paths-injected — sdd-apply, strict-tdd, go-testing, work-unit-commits, chained-pr, and shared phase protocol; CodeGraph query reported this worktree unindexed.

## Unit 2a2.1b Linked-worktree Path Resolution (Strict TDD)

### Result Contract
- Outcome: passed. The failed combined oracle candidate was split because path resolution is independently reviewable; snapshot identity, sensitivity, special-file/symlink proof, and no-write before/after receipts are deferred to Unit 2a2.1c.
- Parent-owned native token was neither acquired, reset, nor settled. `Radiograph` does not consume this test-owned helper, so path resolution cannot introduce a new runtime failure.

### TDD Cycle Evidence
| Task | Test file / layer | Safety net | RED | GREEN | Triangulate | Refactor |
|---|---|---|---|---|---|---|
| 2.7 | `resolver_test.go` / real-Git integration | `go test ./internal/adapters/git -count=1` — exit 0 | focused command — exit 1: `newlinePaths` return contract and independent linked pointer oracle absent | exit 0: LF/CRLF spaces pass; blank records and stray CR reject | ordinary and linked worktrees | compact framing helper; focused pass |
| 2.8 | `resolver_test.go` / real-Git integration | same | focused command — exit 1: lexical aliases and symlinks were accepted | exit 0: raw absolute alternate aliases reject before cleaning; relative alternates stay green | ordinary + linked + alias/symlink/missing fixtures | compact unexported helper |
| 2.9 | OpenSpec evidence | N/A | N/A | focused/runtime, regressions, compatibility, full, format/check, and accounting pass | N/A | N/A |

### Work Unit Evidence
| Evidence | Exact result |
|---|---|
| Focused test / runtime harness | `go test ./internal/adapters/git -run 'Test(RadiographPathsPreserveFramedPaths|ResolverRadiographPathsResolveOrdinaryAndLinkedWorktrees|ResolverRadiographPathsFailClosed|ResolverRadiographPathsRejectsLexicalAliasesAndSymlinks|ResolverRadiographPathsRejectMissingResolvedPaths)$' -count=1 -v` — exit 0; 5 top-level tests, including absolute alternate alias rejection. Real `git worktree add` covers ordinary and linked roots. |
| Resolver / 2a2.1a regression | `go test ./internal/adapters/git -count=1`; `go test ./internal/adapters/git -run 'TestResolver(RadiographUsesReadOnlyGitCommands|RejectsRootReplacementBeforeEvidence)$' -count=1 -v` — exit 0. |
| Compatibility / full / checks | `go test ./internal/app ./internal/adapters/mcp -count=1`; `go test ./...`; `gofmt -l internal/adapters/git/resolver.go internal/adapters/git/resolver_test.go`; `git diff --check` — all exit 0. |
| Rollback boundary | Revert only `internal/adapters/git/resolver.go`, `internal/adapters/git/resolver_test.go`, `openspec/changes/living-documentation-lifecycle/tasks.md`, and `openspec/changes/living-documentation-lifecycle/apply-progress.md`; preserves 2a2.1a and removes only path-resolution proof. |

### Delivery and Deferred Scope
- Chain: feature-branch-chain child targets `fix/living-documentation-lifecycle-02a2-git-safety`, never `main`; no `size:exception`.
- Pre-correction review-visible child against `f10a07b`: 194 additions + 21 deletions = 215 changed lines across exactly the four rollback files. Fail-closed correction native churn (not PR size): tree `2138a25571de7c1b62a5b0f38c3ab611bf37bada` -> `72219704400780e13c7f8691b3063f6340ae7fa6` = 158 changed lines, explicitly accepted by the maintainer. The 100 lines are net review-visible child growth from 215 to 315, not actor churn; the final child against `f10a07b` is 300 additions + 22 deletions = 322 changed lines. Native split-replanning churn (not PR size): tree `31a61555644b09d9a6ea5fef4525ef9df3876c0d` -> `1b48d6984ce958623b7090abb1167b935b17c790` = 454 changed lines, explicitly accepted by the maintainer because it removed the failed combined oracle candidate.
- Tasks 2.7–2.9 are checked; 2.10–2.12 reserve the complete snapshot oracle, and 2.13 onward remain unchecked. No snapshot/no-write guarantee is claimed in this slice.
- Skill resolution: paths-injected — sdd-apply, strict-tdd, go-testing, work-unit-commits, chained-pr, shared phase protocol, and OpenSpec convention; CodeGraph was unavailable for this worktree.
