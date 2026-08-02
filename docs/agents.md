# Managed Agent Configuration

OpenCode is the only agent wired into guided onboarding in this first slice. From an exact Git root, run `docmanager install`; Docmanager detects `opencode`, shows a plan, initializes repository state, and configures the local MCP server after confirmation. Other agents are not supported by this onboarding command.

## What the implemented adapters guarantee

Each adapter independently reports these dimensions:

| Dimension | Meaning |
| --- | --- |
| `installed` | The agent executable/discovery input is present. |
| `supported` | Its configuration matches a provenance-pinned, versioned fixture shape. |
| `configured` | The exact Docmanager local MCP entry matches. |
| `healthy` | A configured installed agent passes the bounded argv-only `docmanager mcp --version` probe. |

Configuration is serialized with a lock, atomically written, and refused on malformed input, unsafe routes, or ownership conflicts. Existing user keys and MCP entries are preserved. `--dry-run` inspects without writing; `--json` requires either `--dry-run` or `--yes` and never prompts.

## Managed scope by agent

| Agent | Configuration route | Docmanager-owned scope |
| --- | --- | --- |
| OpenCode | `os.UserConfigDir()/opencode/opencode.json` or `opencode.jsonc` | `mcp.docmanager` with `type: local`, the absolute Docmanager command array, and `enabled: true`. |
| Codex | `~/.codex/config.toml` | `[mcp_servers.docmanager]` plus `.codex/AGENTS.md` managed guidance. |
| Claude Code | `~/.claude.json` | `mcpServers.docmanager` plus `.claude/CLAUDE.md` managed guidance; config mode is `0600`. |
| GitHub Copilot | VS Code User `mcp.json` | `servers.docmanager` plus `docmanager.instructions.md`. |
| Pi | `~/.pi/agent/mcp.json` | `mcpServers.docmanager` plus managed guidance. |

Docmanager removes only the exact structural entry it generates. If `mcp.docmanager` already differs, configuration and unconfiguration stop with an ownership conflict rather than overwriting it. JSONC leading comments are preserved; non-leading line or block comments are rejected conservatively.

## Pi prerequisite

Pi configuration requires an already available Pi MCP adapter. Docmanager does **not** run npm, install packages, or fetch an adapter. Missing prerequisites are reported as unsupported rather than repaired automatically.

## Safe operator workflow

Use `docmanager install --dry-run` to inspect the plan, then `docmanager install` for confirmation or `docmanager install --yes` for non-interactive application. Verify the result with `opencode mcp list`.

No agent adapter enables repository hooks. Repository hooks require separate explicit consent through [`workspace install --enable-hook`](compatibility.md#workspace-commands-and-aliases).

## Fixture policy

Adapter support is deliberately gated by versioned fixture provenance. A new agent configuration shape is unsupported until its route, ownership shape, and behavior are added as a versioned fixture and tests prove configure/status/unconfigure without changing unrelated content. This conservative policy prevents schema drift from becoming destructive automation.
