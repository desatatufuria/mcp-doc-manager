# Tasks: Living Documentation Lifecycle

## Review Workload Forecast

| Field | Value |
|---|---|
| Changed-line budget | Historical slices recorded; each Unit 2 child ≤400 additions+deletions |
| 400-line budget risk | High overall; Low per Unit 2 child |
| Chained PRs recommended | Yes |
| Suggested split | tracker→#3→#4/1b→1c→1d→#6→2a1→2a2→2b→2c→3 |
| Delivery strategy | ask-on-risk, resolved by maintainer-approved split |
| Chain strategy | feature-branch-chain |

Decision needed before apply: No
Chained PRs recommended: Yes
Chain strategy: feature-branch-chain
400-line budget risk: High
No `size:exception`: every Unit 2 child remains ≤400 additions+deletions.

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|---|---|---|---|---|---|
| 1 | Domain/DB | #3←tracker | `go test ./internal/domain ./internal/adapters/sqlite` | Lifecycle scenario | recorded Unit 1 files/behavior |
| 1b | Correction | #4/1b←#3 | `go test ./internal/domain` | Identity scenario | recorded Unit 1b files/behavior |
| 1c | Migration foundation | base=PR #4 | `go test ./internal/adapters/sqlite` | v1 fixture→migration | migration/schema tests only |
| 1d | Replay integrity | base=PR 1c | `go test ./internal/domain ./internal/adapters/sqlite` | migrated refusal; fresh-v2 replay | replay domain/Save/tests/evidence |
| 2a1 | Inventory & classification (260–360 lines) | base=PR #6 `a051667` replay-integrity | `go test ./internal/adapters/git` | real temporary Git fixture; helper-only precedence is not runtime evidence | `resolver.go`, `resolver_test.go`, `tasks.md`, `apply-progress.md` |
| 2a2 | Git identity & read-only safety (250–360 lines) | base=2a1 branch | `go test ./internal/adapters/git` | temporary repos: lexical/physical roots, staged/initial/empty-index/`commit -a`/unmerged states; before/after `Radiograph` | identity/snapshot hunks in `internal/adapters/git/{resolver.go,resolver_test.go}` |
| 2b | Planning service (280–380 lines) | base=2a2 branch | `go test ./internal/app` | radiography→bounded plan; denial/no authoring | service/tests only |
| 2c | MCP staging (250–380 lines) | base=2b branch | `go test ./internal/adapters/mcp` | stdio radiography→plan; no visible writes | MCP adapter/tests only |
| 3 | Catalog | base=completed 2c branch | `go test ./internal/app ./internal/adapters/mcp ./internal/adapters/sqlite` | stdio authorize→edit→verify | catalog/audit/verify |

Feature-chain: tracker→#3→#4→1c→1d→#6→2a1→2a2→2b→2c→3; only tracker merges to `develop`. PR 2a1 targets PR #6/current replay-integrity; each later child targets its immediate parent and must be retargeted/rebased if polluted.

## Unit 1

- [x] 1.1 RED: `internal/domain/lifecycle_test.go` transitions, invalidation, local provenance/no-auth claim.
- [x] 1.2 RED: `internal/adapters/sqlite/lifecycle_test.go` ownership, atomic rollback, provenance, replay/divergence.
- [x] 1.3 GREEN: `internal/domain/lifecycle.go` lifecycle states, bounds, catalog/audit/verification.
- [x] 1.4 GREEN: `internal/adapters/sqlite/lifecycle.go` versioned lifecycle/provenance/idempotency DB.

## PR 1b

- [x] 1.5 RED: `internal/domain/lifecycle_test.go` rejects empty `Provenance.Operation` and missing fields.
- [x] 1.6 RED: collision-free idempotency identity; external/provenance mismatch; NUL.
- [x] 1.7 GREEN: `internal/domain/lifecycle.go` provenance and `internal/adapters/sqlite/lifecycle.go` identity/key.
- [x] 1.8 Evidence: focused/runtime/full/check-only; apply-progress 370→exact.

## PR 1c: Migration Foundation

