# Design: Docmanager Installer and Agent Integration

## Technical Approach

Migrate `go.mod`/imports to `github.com/desatatufuria/mcp-doc-manager`; preserve MCP tools/results, receipts, and `.docmanager/ledger.db`. Resource-first services own separate state. CLI renders for future TUI/Gentle AI orchestration, with no runtime or ownership dependency.

## Architecture Decisions

| Decision | Choice and rationale |
|---|---|
| Boundaries/CLI | Add `internal/app` use cases/ports and release/filesystem/agent adapters, retaining domain → app → adapter. Closed resources are `release`, `agent`, `workspace`; retain human-only deprecated `install|doctor|uninstall --target` aliases for one major version. |
| Identity | `internal/app/identity.go` validates manifest/installer identity; legacy input returns `legacy_identity`. RED test it; MCP, receipt, and ledger regressions remain unchanged. |
| Agent support | Explicit adapters own discovery, route, merge, launcher, guidance, probe. Fixtures: OpenCode XDG `opencode.json/jsonc`/`mcp.<name>`; Codex `~/.codex/config.toml`/`mcp_servers.<name>`; Claude mode-0600 `~/.claude.json`/`mcpServers`; Copilot VS Code User `mcp.json`/`servers`; Pi `~/.pi/agent/mcp.json`/MCP adapter. `supported=true` requires provenance-pinned inputs. |
| State/trust | Versioned state is outside `.docmanager`; locks, `Lstat`, digest/mode backups, fsynced temp rename, drift refusal. Manifest: `schema,key_id,issued,expires,artifacts[{version,os,arch,url,size,sha256}]`; embedded Ed25519 policy authorizes overlap/revocation. Verify signature, expiry, allowlisted HTTPS, size/digest before extraction; bootstrap verifies, never evaluates shell. |

## Complex Flows

```text
upgrade -> lock -> fetch manifest -> verify trust/artifact -> snapshot owned binary
 -> atomic replace -> health -> commit state
                         \-> failure: restore snapshot -> rollback_error/result
```
```text
configure/unconfigure -> validate route/fixture -> lock -> snapshot owned files
 -> atomic config+guidance commit -> health -> commit state
                                  \-> failure: restore snapshot; no destructive mutation
```

`workspace` requires exact Git root; hooks need explicit opt-in and report absent/opted-in/drifted. Agents never enable hooks. Archives are `docmanager_<version>_<linux|darwin>_<amd64|arm64>.tar.gz`; Windows is rejected.

## Interfaces / Contracts

```go
type Operation string
type Change struct { Kind, Target, Before, After string }
type AgentStatus struct { Agent string; Installed, Supported, Configured, Healthy bool; Detail string }
type ReleaseStatus struct { Installed, Healthy bool; Version, Platform string }
type WorkspaceStatus struct { Root, State, Hook string }
type Status struct { Agent *AgentStatus; Release *ReleaseStatus; Workspace *WorkspaceStatus }
type Result struct { Operation Operation; Target, Outcome string; Changes []Change; Status *Status; Error *StableError }
type StableError struct { Code, Classification string; Detail string }
```

| Closed operation | Required | Optional | Forbidden |
|---|---|---|---|
| `release.install|upgrade` | `manifest_url,install_dir` | `version` | agent/workspace fields |
| `release.rollback`; `release.status|doctor` | rollback: `install_dir`; status/doctor: — | status/doctor: `install_dir` | rollback: manifest/version; status/doctor: mutation fields; agent/workspace |
| `agent.configure`/`unconfigure`; `detect|status|doctor` | configure: `agent,binary`; others: `agent` | configure/unconfigure: `config_root,dry_run`; read-only: `config_root` | unconfigure/read-only: `binary`; read-only: `dry_run`; release/workspace |
| `workspace.install`; `uninstall`; `doctor|status` | `target` | install: `enable_hook,dry_run`; uninstall: `dry_run` | read-only: `enable_hook,dry_run`; otherwise release/agent fields |
| aliases | `install|doctor|uninstall --target` | `--json`; matching workspace operation | resource/release/agent fields |

Unknown/missing/incompatible/read-only mutation fields return `invalid_input` before side effects. Services take `context.Context`; JSON success=`success`, status=`status` plus union, dry-run=`planned` without writes, failure=`failure` plus stable `invalid_input`, `legacy_identity`, `unsupported_platform`, `unsupported_agent`, `malformed_config`, `drift`, `locked`, `manifest_verification`, `ownership`, `probe_failed`, or `rollback_failed`. Installed=discoverable; supported=verified gate; configured=owned MCP/guidance match; healthy=expected binary plus bounded argv-only probe.

## File Changes and Testing

| File | Action | Description |
|---|---|---|
| `go.mod`, `**/*.go`; `cmd/docmanager/main.go` | Modify | Canonical imports; routing, aliases, JSON; preserve MCP/ledger. |
| `internal/app/{identity,release,agent,workspace,contract}.go` | Create/Modify | Typed services/contracts. |
| `internal/adapters/{release,filesystem,agent}/`, `testdata/agent/` | Create | Trust/state adapters and provenance fixtures. |
| `assets/`, `.github/workflows/`, `**/*_test.go` | Modify/Create | Guidance, signed releases, strict tests. |

RED-first table tests use `t.TempDir`, fake clock/HTTP/runner, isolated XDG/HOME/VS Code; external commands skip under `testing.Short()`. Cover contracts, trust, rollback, adapters/gates, aliases/hooks, MCP/receipt/SQLite.

## Threat Matrix

| Boundary | Applicability | Safe failure and RED expectation |
|---|---|---|
| Download trust | Applicable | Allowlist HTTPS; reject bad signature/expiry/revocation, redirect, oversize, digest mismatch; RED each. |
| Archive extraction | Applicable | Isolated temp only; reject traversal/absolute, symlink/hardlink, duplicate, oversized/decompression-limit, unexpected entries; accept only manifest-authorized regular binary/authorized metadata; never replace; RED each. |
| Subprocess/probe | Applicable | `exec.CommandContext` argv only, bounded output/timeout; literal metacharacters, timeout/failure/wrong executable give `probe_failed` without writes. |
| Executable replacement | Applicable | Refuse symlink/non-owned target; failed health restores prior binary; RED each. |
| XDG/HOME/VS Code/Git routing | Applicable | Exact configured root; reject route escape/symlink and non-root relative/absolute Git selectors; RED each. |
| Documentation-like paths | N/A — fixed embedded assets are not classified/executed. | N/A |
| Commit state | N/A — no commit operation. | N/A |
| Push state | N/A — no push operation. | N/A |
| PR commands | N/A — no PR operation. | N/A |

## Migration, Risks, and Gates

Publish signed Linux/macOS amd64/arm64 archives, then bootstrap. Upgrade retains prior owned binary/state until health; rollback restores it. Offline recovery requires a manually verified trusted binary, never bypass. Risks: schema drift (pinned fixtures); key compromise/rotation (revocation, overlap, expiry, recovery); unsafe replacement/platform constraints (ownership, locks, rollback); route/probe failure (no mutation). Non-blocking gates: record per-agent fixture provenance, discovery, launcher, and probe evidence before enabling support.
