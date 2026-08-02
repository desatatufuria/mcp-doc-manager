# Tasks: Living Documentation Lifecycle

## Review Workload Forecast

| Field | Value |
|---|---|
| Estimated changed lines | 1,050; PR 1b 440–460 |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Suggested split | tracker→PR #3→PR 1b→Unit 2→Unit 3 |
| Delivery strategy | exception-ok (PR 1b) |
| Chain strategy | feature-branch-chain |

Decision needed before apply: No
Chained PRs recommended: Yes
Chain strategy: feature-branch-chain
400-line budget risk: High
Draft/no-merge tracker targets `develop` after children.
PR 1b: maintainer-approved `size:exception` for combined correction/hardening rather than more child PRs, 421→440–460; no Unit 2/3 exception.

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|---|---|---|---|---|---|
| 1 | Domain/DB verified | #3 (base: tracker) | `go test ./internal/domain ./internal/adapters/sqlite` | Lifecycle scenario | lifecycle domain/DB |
| 1b | Minimal hardening | correction child (base: #3) | `go test ./internal/domain` | Identity scenario | lifecycle domain + test |
| 2 | Radiography/planning | Unit 2 (base: PR 1b) | `go test ./internal/app ./internal/adapters/git ./internal/adapters/mcp` | Stdio radiograph→plan | service/resolver/MCP slice |
| 3 | Catalog/verification | Unit 3 (base: Unit 2) | `go test ./internal/app ./internal/adapters/mcp ./internal/adapters/sqlite` | Stdio authorize→edit→verify | catalog/audit/verify slice |

## Unit 1: Domain/DB

- [x] 1.1 RED: `internal/domain/lifecycle_test.go` transitions; no-clock invalidation: completion/plan/policy/scope/baseline/evidence drift; local-user provenance; no auth claim.
- [x] 1.2 RED: `internal/adapters/sqlite/lifecycle_test.go` owned DB, forward migrations, `BEGIN IMMEDIATE` rollback, provenance, replay/divergent-key failure.
- [x] 1.3 GREEN: create `internal/domain/lifecycle.go` states, bounds, actions, catalog/audit/verification, transitions.
- [x] 1.4 GREEN: create `internal/adapters/sqlite/lifecycle.go` versioned DB; atomic migration/state/provenance/key; retain `ledger.go`.

## Unit 1 Hardening: Bounded Correction (PR 1b)

- [x] 1.5 RED: `internal/domain/lifecycle_test.go` rejects empty `Provenance.Operation`; tests required provenance fields individually where appropriate.
- [x] 1.6 RED: collision-free canonical length-prefix/equivalent idempotency identity; reject external/provenance-key mismatch; safely cover NUL fields.
- [x] 1.7 GREEN: provenance validation in `internal/domain/lifecycle.go` and canonical idempotency/key matching in `internal/adapters/sqlite/lifecycle.go`; no Unit 2/3 behavior; minimal PR 1b exception scope.
- [x] 1.8 Evidence: focused `go test ./internal/domain`; runtime; full `go test ./...`; check-only diff/accounting; reconcile apply-progress: native 370→exact final.

## Unit 2: Radiography/Planning

- [ ] 2.1 RED: `internal/adapters/git/resolver_test.go` excludes requirements/CMake/executable Markdown-MDX/`README.sh` from import/auth; record reason/uncertainty.
- [ ] 2.2 RED: `internal/adapters/git/resolver_test.go` rejects `git -C`, relative, nested, mismatched roots without DB/visible writes; staged/empty-index/`commit -a` preserve state.
- [ ] 2.3 GREEN: extend `internal/adapters/git/resolver.go` read-only discovery/evidence, generated/vendor exclusions, ownership conflict/uncertainty, absolute scope.
- [ ] 2.4 RED: `internal/app/lifecycle_service_test.go` complete/empty radiography, confirmed context, plan denial, policy/batch approval, automatic in-plan content, blocked structural actions.
- [ ] 2.5 GREEN: create `internal/app/lifecycle_service.go`; propose without persistence/authoring; approve bounded plan/batch/policy with local provenance.
- [ ] 2.6 RED/GREEN: `internal/adapters/mcp/{mcp_test.go,mcp.go}` staged radiograph/propose/approve/authorize; reject ownership conflict; no visible content.

## Unit 3: Catalog/Verification

- [ ] 3.1 RED: `internal/app/lifecycle_service_test.go` in-place imports/internal provenance, stale/uncertain evidence, orphan approval, no visible mutation/markers.
- [ ] 3.2 RED: same test reports evidenced update/review/orphan/conflict/no-action, never truthfulness.
- [ ] 3.3 RED: `VerifyOutcome` records caller edits; rejects inactive/out-of-plan/stale/revision-scope-baseline drift as mismatch.
- [ ] 3.4 GREEN: `internal/app/lifecycle_service.go`/`internal/domain/lifecycle.go` catalog/audit/orphan/verification; idempotent MCP outcomes.
- [ ] 3.5 REFACTOR/verify: focused commands, stdio, `go test ./...`; record runtime/rollback per work-unit commit.