- [x] 1.9 Characterization/proof: `internal/adapters/sqlite/lifecycle_test.go` exact multi-record v1 fixture; assert every provenance/idempotency value survives with NULL links and `legacy_unavailable`.
- [x] 1.10 Characterization/proof: inject migration failure and compare pre/post table schemas, every row/value, and `schema_migrations` version exactly.
- [x] 1.11 GREEN: `internal/adapters/sqlite/lifecycle.go` transactionally rebuilds v1 tables into v2; fresh DB creates v2 directly; no invented association or replay wiring.
- [x] 1.12 Evidence: record focused/runtime/check-only results and exact PR 1c Git accounting; state migration proof only and its isolated rollback boundary.

## PR 1d: Replay Integrity

- [x] 1.13 RED: domain/SQLite lifecycle tests require typed `IdempotencyReplayState` (`available`, `legacy_unavailable`) and every migrated-key refusal before validation, transaction, execution, or audit mutation.
- [x] 1.14 RED: prove a new-v2 `available` key returns its exact replay; migrated refusals leave records and audit unchanged.
- [x] 1.15 GREEN: `internal/domain/lifecycle.go` and `internal/adapters/sqlite/lifecycle.go` use typed state/error and pre-validation, pre-`BEGIN IMMEDIATE` legacy lookup/refusal.
- [x] 1.16 Evidence: cumulative focused/runtime/full/check-only results; truthful exact Git accounting and complete cross-slice rollback proof/boundary.

## Unit 2a1: Repository Inventory & Classification (base=PR #6 `a051667` replay integrity)

- [x] 2.1 RED: `internal/adapters/git/resolver_test.go` inventories tracked plus applicable untracked Markdown/MDX; classifies uncertain and excluded without invented certainty, with evidence/reasons and deterministic ordering.
- [x] 2.2 GREEN: `internal/adapters/git/resolver.go` discovers/classifies paths; precedence is non-doc, symlink, generated/vendor, executable; directory location never establishes maintained status.
- [x] 2.3 Evidence: record exact focused real-Git-fixture runtime result, compatibility/full/check/accounting, and the four-file 2a1-only rollback boundary; do not mark complete until proof passes.

## Unit 2a2: Git Identity & Read-only Safety (base=completed 2a1)

- [ ] 2.4 RED: `internal/adapters/git/resolver_test.go` distinguishes exact/physical roots, lexical duplicate inputs, staged/initial/empty-index/`commit -a`, and unmerged stage-sensitive identities; snapshot before/after `Radiograph`.
- [ ] 2.5 GREEN: `internal/adapters/git/resolver.go` canonicalizes physical roots and stage-sensitive identities, deduplicates normalized roots, and snapshots `Radiograph` state with no writes.
- [ ] 2.6 Evidence: record exact before/after repository-state receipts for every fixture, focused command, line accounting, and the 2a2-only rollback boundary.

## Unit 2b: Planning Service (base=completed 2a2)

- [ ] 2.7 RED: `internal/app/lifecycle_service_test.go` covers radiography, denial, policy/batch, and in-plan/blocked actions.
- [ ] 2.8 GREEN: `internal/app/lifecycle_service.go` creates bounded plan/batch/policy and local provenance; no authoring.

## Unit 2c: MCP Staging (base=completed 2b)

- [ ] 2.9 RED/GREEN: `internal/adapters/mcp/{mcp_test.go,mcp.go}` stages tools, ownership conflicts, stdio radiography→plan, no visible writes, and compatibility.

## Unit 3: Catalog (base=completed 2c)

- [ ] 3.1 RED: `internal/app/lifecycle_service_test.go` imports/provenance, stale/uncertain/orphan, no visible mutation.
- [ ] 3.2 RED: same test covers evidenced update/review/orphan/conflict/no-action, never truthfulness.
- [ ] 3.3 RED: `VerifyOutcome` rejects inactive/out-of-plan/stale/revision-scope-baseline mismatch.
- [ ] 3.4 GREEN: `internal/app/lifecycle_service.go`/`internal/domain/lifecycle.go` catalog/audit/orphan/verification; idempotent MCP.
- [ ] 3.5 REFACTOR/verify: focused/stdio/`go test ./...`; runtime/rollback per commit.
