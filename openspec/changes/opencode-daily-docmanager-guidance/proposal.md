# Proposal: OpenCode Daily DocManager Guidance

## Intent

Complete the OpenCode daily-consumption gap: safely make managed, project-local DocManager guidance available to OpenCode sessions so users get bounded, review-first documentation-impact assistance without changing user-owned instructions or reopening the signed installer foundation.

## Scope

### In Scope
- Install a versioned guidance asset in the owned `.docmanager/guidance/` area and register its absolute path through OpenCode's additive `instructions` configuration.
- Preserve exact ownership boundaries: reject unsupported/unknown configuration, conflicts, and drift; make matching reconfiguration a no-op; permit checked upgrades only for owned guidance.
- Add OpenCode status/doctor reporting for supported capability, owned MCP entry and instruction reference, guidance version/readability, and bounded binary probe; unconfigure only exact owned entries.
- Enrich read-only MCP tool descriptions and schema guidance with explicit scope selection, one-call checkpoints, negative triggers, human review, and receipt-verification conditions.
- Prove realistic end-to-end configuration, invocation/non-invocation, drift, upgrade, unconfigure, and receipt-review acceptance; align user-facing workflow documentation.

### Out of Scope
- Other agents; generic agent/release CLI composition; hook auto-enable; MCP lifecycle or mutation tools.
- Automatic documentation editing, inferred scopes, edits to repository `AGENTS.md`, and changes to existing workspace/hook behavior.
- Reopening or using historical installer-plan completion as evidence for this change.

## Capabilities

### New Capabilities
- `opencode-daily-docmanager-guidance`: Owned additive OpenCode guidance, bounded read-only MCP usage, and safe integration lifecycle semantics.

### Modified Capabilities
None — `openspec/specs/` contains no existing capabilities.

## Approach

Extend the owned OpenCode integration from MCP-only to one exact additive instruction reference. Guidance selects exactly one explicit `worktree`, `staged`, or two-dot `range` checkpoint; it requires human review after `document_change`, and allows one `verify_receipt` only when scope and considered documentation bytes remain unchanged. Pin the supported config shape in fixtures and fail closed before mutation.

## Affected Areas

| Area | Impact | Description |
|---|---|---|
| `internal/adapters/agent/opencode.go` | Modified | Owned instruction merge, drift, status/doctor, unconfigure. |
| `cmd/docmanager/install.go`, `internal/app/lifecycle.go` | Modified | Coordinate guidance lifecycle boundaries. |
| `assets/`, `internal/adapters/mcp/mcp.go` | Modified | Versioned guidance and bounded MCP metadata. |
| `internal/*/*_test.go` | Modified | RED-first lifecycle and MCP acceptance coverage. |
| `README.md`, `docs/agents.md`, `ONBOARDING.md` | Modified | Daily workflow expectations. |

## Risks

| Risk | Likelihood | Mitigation |
|---|---|---|
| OpenCode instruction semantics drift | Med | Pinned capability fixture; refuse mutation. |
| Moved workspace leaves stale path | Med | Expose via status/doctor; ownership-safe unconfigure. |
| Guidance causes excess calls | Med | Negative triggers and exact-scope tests. |

## Rollback Plan

Unconfigure the exact owned OpenCode MCP and instruction entries, retain user configuration and `.docmanager` state, then revert the release. Never edit user-owned instructions.

## Dependencies

- Supported OpenCode additive `instructions` configuration shape, validated by fixture.
- Existing signed installer, MCP registration, and receipt foundation.

## Success Criteria

- [ ] OpenCode consumes only the owned additive guidance path without replacing project instructions.
- [ ] Configuration lifecycle rejects drift/conflicts and preserves unrelated configuration.
- [ ] Guidance produces one explicit analysis checkpoint, human review before eligible receipt verification, and no calls for defined negative triggers.
- [ ] End-to-end tests demonstrate install, no-op, upgrade, status/doctor, unconfigure, and receipt invalidation/re-analysis behavior.
