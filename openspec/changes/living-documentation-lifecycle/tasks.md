# Tasks: Living Documentation Lifecycle

## Review Workload Forecast

| Field | Value |
|---|---|
| Estimated changed lines | 1,050 |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Suggested split | Draft tracker→develop; PR 1 → PR 2 → PR 3 |
| Delivery strategy | ask-on-risk |
| Chain strategy | feature-branch-chain |

Decision needed before apply: No
Chained PRs recommended: Yes
Chain strategy: feature-branch-chain
400-line budget risk: High
Tracker: draft/no-merge; only it integrates to develop after children.

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|---|---|---|---|---|---|
| 1 | Domain + DB | PR 1 (base: tracker) | `go test ./internal/domain ./internal/adapters/sqlite` | `go test ./internal/adapters/sqlite -run Lifecycle` | `internal/domain/lifecycle.go` |
| 2 | Radiography + planning | PR 2 (base: PR 1) | `go test ./internal/app ./internal/adapters/git ./internal/adapters/mcp` | Stdio radiograph→plan via `go run ./cmd/docmanager` | service/resolver/MCP plan slice |
| 3 | Catalog + verification | PR 3 (base: PR 2) | `go test ./internal/app ./internal/adapters/mcp ./internal/adapters/sqlite` | Stdio authorize→edit→verify transcript | catalog/audit/verify tables/tools |

## Phase 1: Domain and Persistence (Unit 1)

- [ ] 1.1 RED: in `internal/domain/lifecycle_test.go`, test transitions, no-wall-clock evidence-bound invalidation (completion/plan/policy/scope/baseline/evidence drift), local-actor provenance, and no-authentication claim.
- [ ] 1.2 RED: in `internal/adapters/sqlite/lifecycle_test.go`, test owned DB, forward migrations, `BEGIN IMMEDIATE` rollback, provenance, identical-key replay, and divergent-key failure.
- [ ] 1.3 GREEN: create `internal/domain/lifecycle.go` typed states, authorization bounds, planned actions, catalog/audit/verification records, and transition validation.
- [ ] 1.4 GREEN: create `internal/adapters/sqlite/lifecycle.go` versioned `.docmanager/lifecycle.db`, atomic migrations/state/provenance/keys; retain `ledger.go` unchanged.

## Phase 2: Radiography, Planning, and Approval (Unit 2)

- [ ] 2.1 RED: in `internal/adapters/git/resolver_test.go`, exclude `requirements.txt`, `CMakeLists.txt`, executable Markdown/MDX, and `README.sh` from import/authorization with reason/uncertainty.
- [ ] 2.2 RED: in `internal/adapters/git/resolver_test.go`, reject `git -C`, relative, nested, and mismatched roots without DB/visible writes; staged/empty-index/`commit -a` preserve index/worktree state.
- [ ] 2.3 GREEN: extend `internal/adapters/git/resolver.go` discovery/evidence for read-only radiography, generated/vendor exclusions, ownership conflicts, uncertain classification, and validated absolute scope.
- [ ] 2.4 RED: in `internal/app/lifecycle_service_test.go`, cover complete/empty radiography, confirmed context, unapproved/refused plan denial, policy audit, batch approval, automatic in-plan content only, and blocked structural actions.
- [ ] 2.5 GREEN: create `internal/app/lifecycle_service.go`; propose plans without persistence/authoring, approve bounded plans/batches/policy with declared local interaction provenance.
- [ ] 2.6 RED/GREEN: extend `internal/adapters/mcp/mcp_test.go` and `internal/adapters/mcp/mcp.go` staged radiograph/propose/approve/authorize tools; reject ownership conflict and never return/write visible document content.

## Phase 3: Catalog, Audit, and Verification (Unit 3)

- [ ] 3.1 RED: in `internal/app/lifecycle_service_test.go`, test in-place approved imports with internal-only provenance; stale/uncertain evidence, evidenced orphan pending approval, and no visible-document mutation/markers.
- [ ] 3.2 RED: in `internal/app/lifecycle_service_test.go`, test on-demand/change audits return evidenced update/review/orphan/conflict/no-action, never a truthfulness verdict.
- [ ] 3.3 RED: in `internal/app/lifecycle_service_test.go`, test `VerifyOutcome` records matching caller edits; reject completed/inactive authorization, out-of-plan paths/actions, stale evidence, and revision/scope/baseline drift as mismatch.
- [ ] 3.4 GREEN: implement catalog import/audit/orphan and verification in `internal/app/lifecycle_service.go`/`internal/domain/lifecycle.go`; wire MCP audit/verify tools with idempotent persisted outcomes.
- [ ] 3.5 REFACTOR/verify: run focused unit commands, stdio transcripts, then `go test ./...`; record each unit’s runtime result and rollback evidence with its work-unit commit.
