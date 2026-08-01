# Managed Agent Configuration

Agent adapters are implemented and tested as application components, but they are **not wired into the public CLI composition yet**. Do not run guessed `docmanager agent configure` commands against personal configuration: the CLI currently returns placeholder inspection results or an unsupported operation instead of constructing real adapters.

## What the implemented adapters guarantee

Each adapter independently reports these dimensions:

| Dimension | Meaning |
| --- | --- |
| `installed` | The agent executable/discovery input is present. |
| `supported` | Its configuration matches a provenance-pinned, versioned fixture shape. |
| `configured` | The owned Docmanager MCP entry and managed guidance match. |
| `healthy` | A configured installed agent passes the bounded argv-only `docmanager mcp --version` probe. |

Configuration is serialized with a lock, atomically written, and refused on malformed input, unsafe routes, ownership conflicts, or drift. Existing user MCP entries are preserved. Dry-run and JSON result contracts are implemented in the application service; no current public CLI invocation performs the real adapter mutation.

## Managed scope by agent

| Agent | Configuration route | Docmanager-owned scope |
| --- | --- | --- |
| OpenCode | XDG `opencode.json` or `opencode.jsonc` | `mcp.docmanager` plus `docmanager_guidance`. |
| Codex | `~/.codex/config.toml` | `[mcp_servers.docmanager]` plus `.codex/AGENTS.md` managed guidance. |
| Claude Code | `~/.claude.json` | `mcpServers.docmanager` plus `.claude/CLAUDE.md` managed guidance; config mode is `0600`. |
| GitHub Copilot | VS Code User `mcp.json` | `servers.docmanager` plus `docmanager.instructions.md`. |
| Pi | `~/.pi/agent/mcp.json` | `mcpServers.docmanager` plus managed guidance. |

Only entries marked `_docmanager: "managed/v1"` with the expected binary and `mcp` arguments can be removed. If a user changes that entry, configuration and unconfiguration stop with drift rather than overwrite it. Ownership checks, locks, and atomic writes protect managed mutations; inspect a reported conflict manually rather than deleting configuration.

## Pi prerequisite

Pi configuration requires an already available Pi MCP adapter. Docmanager does **not** run npm, install packages, or fetch an adapter. Missing prerequisites are reported as unsupported rather than repaired automatically.

## Safe operator workflow when CLI wiring lands

The documented future workflow is: detect/status first, inspect the plan with dry-run JSON, configure one supported agent, verify status, and unconfigure only the owned entry when needed. This is a workflow description, not a command promise until the composition root exposes it.

No agent adapter enables repository hooks. Repository hooks require separate explicit consent through [`workspace install --enable-hook`](compatibility.md#workspace-commands-and-aliases).

## Fixture policy

Adapter support is deliberately gated by versioned fixture provenance. A new agent configuration shape is unsupported until its route, ownership shape, and behavior are added as a versioned fixture and tests prove configure/status/unconfigure without changing unrelated content. This conservative policy prevents schema drift from becoming destructive automation.
