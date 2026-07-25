# Proposal: Repository Documentation Manager

## Intent

Give agents and developers one explicit feature-close analysis that determines whether repository documentation needs attention. Git remains authoritative; the result is inspectable evidence, not an automatic documentation change.

## Product Outcome

A repository-local, agent-neutral tool makes documentation impact visible and actionable at feature close: `update`, `create`, or `no-impact`, with the exact Git scope, rationale, confidence, and evidence behind the outcome.

## Scope

### In Scope
- A standalone, repository-only CLI and MCP stdio server with one primary `document_change` analysis operation.
- Caller-selected Git range, staged, or worktree scope; resolved refs and content identity are reported as evidence.
- Exactly three outcomes: `update`, `create`, and `no-impact`; each identifies candidate paths, rationale, confidence, and receipt/checkpoint data.
- Explicit feature-close invocation through distributed skill and `AGENTS.md` guidance.
- Optional deterministic receipt verification that validates a content-bound successful analysis without invoking an LLM or mutating files.
- Go as the recommended runtime, gated by a macOS/Linux/Windows spike for the chosen Tier-1 MCP SDK and pure-Go SQLite driver.
- A later, thin Gentle AI community-tool adapter that delegates to the external binary.

### Out of Scope
- Automatic documentation writes, patch application, or feature-completion inference from merge state.
- External providers, plugin loading, daemon/background services, remote synchronization, or embedded Gentle AI runtime/skill duplication.
- FTS or a documentation graph before measured discovery-quality need.

## Capabilities

### New Capabilities
- `repository-documentation-analysis`: Evidence-backed documentation-impact analysis and receipt verification for repository Git changes.

### Modified Capabilities
None — no existing specifications exist.

## Approach

Build an adapter-bounded Go single binary: shared Git-analysis and receipt service exposed through CLI and MCP stdio. Repository-local SQLite is a cache/ledger, never truth. Define the Git-scope, evidence, and canonical receipt-digest contracts before optional hooks; agent review applies any proposed documentation change.

## Affected Areas

| Area | Impact | Description |
|---|---|---|
| `go.mod`, `cmd/`, `internal/` | New | Go CLI, MCP adapter, analysis core, and compatibility spike. |
| `.docmanager/` | New | Local metadata, analyses, and receipts. |
| `AGENTS.md`, skill assets | New | Explicit feature-close invocation guidance. |
| `.githooks/` | New (optional) | Deterministic receipt verification only. |

## Risks

| Risk | Likelihood | Mitigation |
|---|---|---|
| Ambiguous Git scope | High | Require explicit scope and expose resolved refs/digest. |
| Stale receipt acceptance | Med | Bind digest to scope, documentation blobs, tool/schema versions. |
| Cross-platform runtime assumptions | Med | Spike and CI validation before committing to Go. |
| Heuristic misclassification | Med | Return inspectable evidence; humans/agents retain edit control. |

## Rollback and Exit Criteria

Remove the external binary, MCP registration, guidance, and optional hook; delete only local `.docmanager/` cache/receipts. Do not proceed past the spike if the Go MCP/SQLite path cannot produce supported macOS, Linux, and Windows binaries with deterministic receipt verification; reassess Rust.

## Dependencies

- Git available in the target repository.
- Selected Tier-1 Go MCP SDK and pure-Go SQLite driver pass the compatibility spike.

## Success Criteria

- [ ] One explicit CLI/MCP analysis returns exactly one evidence-backed outcome for supported Git scopes.
- [ ] The MVP never writes documentation; invoking guidance remains the feature-close control point.
- [ ] Optional receipt verification deterministically accepts only the matching successful analysis.
- [ ] The cross-platform spike validates the proposed Go distribution path.
