## Exploration: docmanager-installer-agent-integration

### Current State
`docmanager` is a Go 1.26 CLI and stdio MCP server with a layered `domain` → `app` → adapter shape. Its flat CLI exposes `mcp`, `hook-verify`, `document-change`, `verify`, `install`, `doctor`, and `uninstall`. The last three operate only on an exact Git root and write/validate/remove an owned repository-local `.docmanager` directory containing embedded guidance, a skill, and declarative hook metadata. They reject traversal, symlinks, unowned state, and altered assets; creation rolls back when ownership creation fails.

The MCP server is intentionally read-only and provides `document_change` and `verify_receipt`. Existing tests cover temporary Git repositories, ownership safety, receipts, hooks, CLI flows, MCP, and SQLite. CI tests and CGo-free cross-builds Linux/amd64, Darwin/arm64, and Windows/amd64, but it does not package, sign, publish, install, or upgrade releases. The module and all internal imports still use `github.com/gentleman-programming/repository-documentation-manager`, not the canonical `github.com/desatatufuria/mcp-doc-manager` identity.

The existing `install` name is repository wiring, not binary distribution. Reusing it for a downloader would silently change a working contract and couple machine state to repository state.

### Affected Areas
- `go.mod` and all Go imports — correct the canonical module identity before published consumers install the binary.
- `cmd/docmanager/main.go` — replace the flat lifecycle surface with a stable, noninteractive command hierarchy while retaining compatible aliases during migration.
- `internal/app/lifecycle.go` — preserve repository-local ownership checks as a dedicated workspace-wiring service, separate from machine installation and release update logic.
- `internal/adapters/{agent,release,filesystem}/` (new) — isolate agent detection/config schemas, signed manifest retrieval, locking, atomic files, and process execution behind ports.
- `assets/` — evolve generic embedded guidance into versioned, agent-specific managed content and optional hook templates.
- `.github/workflows/ci.yml` and new release workflow/configuration — produce Linux/Darwin amd64/arm64 archives, checksum manifest, signature, provenance, and publishable release assets.
- `cmd/docmanager/main_test.go`, `internal/app/lifecycle_test.go`, and new fixture-driven adapter tests — retain current behavior and add strict RED-GREEN-REFACTOR coverage for configuration, rollback, upgrade, and compatibility cases.
- `README.md` and `TESTING.md` — document the headless contract and migration; do not make a TUI a release dependency.

### Approaches
1. **Preserve flat commands and add flags** — Extend `install`, `doctor`, and `uninstall` with machine/agent flags.
   - Pros: Small apparent CLI diff; current scripts keep their command names.
   - Cons: Ambiguous ownership and destructive scope; unsafe migration from repository setup to binary distribution; difficult to express status, rollback, and future TUI actions consistently.
   - Effort: Medium initially, High to maintain.

2. **Resource hierarchy with compatibility aliases** — Use noun-first commands such as `release install|upgrade|status`, `agent detect|configure|unconfigure|status`, `workspace install|doctor|uninstall`, plus top-level `doctor`; retain current flat commands as documented deprecated aliases for one major release.
   - Pros: Separates machine, agent, and repository ownership; supports dry-run/JSON uniformly; maps directly to application use cases and a future TUI; enables a stable Gentle AI orchestration contract.
   - Cons: More commands and a temporary compatibility layer.
   - Effort: High, but bounded and reviewable.

3. **Delegate all installation and agent wiring to Gentle AI** — Keep only the MCP binary here.
   - Pros: Less immediate implementation.
   - Cons: Violates independent-project and headless-community-tool requirements; makes agent setup unavailable without Gentle AI; splits ownership of user configuration.
   - Effort: Low now, High integration risk later.

### Recommendation
Choose **resource hierarchy with compatibility aliases**. Define a headless JSON contract around typed requests/results, action plans, status dimensions, and stable error codes; the CLI renders that contract, and a future TUI calls the same application services. Keep repository wiring as `workspace`, reserve `release` for binary acquisition/update, and add `agent` for adapters. Compatibility aliases should preserve the current `install --target`, `doctor --target`, and `uninstall --target` semantics exactly, emit a deprecation warning only on human output, and remain until the next major version.

Use a signed release manifest from the first release: versioned canonical JSON listing archive URL, platform tuple, SHA-256, size, and release version; verify its detached Ed25519 signature against an embedded trusted public key before downloading an archive; verify archive checksum before extraction; atomically replace only a docmanager-owned binary under a lock; preserve the prior binary and state for rollback. The shell installer should only bootstrap the verified binary. The application owns detection, plans, configuration, snapshots, validation, and ownership state.

Agent adapters should be data-driven and fixture-first. Each adapter independently reports `installed` (executable/app discoverable), `supported` (platform/version/config schema accepted), `configured` (a matching docmanager-owned MCP entry and guidance reference), and `healthy` (configured entry resolves to the expected binary and a bounded MCP/command probe succeeds). It must mutate only a namespaced managed entry, snapshot every changed file with mode and digest, write sibling temporary files then rename atomically, serialize access with a per-scope lock, and restore only the exact owned snapshot on rollback/unconfigure. Unknown, malformed, externally edited, symlinked, or concurrently changed config must fail without overwrite and be reported diagnostically.

Model OpenCode, Codex, Claude Code, GitHub Copilot, and Pi as explicit adapters with versioned schema fixtures rather than a generic JSON editor. Detection may be common, but each adapter owns discovery paths, config format, merge rules, launcher command, guidance location, and health probe. Optional repository hooks remain a `workspace` operation and must be opt-in; they must not be enabled by agent configuration or binary installation.

For delivery, split the work into chained review units: module/CLI contract and core state; release trust/update; adapter framework plus one reference adapter; remaining adapters and fixtures; migration/docs/acceptance. This is substantially above the 800-line review budget if delivered as one change.

### Risks
- Agent configuration schemas and locations change independently. Pin representative real-world fixtures per supported version, reject unrecognized shapes, and test no-op/dry-run/configure/unconfigure/rollback for every adapter.
- A signing design without an explicit key rotation and compromise policy can make updates permanently unsafe or permanently unavailable. Specify trusted-key identifiers, overlap rotation, manifest expiry, and an offline/manual recovery path.
- Replacing the running binary differs across platforms and can be blocked by permissions, locks, or package-manager ownership. Never overwrite a non-owned target; report remediation and retain the previous binary until post-install health succeeds.
- Current lifecycle operations and SQLite ledger share `.docmanager`; moving it without a migration can break receipt verification. Preserve the repository ledger location and version its ownership metadata separately from machine-level state.
- Network access, executable discovery, and agent config edits expand the threat boundary. Apply URL allowlisting, HTTPS, bounded downloads, no shell interpolation, restrictive file modes, symlink refusal, redacted diagnostics, and timeouts.
- No release packaging or agent fixtures exist today. Strict TDD means each contract and failure mode needs a RED test before its implementation; integration tests will need isolated home/config environments and fake release servers.

### Ready for Proposal
Yes. No product decision blocks a proposal: the proposal can establish the default signed-manifest trust root, resource hierarchy, ownership model, and adapter fixture policy. The only implementation inputs that must be verified before an adapter is declared supported are the current schema/location fixtures and health command for each named agent. The proposal should explicitly keep Windows out of the first distribution matrix despite current CI coverage, retain the current MCP tool contract, and state that a future TUI consumes application services rather than CLI parsing.
