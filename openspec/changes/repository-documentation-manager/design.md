# Design: Repository Documentation Manager

## Technical Approach

After a macOS/Linux/Windows spike, ship a Go binary with pure-Go SQLite and Tier-1 Go MCP SDK. It resolves scope, creates evidence, returns `update`, `create`, or `no-impact`, records receipt, and proposes (never applies) a patch.

## Architecture Decisions

| ADR | Options / trade-off | Decision and rationale |
|---|---|---|
| Runtime | Go: Tier-1 MCP/runtime-free; Rust: stronger types but Tier-2 MCP/higher cost; TypeScript: Tier-1 MCP but Node/Bun surface | **Go, conditional on spike.** Best single-binary/MCP fit; reassess Rust if it fails. |
| Boundaries | Shared adapters; domain/application ports | **Ports/adapters.** `domain` owns rules, `app` use cases, adapters I/O; CLI/MCP share evidence semantics. |
| Truth and state | SQLite truth; Git truth + cache/ledger | **Git authoritative.** SQLite is rebuildable inventory, analysis, receipt, and migration state. |
| Patch ownership | Tool applies; agent applies proposal | **Proposal only.** Target paths/unified diff may be returned; no write/apply command exists. |
| Gentle AI | Embedded integration; thin external-tool bridge | **Deferred and nonblocking.** Gentle AI only registers/lists the community tool and delegates `install --target`, `doctor --json`, and `uninstall --wiring-only` to the external binary; it embeds no runtime, duplicates no skills/guidance, and owns no data/receipts. |

## Data Flow

```text
CLI document-change / MCP document_change
              | explicit Scope
              v
       ScopeResolver -> GitPort -> CanonicalEvidence
              |                         |
              v                         v
       AnalysisEngine <--- DocumentInventory
              | outcome + rationale + proposal
              v
   LedgerPort (analysis/receipt) -> response
```

`Scope` is range, staged, or worktree—never inferred from merge state. Canonical bytes contain schema, kind, resolved identities, and sorted NUL-safe changed/documentation path/blob tuples. SHA-256 yields `diff_digest`; canonical evidence JSON yields `evidence_digest`. A successful receipt binds both to outcome and tool/schema versions. Verification requires exact match.

```text
Agent feature-close -> document-change -> receipt persisted
Git pre-push hook -> receipt verify -> pass | configured warn/fail
```

The hook uses argument arrays—never shell, LLM, or mutation. It resolves stdin ref updates; ambiguity fails closed under enforcement and warns otherwise.

## File Changes

| File | Action | Description |
|---|---|---|
| `go.mod`, `cmd/docmanager/main.go` | Create | Module and entrypoint. |
| `internal/domain/{scope,evidence,analysis,receipt,proposal}.go` | Create | Contracts, canonicalization, outcome/proposal. |
| `internal/app/{document_change,verify_receipt,install,doctor,uninstall}.go` | Create | Use cases. |
| `internal/adapters/{git,sqlite,mcp,cli}/` | Create | I/O adapters. |
| `.docmanager/` | Create at runtime | Ignored local SQLite database and receipts. |
| `assets/{AGENTS.md,skills/docmanager/SKILL.md,.githooks/pre-push}` | Create | Feature-close guidance and opt-in verifier hook. |
| `.github/workflows/{spike,ci}.yml`, `README.md`, `.gitignore` | Create/Modify | Matrix, lifecycle guidance, state exclusion. |

## Interfaces / Contracts

```go
type ScopeKind string // range | staged | worktree
type Outcome string   // update | create | no-impact
type DocumentChangeRequest struct { Repository string; Scope Scope }
type Report struct { Outcome Outcome; Evidence Evidence; Receipt Receipt; Proposal *PatchProposal }
type GitPort interface { Resolve(context.Context, string, Scope) (CanonicalEvidence, error) }
type LedgerPort interface { Save(context.Context, Report) error; Verify(context.Context, ReceiptInput) (Verification, error) }
```

MCP exposes `document_change` and optional read-only `documentation_status`; CLI owns lifecycle commands. `install` writes contained targets; `doctor` is read-only; `uninstall` removes only owned assets/state. Both adapters serialize typed errors (`invalid_scope`, `not_repository`, `git_unavailable`, `unsupported_state`, `ledger_failure`, `receipt_mismatch`, `dependency_unavailable`) without command/environment detail.

## Security and Non-Goals

Validate paths against root; reject traversal/symlink escape; run Git shell-free with bounded output/timeouts, sanitized environment, and no credential logs. Treat filenames/diffs/docs as untrusted; never execute files. No auto-apply, providers, plugins, daemon, remote sync, FTS/graph, embedded Gentle AI, or PR automation.

## Testing Strategy

| Layer | What to test | Approach |
|---|---|---|
| Unit | Canonical digests, outcomes, errors | Table fixtures; semantic input has one digest. |
| Integration | Git scopes, ledger, CLI/MCP parity | Temp Git repos and stdio fixtures. |
| Cross-platform | Go/MCP/SQLite spike, hook | macOS/Linux/Windows builds, tests, package smoke. |

## Threat Matrix

| Boundary | Applicability | Safe/failure behavior | Planned RED tests |
|---|---|---|---|
| Documentation-like paths | Applicable: classifier | Never execute; classify tracked paths. | `requirements.txt`, `CMakeLists.txt`, executable Markdown/MDX, `README.sh`. |
| Git repository selection | Applicable: caller repository | Resolve root; reject outside/non-repo. | `git -C`, relative, absolute. |
| Commit state | Applicable: staged/worktree | Preserve identities; empty scope is typed. | staged, `commit -a`, empty index. |
| Push state | Applicable: hook | Resolve updates; enforcement rejects ambiguity. | tracking, first push, explicit refspec. |
| PR commands | N/A: no PR automation | No PR command is generated or run. | N/A. |

## Migration / Rollout

No data migration. Gate implementation on the spike, then publish binaries and opt-in assets. `doctor` validates Git/platform/database/hook; `uninstall` is reversible. Roll back registration/assets and `.docmanager/` only.

## Open Questions

- [ ] Which Tier-1 Go MCP SDK and pure-Go SQLite driver pass the spike's binary-size and platform criteria?
- [ ] Should hook default mode warn or fail for the first release?
