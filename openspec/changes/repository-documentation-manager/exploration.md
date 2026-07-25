## Exploration: repository-documentation-manager

### Current State
This is a greenfield repository: it has no commits, source files, manifests, tests, CI, remote, or established language. OpenSpec is the only project content. Product constraints are already clear: a standalone, repository-only tool; Git is authoritative; agents invoke one feature-close operation; the tool returns evidence-backed `update`, `create`, or `no-impact` and never writes documentation in the MVP.

The MVP is a local CLI plus MCP stdio server. The MCP project documentation classifies Go and TypeScript SDKs as Tier 1 (fully supported), while Rust is Tier 2 (actively maintained toward full support). All three can implement stdio servers, but only Go and Rust naturally deliver a runtime-free single executable.

### Affected Areas
- `openspec/changes/repository-documentation-manager/exploration.md` — exploration artifact for the new product change.
- `go.mod`, `cmd/`, `internal/` (new) — likely Go module and layered CLI/MCP/domain implementation if the recommendation is approved.
- `.docmanager/` (new, repository-local and ignored or user-configurable) — future SQLite state, receipts, and checkpoints; not source documentation.
- `docs-manager` agent guidance / `AGENTS.md` snippets (new distribution assets) — explicit invocation at feature close; the product owns these assets, not Gentle AI.
- `.githooks/` (optional, new) — deterministic receipt verification only; hooks must not call an LLM or mutate documentation.

### Approaches
1. **Go single binary with an adapter boundary** — Implement core Git analysis, evidence/receipt storage, and CLI in Go; expose the same application service through Cobra-style CLI commands and a Go Tier-1 MCP stdio adapter.
   - Pros: Native single-binary distribution and straightforward cross-compilation; official Tier-1 Go MCP support; direct process/Git integration; predictable startup; one static deployment artifact per target; good fit for a thin optional Gentle AI external-tool integration. Go's simple test tooling and interfaces suit deterministic Git/receipt tests.
   - Cons: SQLite/FTS needs a driver choice: CGO-backed SQLite complicates portable builds, while pure-Go SQLite increases binary size and must be benchmarked; less compile-time domain enforcement than Rust.
   - Effort: Medium.

2. **Rust single binary** — Build the same core with Cargo, `rmcp`, and a Rust SQLite stack.
   - Pros: Excellent single-binary distribution, low startup/memory use, strong type safety, mature SQLite ecosystem, and strong cross-platform packaging options.
   - Cons: The MCP SDK is currently Tier 2 rather than Tier 1; async/protocol integration and contributor learning curve add maintenance cost for a product whose complexity is mainly Git semantics, evidence, and workflow rather than CPU-bound processing.
   - Effort: Medium-High.

3. **TypeScript with Bun/Node packaging** — Build a TypeScript core with the Tier-1 TypeScript MCP SDK, distributed through a runtime or compiled per-platform executable.
   - Pros: Most mature/visible MCP SDK ecosystem; fast agent-integration iteration; familiar schemas and fixtures; Bun can produce executables.
   - Cons: Node requires a managed runtime; Bun's compiled artifacts remain platform-specific and introduce an additional toolchain/runtime compatibility surface. Native Git and SQLite packaging, startup, and long-term support are less operationally simple than Go/Rust for a repository-local binary. This is a poor fit for the stated single-binary-first distribution requirement.
   - Effort: Medium.

### Recommendation
Choose **Go**, subject to a small implementation-phase spike that validates the selected Tier-1 Go MCP SDK and a pure-Go SQLite driver on macOS, Linux, and Windows. It wins the current tradeoff rather than a user preference: Go combines Tier-1 MCP support with runtime-free single-binary delivery, low operational burden, direct Git subprocess support, cross-platform builds, and approachable testing/maintenance. Rust is the credible alternative if profiling or a future reliability/security requirement justifies its higher engineering cost; TypeScript/Bun should not be the MVP runtime because distribution is a first-class product constraint.

The MVP boundary should be intentionally narrow:

| Area | MVP decision |
|---|---|
| Public operation | `document_change` (provisional) analyzes a caller-supplied Git range or staged/worktree change and returns a structured impact report. |
| Outcomes | Exactly `update`, `create`, or `no-impact`, each with paths, Git evidence, rationale, confidence, and receipt/checkpoint data. |
| Write boundary | The tool proposes patches/targets but never applies documentation edits. The invoking agent owns review and patch application. |
| Git boundary | Git refs, tracked content, and diff identity are authoritative. Do not infer feature completion from merge state. |
| MCP/CLI surface | CLI: `document-change`, `receipt verify`, `install`, `doctor`, `uninstall`; MCP: `document_change` plus read-only `documentation_status` if needed. Keep install/doctor/uninstall as CLI-owned operations. |
| Providers | Built-in repository/Git provider only. Provider interfaces are internal seams; do not ship plugin loading, Confluence, daemon, or remote sync. |
| Gentle AI | Optional thin community-tool registration that delegates install/doctor/uninstall to the external binary. No embedded runtime or duplicated skill body. |

Use a repository-local SQLite database only for inventory metadata, analysis records, and receipts; it is a cache/ledger, never a source of truth. A minimum model is: `Document(path, content_oid, role)`, `Analysis(id, base_ref, head_ref, diff_digest, tool_version, outcome, evidence_json, created_at)`, and `Receipt(analysis_id, scope, digest, status)`. Add FTS only when measured document-discovery quality requires it; the initial analysis can derive candidates from changed paths, repository conventions, and direct content scans. This avoids binding the MVP to a native FTS extension or a prematurely complex documentation graph.

Receipts must be content-bound: canonicalize the selected Git range plus relevant documentation blob IDs, hash it, and record the tool/schema version, outcome, evidence digest, and timestamp. A deterministic hook recomputes that identity and accepts only a matching successful receipt. The hook may warn or fail according to configuration, but it must neither run an LLM nor invoke `document_change`; explicit skill/AGENTS.md invocation remains the control point.

### Risks
- Git range selection is ambiguous for staged, uncommitted, rebased, squashed, or partially merged work; the public contract must require an explicit scope and report the resolved refs/digest.
- A receipt that omits documentation blob identity or tool/schema version can validate a stale analysis; the canonical digest contract must be specified and regression-tested before hooks exist.
- Pure-Go SQLite, MCP SDK APIs, binary size, and Windows/macOS/Linux packaging are assumptions until the proposed compatibility spike runs in CI.
- Heuristic impact classification can produce false positives/negatives. Evidence must be inspectable, and agents—not the tool—remain responsible for applying proposed changes.
- Optional provider/plugin loading and a daemon would dilute the MVP boundary; retain only internal interfaces until a concrete second provider or background workload proves the need.

### Ready for Proposal
Yes — propose the standalone Go-based MVP with an explicit cross-platform compatibility spike and a contract-first definition of Git scope, evidence, receipt digest, and the provisional `document_change` schema. Keep TypeScript/Bun and Rust recorded as rejected-for-now alternatives, not as prohibited future implementations.
