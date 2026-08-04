## Exploration: opencode-daily-docmanager-guidance

### Current State
The signed installer, repository workspace installation, OpenCode MCP registration, read-only `document_change`, ledger receipt, and `verify_receipt` work end-to-end. OpenCode configuration currently owns only `mcp.docmanager`; it is idempotent for its exact entry, preserves unrelated keys, rejects ownership conflicts and write-time drift, and probes `docmanager mcp --version`. The MCP surface has two minimal-description tools with an explicit caller-selected `range`, `staged`, or `worktree` scope.

Repository-local `.docmanager/guidance/AGENTS.md` and skill files are installed and ownership-validated, but OpenCode does not consume them. OpenCode's documented project instruction discovery selects the first matching project `AGENTS.md`/`CLAUDE.md`/`CONTEXT.md` hierarchy, so copying a managed file into the repository root would overwrite or compete with user-owned instructions. Its configured `instructions` paths are additive, which is the appropriate integration point for a separately owned project-local guidance file.

### Affected Areas
- `internal/adapters/agent/opencode.go` — extend the owned OpenCode integration from MCP-only to an additive managed instruction reference, with exact ownership, drift, upgrade, and unconfigure behavior.
- `cmd/docmanager/install.go` and `internal/app/lifecycle.go` — coordinate repository-local managed guidance with agent configuration while preserving the existing workspace ownership boundary.
- `assets/` and `internal/app/lifecycle_test.go` — version the narrow OpenCode-facing guidance asset and prove safe idempotent installation/validation.
- `internal/adapters/mcp/mcp.go` and `internal/adapters/mcp/mcp_test.go` — make tool descriptions and input schema guidance precise without adding mutation or lifecycle tools.
- `internal/adapters/agent/core_opencode_test.go` — add RED-first cases for instruction merge, conflict/drift refusal, upgrade, status/doctor, and unconfigure.
- `README.md`, `docs/agents.md`, and `ONBOARDING.md` — align the daily workflow and remove stale statements that installation does not configure an MCP client.

### Approaches
1. **Managed additive OpenCode instruction file** — install a versioned file under the owned `.docmanager/guidance/` directory and add only that absolute path to OpenCode's additive `instructions` configuration, alongside the existing MCP entry.
   - Pros: Does not replace repository `AGENTS.md`; makes invocation guidance available in project sessions; keeps user instructions and product-managed content separately owned; supports exact drift detection and clean unconfigure.
   - Cons: Depends on OpenCode's configured-instructions capability and requires a pinned fixture/capability check.
   - Effort: Medium.

2. **Copy or merge guidance into repository `AGENTS.md`** — add DocManager rules directly to the discovered project instruction file.
   - Pros: Simple apparent discovery path.
   - Cons: Unsafe user-owned-file mutation, ambiguous merge/removal ownership, and conflicts with OpenCode's first-project-instruction-file precedence.
   - Effort: Medium initially, High to operate safely.

3. **Rely only on richer MCP descriptions** — encode all behavioral guidance in `document_change` and `verify_receipt` descriptions/schema.
   - Pros: No instruction-file configuration change.
   - Cons: Tool metadata explains calls but cannot reliably establish when a model should use the tools, when to stop, or the human review boundary.
   - Effort: Low, but insufficient.

### Recommendation
Choose **managed additive OpenCode instruction file**, plus concise richer MCP metadata. Keep the change OpenCode-first: configure one owned local MCP entry and one owned additive instruction-path reference; do not edit or infer repository `AGENTS.md`, and do not add other-agent or generic release/agent CLI composition.

The guidance should make invocation deterministic but bounded: call `document_change` once at an explicit documentation-impact checkpoint, never repeatedly during ordinary edits. Use `worktree` only to inspect tracked, unstaged edits before staging; use `staged` when the intended next commit is staged; use `range` for committed review/push ranges with explicit two-dot endpoints. Do not call for non-code/documentation administrative work, empty/unknown scope, or merely because the tool is available. Treat `update`, `create`, and `no-impact` as a report for human judgment, not a documentation correctness verdict. After the human completes any documentation work, call `verify_receipt` once before handoff only if the selected scope and considered documentation bytes have not changed; otherwise re-run analysis and review the new report before verifying.

Retain the optional pre-push hook as a separate, explicit workspace concern. It verifies stored receipts for push updates and must neither select MCP tools nor be enabled by agent configuration. The narrow slice should add an agent status/doctor capability record sufficient to report: supported OpenCode fixture/capability, exact owned MCP entry, exact owned instruction reference, readable matching managed guidance version, and bounded binary probe. It should refuse unknown configuration shape, changed managed reference/content, or unsupported capability rather than repair silently. Reconfiguration with matching state is a no-op; upgrades may replace only a versioned owned guidance asset/reference after snapshot/drift checks; unconfigure removes only exact owned entries and leaves repository/user instructions and `.docmanager` workspace state intact unless the user separately uninstalls that workspace.

This reconciles, rather than reopens, `docmanager-installer-agent-integration`: its managed MCP-and-guidance intent remains the source direction, while this change completes only the proven OpenCode daily-consumption gap. Historical plan completion and stale verification records are not acceptance evidence for this new slice.

### Risks
- OpenCode configuration-schema or instruction-path semantics can drift; pin the supported shape/capability in fixtures and fail closed before mutation.
- An absolute project guidance path can become stale after repository moves, copied worktrees, or workspace uninstallation; status/doctor must expose it and unconfigure must remain ownership-safe.
- A prescriptive instruction can cause unnecessary calls; test/document negative triggers and require exactly one explicit scope rather than automatic repeated analysis.
- Receipts deliberately become invalid when staged/worktree evidence or considered documentation bytes change; the guidance must require re-analysis and fresh human review, not retry verification.
- The existing hook's first-push `initial` receipt gap remains out of scope; do not present the hook as a replacement for the model-guided daily workflow.

### Ready for Proposal
Yes — propose a single-PR, OpenCode-only slice under the 1000-line review budget. State explicit non-goals: other agents, generic agent/release CLI composition, hook auto-enable, MCP lifecycle/mutation tools, automatic documentation editing, inferred scopes, and any revision of the historical installer plan.
