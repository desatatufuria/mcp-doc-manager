# Design: Living Documentation Lifecycle

## Technical Approach

DocManager is a staged MCP lifecycle authority, not a visible-document writer: evidence-backed plans, approved state in `.docmanager/lifecycle.db`, bounded caller edits, then verification/audit. It retains domain/app/adapter boundaries, Git resolution, SQLite, and stdio.

## Architecture Decisions

| Decision | Choice | Alternative / tradeoff | Rationale |
|---|---|---|---|
| State | Separate versioned `lifecycle.db` | Extend receipts | Independent migration; receipts remain evidence. |
| Author | MCP returns control data; caller writes | Tool writes docs | Preserves review and attribution. |
| Automation | Plan-matching content create/update only; structural/destructive approval | Per-file approval | Bounded automation with feature context. |
| Discovery | Evidence-scored Markdown/MDX, imported in place | Fixed locations | Confirmation chooses path, language, audience, owner. |
| Authority | Declared local interaction actor | Git/cryptographic authority | Records provenance without claiming stronger authority. |
| Legacy replay | Preserve v1 rows, explicitly refuse unverifiable keys | Guess a record link or delete keys | Audit survives without an invented provenance↔key relationship. |

## Domain Model and State Machines

`Radiography{Draft,Confirmed,Rejected} -> Plan{Proposed,Approved,Refused,Drifted} -> Batch{Proposed,Authorized,Reported,Verified,Rejected}`. Authorization invalidates on completion or plan/policy/scope/baseline/evidence change. Catalog states are `{Imported,Active,Stale,Uncertain,Orphan,Archived}`; audits return `{Update,Review,Orphan,Conflict,NoAction}`; transitions append provenance.

## Data Flow

```text
agent -> radiograph -> Git evidence -> proposed plan -> approval -> lifecycle.db
agent <- authorization <----------------------------- approved batch/policy
agent --Git edits + report--> verify/audit -> provenance/idempotency -> lifecycle.db
```

`radiograph` returns evidence/confidence/exclusions; `propose_plan` is unpersisted. Tools never write visible files. Available keys replay; divergent keys conflict.

## Persistence, Evidence, and Recovery

`internal/adapters/sqlite/lifecycle.go` requires a real non-symlink private workspace and uses `BEGIN IMMEDIATE`. V2 `provenance` preserves v1 `record_id`, actor, interaction, and context, adding explicit legacy values for v2-only fields. V2 `idempotency` persists `key`, original `input`/`result`, canonical `identity`, nullable `record_id`, and `replay_state TEXT NOT NULL CHECK (replay_state IN ('available','legacy_unavailable'))`.

New entries write `available`, a non-null link, and length-prefixed identity; `Save` replays their exact result. V1 lacks reliable key-to-record evidence: copy every row, retain key/input/result/audit values, set `record_id=NULL` and `legacy_unavailable`; never assign keys to the first provenance row. `Save` checks state before execution/mutation and returns `domain.ErrLegacyIdempotencyReplayUnavailable`; repeated refusal changes nothing.

V1→V2 is one transaction: `BEGIN IMMEDIATE`, rename/create/copy/validate/rename/drop, insert version 2, commit. Failure restores untouched v1 schema/data/version. Fresh DBs create final v2 tables directly. Provenance records actor, interaction/context, request/key, time, approval, evidence, input/result, versions; no cryptographic or Git authority claim.

Plans import discovered docs in place. The resolver validates roots/scopes/identities/digests; confidence is coverage, not truth. Exclusions have reasons, drift requires re-plan, conflicts block authorization, and orphans remain pending. DB rollback is atomic; visible rollback is an auditable Git revert.

## File Changes

| File | Action | Description |
|---|---|---|
| `internal/domain/lifecycle.go` | Modify | Add stable legacy-replay refusal error/state contract. |
| `internal/adapters/sqlite/lifecycle.go` | Modify | Final v2 schema, faithful legacy copy, atomic migration, replay-state gate. |
| `internal/adapters/sqlite/lifecycle_test.go` | Modify | Historical-fixture migration/replay/rollback characterization and proof coverage. |

Do not modify `opencode-daily-docmanager-guidance`, OpenCode configuration/guidance, or visible documentation; reuse only their ownership/drift and stdio patterns.

## Interfaces / Contracts

```go
var ErrLegacyIdempotencyReplayUnavailable = errors.New("legacy_idempotency_replay_unavailable")
type IdempotencyReplayState string
const ( IdempotencyAvailable IdempotencyReplayState = "available"; IdempotencyLegacyUnavailable IdempotencyReplayState = "legacy_unavailable" )
```

`Authorization` remains evidence-bound (plan/policy revisions, scope, baseline, evidence, actions). `VerifyOutcome` rejects inactive/completed, out-of-bound/unapproved structural writes, stale evidence, and drift.

## Testing Strategy

| Layer | Test | Approach |
|---|---|---|
| Unit | state/error contract | Table-driven `t.Run` success/failure cases. |
| SQLite adapter | Exact multi-record v1 fixture: all provenance/idempotency values retained; every ambiguous key unavailable; refusal causes no mutation; new v2 exact replay | `t.TempDir()` fixture and direct row assertions. |
| SQLite adapter | Forced transformation failure restores exact v1 rows, schemas, and version; fresh DB is final v2 | Failure trigger plus schema/value comparison. |
| MCP | Staged flow/no visible writes | Existing harness; external Git skips under `testing.Short()`. |

## Threat Matrix

| Boundary | Applicability | Safe/failure behavior | Planned RED tests |
|---|---|---|---|
| Documentation-like paths | Applicable: discovery | Exclude `requirements.txt`, `CMakeLists.txt`, executable Markdown/MDX, `README.sh` unless confirmed; record reason/uncertainty. | Fixture per class: no import/authorization. |
| Git repository selection | Applicable: evidence | Require validated absolute root; reject `git -C`, relative, nested, mismatched paths. | Each fails without DB/visible write. |
| Commit state | Applicable: scope evidence | Preserve index/worktree; analyze explicit state only, including empty index and `commit -a`. | Staged, empty-index, `commit -a` fixtures preserve status. |
| Push state | N/A: no push operation. | — | — |
| PR commands | N/A: no PR operation. | — | — |

## Migration / Rollout

Delivery is tracker → PR #4 → PR 1c → PR 1d → Unit 2; only the tracker integrates to `develop`. PR 1c is currently 388 changed lines and remains ≤400 with no size exception; PR 1d is separately budgeted ≤400 with no exception and remains required before Unit 2. State is additive; lifecycle tools can be disabled while records remain. No visible-document migration is required.

## Open Questions

None.
