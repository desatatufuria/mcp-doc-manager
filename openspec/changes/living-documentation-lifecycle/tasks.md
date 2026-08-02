# Tasks: Living Documentation Lifecycle

## Review Workload Forecast

| Field | Value |
|---|---|
| Changed-line evidence/budget | PR 1c: 388 current; PR 1d: separately budgeted ≤400 |
| 400-line budget risk | High overall; Medium per child |
| Chained PRs recommended | Yes |
| Suggested split | tracker→#3→#4/1b→PR 1c→PR 1d→Unit 2→Unit 3 |
| Delivery strategy | ask-on-risk, resolved by user-selected split |
| Chain strategy | feature-branch-chain |

Decision needed before apply: No
Chained PRs recommended: Yes
Chain strategy: feature-branch-chain
400-line budget risk: High
No `size:exception`: PR 1c and PR 1d must each remain ≤400 changed lines.

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|---|---|---|---|---|---|
| 1 | Domain/DB | #3←tracker | `go test ./internal/domain ./internal/adapters/sqlite` | Lifecycle scenario | domain/DB |
| 1b | Correction | #4/1b←#3 | `go test ./internal/domain` | Identity scenario | domain + tests |
| 1c | Historical v1→v2 migration foundation; no Save replay integration | PR 1c base=PR #4 | `go test ./internal/adapters/sqlite` | exact v1 fixture→migration | migration/schema tests only |
| 1d | Typed replay integrity and terminal proof | PR 1d base=PR 1c | `go test ./internal/domain ./internal/adapters/sqlite` | migrated-key refusal and fresh-v2 replay | replay domain/Save/tests/evidence |
| 2 | Radiography | Unit 2 base=completed PR 1d | `go test ./internal/app ./internal/adapters/git ./internal/adapters/mcp` | Stdio radiograph→plan | service/resolver/MCP |
| 3 | Catalog | Unit 3←Unit 2 | `go test ./internal/app ./internal/adapters/mcp ./internal/adapters/sqlite` | Stdio authorize→edit→verify | catalog/audit/verify |

Feature-chain: tracker→PR #3→PR #4→PR 1c→PR 1d→Unit 2; only tracker merges to `develop`.

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

## Unit 2

- [ ] 2.1 RED: `internal/adapters/git/resolver_test.go` excludes non-doc/generated inputs; reports reason/uncertainty.
- [ ] 2.2 RED: resolver root variants and staged/empty-index/`commit -a` make no writes or state changes.
- [ ] 2.3 GREEN: `internal/adapters/git/resolver.go` read-only discovery, evidence, exclusions, ownership uncertainty.
- [ ] 2.4 RED: `internal/app/lifecycle_service_test.go` radiography, denial, policy/batch, in-plan/blocked actions.
- [ ] 2.5 GREEN: `internal/app/lifecycle_service.go` bounded plan/batch/policy and local provenance; no authoring.
- [ ] 2.6 RED/GREEN: `internal/adapters/mcp/{mcp_test.go,mcp.go}` staged tools, ownership conflict, no visible writes.

## Unit 3

- [ ] 3.1 RED: `internal/app/lifecycle_service_test.go` imports/provenance, stale/uncertain/orphan, no visible mutation.
- [ ] 3.2 RED: same test covers evidenced update/review/orphan/conflict/no-action, never truthfulness.
- [ ] 3.3 RED: `VerifyOutcome` rejects inactive/out-of-plan/stale/revision-scope-baseline mismatch.
- [ ] 3.4 GREEN: `internal/app/lifecycle_service.go`/`internal/domain/lifecycle.go` catalog/audit/orphan/verification; idempotent MCP.
- [ ] 3.5 REFACTOR/verify: focused/stdio/`go test ./...`; runtime/rollback per commit.
