## Exploration: living-documentation-lifecycle

### Current State
DocManager is currently a safe, **read-only documentation-impact review** product. The `document_change` MCP tool accepts only an explicit Git scope, analyzes evidence, creates a content-bound receipt, and persists it in `.docmanager/ledger.db`; `verify_receipt` requires a caller-supplied `reviewed: true` flag and verifies unchanged scope/evidence. The MCP deliberately exposes no onboarding, lifecycle, planning, catalog, document-writing, or mutation tools.

The uncommitted `opencode-daily-docmanager-guidance` work makes this workflow easier to invoke from OpenCode: it installs an owned `.docmanager/guidance/opencode.md`, safely registers that absolute path plus the MCP entry in OpenCode configuration, preserves unrelated configuration, detects drift, and adds a test-only transcript oracle. It does **not** make DocManager a living documentation manager. Its instruction explicitly forbids automatic editing and lifecycle management. Receipt verification only compares stored receipt content; it does not consume receipts or expose detailed status, and the transcript does not execute an LLM.

Industry evidence supports version-controlled human-facing documentation reviewed with code (Write the Docs), reader-goal-based information architecture rather than fixed filenames (Diátaxis), progressive architecture views (C4), decision rationale as a separate log (ADRs), and separately modeled catalog ownership/metadata (Backstage). These principles support a repository-discovered document plan with an internal catalog, not a hard-coded README/ONBOARDING generator. Sources: [Docs as Code](https://www.writethedocs.org/guide/docs-as-code/), [Diátaxis](https://diataxis.fr/), [C4](https://c4model.com/), [ADRs](https://adr.github.io/), [Backstage Catalog/TechDocs](https://backstage.io/docs/features/techdocs/), and [NIST AI RMF](https://www.nist.gov/itl/ai-risk-management-framework).

### Affected Areas
- `internal/adapters/mcp/mcp.go` — replace the two-tool, read-only-only surface with a deliberately staged lifecycle surface; retain scope validation and error classification.
- `internal/app/document_change.go`, `internal/app/verify_receipt.go`, `internal/app/hook.go` — preserve evidence, receipt, and optional push-verification primitives, but do not treat a receipt as documentation freshness or authoring approval.
- `internal/app/lifecycle.go` and `internal/adapters/sqlite/ledger.go` — reusable owned-state and local SQLite foundations; need a distinct versioned lifecycle catalog/policy model rather than overloading the receipt ledger.
- `internal/adapters/git/` and `internal/domain/` — reusable safe repository-root validation, explicit Git scopes, evidence collection, typed errors, and domain boundaries for discovery and change-impact inputs.
- `internal/adapters/agent/opencode.go`, `cmd/docmanager/install.go`, `assets/opencode.md` — salvage safe OpenCode installation/configuration mechanics, but replace the read-only guidance contract with lifecycle orchestration guidance after the MCP contract exists.
- `cmd/docmanager/acceptance_test.go` and `internal/**/*_test.go` — retain temporary-repository, MCP-stdio, configuration-drift, and receipt tests; replace the string/transcript-only behavior claims with executable lifecycle scenarios.
- `README.md`, `ONBOARDING.md`, `docs/agents.md` — current worktree wording frames DocManager as a checker; future work must reframe it only after approved lifecycle behavior exists.

### Approaches
1. **Repository-local human docs with a separate internal lifecycle catalog** — During onboarding, inspect repository structure, existing documentation, build/test conventions, audiences, ownership signals, and recent history. Return a radiography and proposed information architecture/storage location; after explicit approval, persist policy, inventory, ownership, provenance, freshness signals, and proposed/accepted actions in `.docmanager/`. The calling agent writes approved human-facing files through ordinary repository edits; DocManager records and verifies the lifecycle.
   - Pros: Keeps visible documentation reviewable and portable; keeps private operational metadata separate; supports arbitrary approved locations; maps well to docs-as-code, Diátaxis, C4, ADRs, and Backstage-style ownership; preserves human control and traceability.
   - Cons: Requires a new durable catalog schema, discovery heuristics, approval state machine, and migration/import path.
   - Effort: High overall; Medium for a bounded onboarding-and-plan MVP.

2. **Managed documentation workspace under `.docmanager/`** — Generate and maintain all documentation inside the tool-owned directory, with metadata and reader documentation together; optionally export selected files later.
   - Pros: Simple ownership and cleanup; lowest initial risk of overwriting existing docs; existing workspace installer is directly reusable.
   - Cons: Hides project assets from normal contributor flows, mixes machine state with reader content, makes repository conventions secondary, and creates a difficult export/migration problem. It conflicts with the requirement that repository/user documentation remain visible assets.
   - Effort: Medium initially, High after export/discovery/compatibility needs emerge.

3. **Catalog-only impact checker with external agent prompts** — Keep DocManager read-only, enrich receipts with inventory/freshness findings, and tell coding agents to plan/write documentation themselves without durable lifecycle policy or approved information architecture.
   - Pros: Reuses nearly all current code; small API change; keeps mutation risk outside the MCP.
   - Cons: Cannot onboard a repository, remember automation policy or ownership, coordinate removals/refactors, or reliably distinguish accepted deferred work from stale docs. It reframes the product inadequately and makes traceability dependent on each caller.
   - Effort: Low.

4. **Autonomous DocManager authoring MCP** — Let tools directly create, modify, move, and delete human-facing documentation once change impact is detected.
   - Pros: Fastest apparent update loop and can automate repetitive maintenance.
   - Cons: Couples discovery, planning, generation, and mutation in one trust boundary; increases hallucination, overwrite, and incorrect-deletion risk; obscures whether DocManager or the calling agent authored a change. Human-in-the-loop controls recommended by the NIST risk-management approach become difficult to enforce.
   - Effort: High.

### Recommendation
Choose **Approach 1**, with DocManager as the lifecycle **orchestrator and evidence/catalog authority**, and the calling coding agent/LLM as the **author** of approved human-facing documentation. The MCP should discover, classify, propose, record policy/approval/ownership/provenance, detect impact and staleness, and produce a bounded change plan. It must not silently choose document names or locations, edit user-facing documentation, or infer approval. A caller then writes the approved plan using normal repository tools and reports changed paths; DocManager validates the result against the approved plan and records a traceable outcome.

#### Bounded MVP
- Add an explicit `onboard`/radiography operation that is read-only and reports discovered documentation, candidate audiences, architecture evidence, missing/weak areas, and confidence/limits.
- Add a proposal operation that returns—not writes—a proposed information architecture, storage location, inventory, initial document set, ownership candidates, and migration treatment for existing docs. Require approval before persistence or generation begins.
- Persist a project-selected policy: `approval-required` or `automatic-after-approved-plan`; allow an explicit policy-change operation with an audit record. “Automatic” may create lifecycle proposals/actions only within the approved plan and must remain traceable and reversible.
- Store catalog/policy/receipts internally under `.docmanager/`, but keep generated/maintained documentation in an approved visible repository location. The MVP must support existing documents as imported inventory, not duplicate or replace them by default.
- Add change-impact analysis that maps changed repository evidence to catalog entries and returns `update`, `review`, `orphan`, `conflict`, or `no-action`; require the caller to write approved updates and then record verification.
- Start with text/Markdown inventory and Git evidence. Defer semantic truthfulness scoring, autonomous deletion, and full cross-repository catalogs.

#### Non-goals
- Selecting `README`, `ONBOARDING`, `docs/`, or any other filename/location without repository discovery and user approval.
- Direct DocManager mutation of human-facing documentation, user instructions, or source code.
- Replacing documentation authors, code review, or subject-matter ownership with an LLM.
- A universal documentation taxonomy, mandatory Diátaxis/C4/ADR output, or Backstage integration; these are optional evidence-informed patterns.
- Repairing, resetting, or changing the blocked `opencode-daily-docmanager-guidance` OpenSpec change or its Gentle AI provider/RDD state.

#### Salvageable Current Worktree Scope
**Keep as infrastructure:** safe owned `.docmanager` installation and path checks; SQLite/local ownership patterns; Git-root and explicit-scope validation; evidence collection; receipts as a narrow “evidence unchanged” primitive; MCP stdio boundary; OpenCode config merge/drift/ownership protections; temporary-repository and MCP integration test patterns.

**Replace or reframe:** tool descriptions, `assets/opencode.md`, acceptance transcript claims, and user-facing workflow prose that define DocManager as one read-only end-of-feature check. Receipts must become one verification input inside the broader lifecycle, not the product’s definition of documentation health. The current 704 changed lines are within the stated 800-line review budget, but incorporating them unchanged into the lifecycle MVP would create an incoherent product contract; split/reframe them after lifecycle APIs and policy semantics are specified.

### Risks
- Discovery may overstate confidence or mistake generated/vendor content for maintained documentation; radiography must expose evidence, exclusions, and uncertainty.
- “Automatic” policy can be interpreted as permission to overwrite human assets; scope it to approved plans, record actions, and retain approval-required as the default recommendation.
- A catalog can become a second stale documentation system; make source paths, owner, freshness basis, last verification, and lifecycle state explicit, and detect drift rather than asserting freshness.
- Mapping code changes to documentation is probabilistic; report candidates and rationale, not a correctness verdict, especially for refactors/removals.
- Existing worktree guidance and tests may imply guaranteed agent behavior although an LLM is not executed; acceptance must test tool contracts and audit state, with end-to-end agent behavior treated separately.
- The product boundary can blur if MCP tools write documents directly; retain the orchestrator/author separation and explicit provenance for every lifecycle action.

### Unresolved Product Questions
- What is the minimum approval record: a plan-wide approval, document-level approvals, or both for the approval-required policy?
- Which actor identities and ownership sources are trusted (Git, CODEOWNERS, catalog files, user input), and how are conflicts resolved?
- Does automatic mode authorize only proposal/plan updates, or also agent-authored document patches after the plan is approved?
- What evidence establishes freshness for documents that are not tied to a Git change, and what thresholds trigger a review?
- How should generated documentation be identified, reviewed, and rolled back without conflating it with internal `.docmanager` state?

### Ready for Proposal
Yes — propose a bounded lifecycle-foundation MVP, not an autonomous documentation writer. The proposal should first lock the user-approval state machine, catalog schema boundary, MCP/coding-agent responsibility split, and executable onboarding/change-impact scenarios. With `ask-on-risk` and an 800-line review budget, require a delivery decision before apply if the plan combines persistence, MCP contracts, OpenCode guidance, migrations, and documentation rewrites beyond one reviewable slice.
