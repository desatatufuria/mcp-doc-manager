# Design: Living Documentation Lifecycle

## Technical Approach

DocManager becomes a staged MCP lifecycle authority, never a visible-document writer. It returns evidence-backed proposals, persists approved state in `.docmanager/lifecycle.db`, authorizes bounded caller work, then verifies/audits reported Git edits. It retains the domain/app/adapter split, Git resolver, SQLite, receipts, ownership checks, and stdio.

## Architecture Decisions

| Decision | Choice | Alternative / tradeoff | Rationale |
|---|---|---|---|
| State | Separate versioned `lifecycle.db` | Extend receipts | Independent migrations; receipts stay evidence. |
| Author | MCP returns control data; caller writes | Tool writes docs | Preserves review and attribution. |
| Automation | Automatic only for plan-matching content create/update; approval-required per batch | Per-file approval | Feature context; structural/destructive always approved. |
| Discovery | Evidence-scored Markdown/MDX; import in place | Fixed README/docs | Confirmation chooses path, language, audience, owner. |
| Authority | Explicit local user in the active interaction | Git identity or cryptographic authentication | MVP records a declared actor and interaction/context provenance without claiming either stronger authority source. |

## Domain Model and State Machines

`Radiography{Draft,Confirmed,Rejected} -> Plan{Proposed,Approved,Refused,Drifted} -> Batch{Proposed,Authorized,Reported,Verified,Rejected}`. `Authorization{Active,Invalidated}` becomes invalid when its batch completes or its approved plan, policy, scope, baseline, or relevant evidence changes. Policy `{ApprovalRequired,AutomaticAfterApprovedPlan}` changes `{Proposed,Approved,Rejected}` with old/new audit. `CatalogEntry` has required path, purpose, audience, owner/candidates, related areas, state `{Imported,Active,Stale,Uncertain,Orphan,Archived}`, evidence, verification, pending action. `Audit{Running,Reported}` returns `{Update,Review,Orphan,Conflict,NoAction}`. `PendingAction{Open,Approved,Refused,Completed,Cancelled}`; structural actions never auto-approve. `Verification{Pending,Verified,Mismatch,Stale}`. All transitions append provenance.

## Data Flow

```text
agent -> radiograph -> Git/discovery evidence -> proposed plan -> approval -> lifecycle.db
agent <- authorization <-------------------------------- approved batch/policy
agent --normal Git edits + reported paths--> verify/audit -> provenance -> lifecycle.db
```

`radiograph` returns findings, confidence, exclusions, ownership conflicts, context candidates; `propose_plan` returns an unpersisted plan. `approve_plan`, `approve_batch`, `change_policy` record approvals. `authorize_batch` returns allowed paths/actions and baseline; `audit` returns candidates/rationale; `verify_outcome` records verified/mismatch. Tools never return content or write/move/delete visible files. Identical idempotency keys replay; divergent reuse fails.

## Persistence, Evidence, and Recovery

`internal/adapters/sqlite/lifecycle.go` requires a real non-symlink owned workspace, applies forward-only `schema_migrations`, and uses `BEGIN IMMEDIATE` for atomic state plus provenance. Tables: `radiographies`, `plans`, `batches`, `catalog_entries`, `audits`, `pending_actions`, `verifications`, `policy_history`, `provenance`, `idempotency`. Failed migrations roll back; provenance stores declared local actor, interaction/context, request/key, time, approval, evidence, input/result, versions. It does not claim cryptographic authentication or Git-derived authority.

Approved plans import discovered docs in place. `Resolver` supplies validated roots, explicit scopes, identities, and digests; on-demand audit obtains current evidence. Confidence is coverage, never truthfulness. Excluded generated/vendor/executable material has a reason; uncertainty persists. Baseline change causes stale evidence/plan drift and re-plan. Ownership conflict blocks authorization; mismatched writes are `Mismatch`; orphans remain pending. DB failure rolls back; visible rollback is Git revert and state rollback is an auditable compensating transition.

## File Changes

| File | Action | Description |
|---|---|---|
| `internal/domain/lifecycle.go` | Create | Typed states, requests/results, transition validation. |
| `internal/app/lifecycle_service.go` | Create | Radiography, planning, authorization, audit, verification use cases. |
| `internal/adapters/sqlite/lifecycle.go` | Create | Owned database, migrations, transactions. |
| `internal/adapters/git/resolver.go` | Modify | Discovery/audit evidence using safe root/scope validation. |
| `internal/adapters/mcp/mcp.go` | Modify | Staged lifecycle tools and typed errors. |
| `internal/{domain,app,adapters}/*_test.go` | Modify/Create | RED domain, SQLite, Git, stdio contracts. |

Do not modify `opencode-daily-docmanager-guidance`, OpenCode configuration/guidance, or visible documentation. Reuse their ownership/drift and stdio patterns only; receipts do not define freshness or authorization.

## Interfaces / Contracts

```go
type Authorization struct { BatchID, PlanRevision, PolicyRevision string; Scope Scope; Baseline, Evidence string; Allowed []PlannedAction }
type PlannedAction struct { Path string; Kind ActionKind; Structural bool }
type VerifyOutcomeRequest struct { BatchID, IdempotencyKey string; Writes []ReportedWrite; Evidence string }
```

`Authorization` is evidence-bound, not wall-clock-bound: it carries approved plan/policy revisions, scope, baseline, and relevant evidence identity. `VerifyOutcome` rejects inactive/completed authorization, out-of-bound writes, unapproved structural actions, stale evidence, and revision/scope/baseline drift. A current `Receipt` may be attached as evidence only.

## Testing Strategy

| Layer | Test | Approach |
|---|---|---|
| Unit | states, confidence/exclusions, authorization/mismatch | Table-driven RED `t.Run`. |
| Adapter | migrations, ownership/symlinks, idempotency, Git evidence | `t.TempDir()` SQLite/Git fixtures. |
| MCP | staged stdio flow and no visible writes | Existing harness; external Git skips under `testing.Short()`. |

## Threat Matrix

| Boundary | Applicability | Safe/failure behavior | Planned RED tests |
|---|---|---|---|
| Documentation-like paths | Applicable: discovery | Exclude `requirements.txt`, `CMakeLists.txt`, executable Markdown/MDX, `README.sh` unless confirmed; record reason/uncertainty. | Fixture per class: no import/authorization. |
| Git repository selection | Applicable: evidence | Require validated absolute root; reject `git -C`, relative, nested, mismatched paths. | Each fails without DB/visible write. |
| Commit state | Applicable: scope evidence | Preserve index/worktree; analyze explicit state only, including empty index and `commit -a`. | Staged, empty-index, `commit -a` fixtures preserve status. |
| Push state | N/A: no push operation. | — | — |
| PR commands | N/A: no PR operation. | — | — |

## Migration / Rollout

MVP work units: (1) domain + migrations/tests (~300 lines), (2) radiography/plan MCP (~350), (3) catalog audit/verification (~400). With `ask-on-risk`, require a delivery decision before combining units; every unit is below 800 lines. State is additive; disable lifecycle tools to roll back while retaining records. No visible-doc migration is required.

## Open Questions

None.
