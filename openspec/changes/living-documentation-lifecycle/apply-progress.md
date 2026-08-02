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

- [ ] 2.1–2.6 Radiography, planning, approvals, and staged MCP operations.
- [ ] 3.1–3.5 Catalog, audit, verification, and final integration evidence.

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
