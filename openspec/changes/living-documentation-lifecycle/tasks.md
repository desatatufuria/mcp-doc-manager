# Tasks: Living Documentation Lifecycle

## Review Workload Forecast

| Field | Value |
|---|---|
| Changed-line budget | Each child: ≤400 additions+deletions; first slice is 2a2.1b paths only |
| 400-line budget risk | High overall; Low per child |
| Chained PRs recommended | Yes |
| Suggested split | 2a1 → 2a2.1a → 2a2.1b paths → 2a2.1c oracle → 2a2.2 → 2b → 2c → 3 |
| Delivery strategy | ask-on-risk, resolved by approved chain |
| Chain strategy | feature-branch-chain |

Decision needed before apply: No
Chained PRs recommended: Yes
Chain strategy: feature-branch-chain
400-line budget risk: High
No `size:exception`: every child is capped at 400 additions+deletions.

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|---|---|---|---|---|---|
| 2a1 | Inventory/classification | base=PR #6 | `go test ./internal/adapters/git` | temporary Git fixture | recorded 2a1-only hunks |
| 2a2.1a | Safe Git execution/root binding | PR #7; `fix/living-documentation-lifecycle-02a2-git-safety`, base=PR #7 | `go test ./internal/adapters/git -run 'TestResolver(RadiographUsesReadOnlyGitCommands|RejectsRootReplacementBeforeEvidence)$'` | traversal/subdir/symlink and root-replacement fixtures | root contract, binding, safe-env hunks |
| 2a2.1b | Linked-worktree path resolution | base=completed 2a2.1a branch | `go test ./internal/adapters/git -run 'Test(RadiographPaths|ResolverRadiographPaths)'` | ordinary + linked worktree path fixtures | git-dir/common-dir/index/config/hooks/object path hunks |
| 2a2.1c | Complete no-write snapshot oracle | base=completed 2a2.1b paths branch | `go test ./internal/adapters/git -run 'Test.*(Snapshot|NoWrite|Worktree)'` | ordinary + linked worktree before/after `Radiograph` | snapshot-oracle hunks |
| 2a2.2 | Stage/scope safety | base=completed 2a2.1c oracle branch | `go test ./internal/adapters/git -run 'Test.*(Unmerged|Staged|Unborn|EmptyIndex|CommitA)'` | unmerged/staged/unborn/initial/empty-index/`commit -a` oracle | stage identity/output hunks |
| 2b | Planning service | base=2a2.2 branch | `go test ./internal/app` | radiography→bounded plan | service/tests only |
| 2c | MCP staging | base=2b branch | `go test ./internal/adapters/mcp` | stdio radiography→plan | MCP adapter/tests only |
| 3a | Catalog import and evidence | base=2c branch | `go test ./internal/app -run 'TestCatalogService'` | in-memory catalog service | catalog import/evidence only |
| 3b | Catalog persistence and verification | base=3a branch | `go test ./internal/app ./internal/adapters/mcp ./internal/adapters/sqlite` | stdio authorize→edit→verify | catalog/audit/verify |

Feature-chain: tracker→#3→#4→1c→1d→#6→2a1→2a2.1a→2a2.1b paths→2a2.1c oracle→2a2.2→2b→2c→3a→3b; only tracker merges to `develop`. Each child targets its immediate parent and must be retargeted/rebased if polluted.

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

- [x] 1.9 Characterization: exact multi-record v1 fixture; preserve values, NULL links, `legacy_unavailable`.
- [x] 1.10 Characterization: injected migration failure preserves schema, rows, and version exactly.
- [x] 1.11 GREEN: transactional v1→v2 rebuild; fresh v2; no invented association/replay wiring.
- [x] 1.12 Evidence: focused/runtime/check-only results and isolated rollback boundary.

## PR 1d: Replay Integrity

- [x] 1.13 RED: typed replay state and migrated-key refusal before validation, transaction, execution, or audit mutation.
- [x] 1.14 RED: exact fresh-v2 replay; migrated refusal leaves records/audit unchanged.
- [x] 1.15 GREEN: typed state/error and pre-validation, pre-transaction legacy refusal.
- [x] 1.16 Evidence: cumulative proof, exact accounting, and rollback boundary.

## Unit 2a1: Repository Inventory & Classification (base=PR #6 `a051667`)

