# Design: OpenCode Daily DocManager Guidance

## Technical Approach

Extend OpenCode MCP with a guidance reference. Workspace owns `.docmanager`; the adapter merges the pair. User instructions, hooks, and installer behavior do not change.

## Architecture Decisions

| Decision | Alternatives / trade-off | Choice and rationale |
|---|---|---|
| Guidance identity | Root `AGENTS.md` conflicts with users | Embed versioned `.docmanager/guidance/opencode.md`; its cleaned absolute path is the owned reference. |
| Configuration ownership | Treat entries independently | Require MCP, instruction path, and guidance identity as one pair; reject conflict or drift before mutation. |
| Instruction merge | Replace instructions | Pin `instructions` as absent or `[]string`; preserve unrelated values/order and append the absent path. |
| Upgrade/removal | Repair changed files; delete state | Upgrade only catalogued older guidance; drift refuses mutation; exact removal retains `.docmanager`. |
| Receipt eligibility | Infer changes broadly | Record selected scope and considered-document bytes. |
| Probe rollback | Remove all entries after error | Roll back only a pair newly created by this attempt, after compare-and-write ownership validation. |

## Data Flow

```text
asset -> WorkspaceInstall writes/verifies .docmanager/guidance/opencode.md
install -> OpenCode Inspect/preflight -> atomic MCP + instructions merge -> bounded probe

explicit scope -> document_change -> receipt snapshot -> human review -> verify_receipt
                                      changed scope/docs -> new analysis -> new review
```

`ReceiptSnapshot` records selected scope, `ConsideredDocs map[path]digest`, and consumption. `receiptEligibility(snapshot, scope, docs, consumed)` returns `scope_changed`, `considered_bytes_changed`, `already_verified`, or eligible; an injected provider supplies `docs`. RED tests cover every invalidator, refusal with no `verify_receipt` call, then new `document_change`, snapshot, and explicit human review before verification. Bytes outside `ConsideredDocs` do not invalidate.

## Installer Failure States

| Failure point | State and safe behavior | RED seam |
|---|---|---|
| Workspace install fails | Do not configure/probe; OpenCode is untouched. Report its failed/previous workspace state; no cross-boundary rollback. | Fake `workspaceInstall` error; assert no configure/probe and unchanged config bytes. |
| OpenCode configuration fails | Successful workspace remains; do not probe. Preflight writes nothing; atomic-write errors retain original bytes and existing owned pair. | Fake preflight/write errors; assert retained workspace, zero probe, byte-identical config. |
| Post-configuration probe fails | Workspace remains. Roll back exactly a pair newly created this attempt only if current bytes equal the post-config snapshot; preserve unrelated values. Retain a no-op/pre-existing pair. Drifted rollback deletes nothing and fails. | Fake runner failure with created, existing, and drifted snapshots; assert exact-entry and retained-workspace behavior. |

Workspace remains after downstream failures: it is independent owned state needed for inspection/retry; deletion could remove valid setup.

## File Changes

| File | Action | Description |
|---|---|---|
| `assets/opencode.md`, `assets/assets.go` | Create/Modify | Guidance catalog. |
| `internal/app/lifecycle.go`, tests | Modify | Asset lifecycle, snapshots, drift. |
| `internal/adapters/agent/{opencode,registry,merge}.go`, tests | Modify | Instructions, paired ownership, status, rollback seams. |
| `cmd/docmanager/{install,install_test,acceptance_test}.go` | Modify | Ordered lifecycle/failure acceptance. |
| `internal/adapters/mcp/{mcp,mcp_test}.go` | Modify | Read-only metadata. |
| `README.md`, `docs/agents.md`, `ONBOARDING.md`, fixtures | Modify | Workflow and pinned config. |

## Interfaces / Contracts

```go
type GuidanceIdentity struct { Path, Version, Digest string }
type ReceiptSnapshot struct { Scope string; ConsideredDocs map[string]string; Consumed bool }
```

Status/doctor report support, MCP, instruction, guidance readability/version, and probe without repair. Guidance requires caller-selected `worktree`, `staged`, or `base..head`; it prohibits inferred scope, routine coding, editing, lifecycle work, other agents, hooks, release composition, and historical-plan evidence. Metadata repeats one-call, read-only, review, and eligibility limits.

## Testing Strategy

Strict TDD: inject snapshots, probe runner, document-byte provider, and transcript recorder rather than relying on an LLM.

| Layer | What to Test | Approach |
|---|---|---|
| Unit | Merge, drift, eligibility | Fixtures; pure eligibility and zero-write refusal. |
| Integration | Asset lifecycle and failure states | Temp Git root/fake runtime; snapshots and retained workspace. |
| Acceptance | Calls/non-calls and review | Transcript oracle, not model proof. |

## Threat Matrix

| Boundary | Applicability | Design response | Planned RED tests |
|---|---|---|---|
| Documentation-like paths | N/A — classification is unchanged. | No classifier behavior changes. | None. |
| Git repository selection | N/A — this change does not alter repository/cwd selection; guidance uses existing scope handling. | No new acceptance behavior for relative, nested, or absolute repository input. | None. |
| Commit state | Applicable — guidance states existing staged/worktree semantics. | Preserve resolver behavior: `worktree` considers unstaged changes; `staged` considers index changes. | Index versus tracked-unstaged fixtures. |
| Push state | N/A — no hook/push behavior changes. | None. | None. |
| PR commands | N/A — no PR command integration. | None. | None. |

## Migration / Rollout

No data migration. MCP-only installs upgrade only after asset validation and config preflight. Roll back with exact unconfigure then revert; never alter user instructions or workspace/hook state.

## Open Questions

None.