- [x] 2.1 RED: `internal/adapters/git/resolver_test.go` inventories applicable tracked/untracked Markdown/MDX with deterministic evidence/reasons.
- [x] 2.2 GREEN: `internal/adapters/git/resolver.go` applies non-doc, symlink, generated/vendor, executable precedence; location never proves maintained.
- [x] 2.3 Evidence: real-Git fixture, compatibility/full/check/accounting, and 2a1-only rollback boundary.

## Unit 2a2.1a: Safe Git Execution & Root Binding (base=PR #7)

- [x] 2.4 RED: `resolver_test.go` rejects traversal, subdirectory, and symlink aliases; detects root replacement between physical validation and command use.
- [x] 2.5 GREEN: `resolver.go` binds exact lexical/physical root identity and centralizes read-only Git env: optional locks, lazy fetch, replacements, external diff/textconv, fsmonitor, system/global config/attributes disabled as applicable.
- [x] 2.6 Evidence: fixtures prove safe failure/no execution on root replacement; exact accounting is ≤400 additions+deletions and only 2a2.1a files/behavior revert.

## Unit 2a2.1b: Linked-worktree Path Resolution (base=completed 2a2.1a)

- [x] 2.7 RED: `resolver_test.go` frames ordinary/linked git-dir and common-dir output without whitespace loss; malformed and non-absolute output fails closed.
- [x] 2.8 GREEN: test-owned resolver exposes exact git-dir, common-dir, index, config, hooks, primary-object, and relative alternate-object paths; missing resolved paths fail closed without changing `Radiograph` behavior.
- [x] 2.9 Evidence: focused spaced ordinary/linked fixtures, resolver and 2a2.1a regressions, compatibility/full/check/accounting pass; paths-only rollback is isolated.

## Unit 2a2.1c: Complete No-write Snapshot Oracle (base=completed 2a2.1b paths)

- [x] 2.10 RED: `resolver_test.go` defines complete ordinary/linked before/after no-write snapshots for worktree, index, config, hooks, primary/alternate objects, and `.docmanager`.
- [x] 2.11 GREEN: implement test-only complete snapshot identity and fail-closed special-file/symlink-target/read errors; prove sensitivity for every bound field.
- [x] 2.12 Evidence: record ordinary/linked no-write receipts, exact accounting, and snapshot-only rollback.

## Unit 2a2.2: Stage & Scope Safety (base=completed 2a2.1c oracle)

- [x] 2.13 RED: `resolver_test.go` proves unmerged stage identity/output using the completed snapshot oracle.
- [x] 2.14 RED/GREEN: apply the oracle to staged, unborn, initial, empty-index, and `commit -a` scenarios.
- [x] 2.15 Evidence: record exact receipts/accounting (≤400 additions+deletions) and stage-only rollback.

## Unit 2b: Planning Service (base=completed 2a2.2)

- [x] 2.16 RED: `internal/app/lifecycle_service_test.go` covers radiography, denial, policy/batch, and in-plan/blocked actions.
- [x] 2.17 GREEN: `internal/app/lifecycle_service.go` creates bounded plan/batch/policy and local provenance; no authoring.

## Unit 2c: MCP Staging (base=completed 2b)

- [x] 2.18 RED/GREEN: `internal/adapters/mcp/{mcp_test.go,mcp.go}` stages tools/conflicts and stdio radiography→plan without visible writes.

## Unit 3a: Catalog Import & Evidence (base=completed 2c)

- [x] 3.1 RED: `internal/app/lifecycle_service_test.go` imports/provenance, stale/uncertain/orphan, no visible mutation.
- [x] 3.2 RED: evidenced update/review/orphan/conflict/no-action, never truthfulness.

## Unit 3b: Catalog Persistence & Verification (base=completed 3a)

- [ ] 3.3 RED: `VerifyOutcome` rejects inactive/out-of-plan/stale/revision-scope-baseline mismatch.
- [ ] 3.4 GREEN: catalog/audit/orphan/verification and idempotent MCP.
- [ ] 3.5 REFACTOR/verify: focused/stdio/`go test ./...`; runtime/rollback per commit.

## Unit 2 Mapping

| Old | New |
|---|---|
| 2a2.1 / 2.4–2.6 (checked candidate) | 2a2.1a / 2.4–2.6 (checked) + 2a2.1b paths / 2.7–2.9 (checked) + 2a2.1c oracle / 2.10–2.12 (checked) |
| 2a2.2 / 2.7–2.9 | 2a2.2 / 2.13–2.15 |
| 2b / 2.10–2.11 | 2b / 2.16–2.17 |
| 2c / 2.12 | 2c / 2.18 |
