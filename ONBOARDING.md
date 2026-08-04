# Repository Documentation Manager: Beginner Onboarding

`docmanager` answers one narrow question: **given an explicit set of Git changes, which documentation paths deserve human attention?** It returns a deterministic JSON report and a content-bound receipt. It does not write documentation, stage files, commit, push, call an AI model, or understand program semantics.

This guide starts with the shortest successful path, then explains receipts, lifecycle state, pre-push verification, MCP, troubleshooting, architecture, and team adoption.

> **Current distribution boundary:** this repository does not establish a published installer, package registry, or release download. Build `docmanager` from source. Do not assume that `go install`, Homebrew, Winget, or another package channel is available.

## Quick Path

You need a source checkout, Go 1.26 or newer, Git, and an existing Git repository with at least one commit.

```bash
SOURCE=/absolute/path/to/repository-documentation-manager
REPOSITORY=/absolute/path/to/your/repository

mkdir -p "$HOME/.local/bin"
go -C "$SOURCE" build -trimpath -buildvcs=false -o "$HOME/.local/bin/docmanager" ./cmd/docmanager

"$HOME/.local/bin/docmanager" install --target "$REPOSITORY"
"$HOME/.local/bin/docmanager" doctor --target "$REPOSITORY"
"$HOME/.local/bin/docmanager" document-change --repo "$REPOSITORY" --scope staged
```

Successful `install` is silent. Successful `doctor` prints:

```json
{"git":true}
```

`document-change` requires a non-empty scope. If nothing is staged, use a scope that actually contains changes or expect an error.

## The Mental Model

Think of `docmanager` as a **documentation-impact checkpoint**, not a documentation author.

```text
You select one Git scope
          |
          v
Git adapter resolves exact changed paths and content identity
          |
          v
Path-based analysis returns an outcome, candidates, and rationale
          |
          v
A content-bound receipt is saved in the local SQLite ledger
          |
          v
CLI, MCP, or an optional pre-push hook can verify that receipt later
```

### What problem it solves

Feature work often reaches review with an unanswered question: "Did this change require documentation work?" `docmanager` makes that checkpoint explicit and inspectable. It records which Git content was analyzed and which documentation bytes were considered, so a later verification can detect stale or altered evidence.

### What it does not do

`docmanager` does **not**:

- Generate, edit, patch, stage, commit, or push documentation.
- Run an LLM, agent, or semantic source-code analysis.
- Infer a scope when none is supplied.
- Decide that documentation is correct or complete.
- Install a Git executable hook automatically.
- Configure an MCP client automatically.
- Publish or download its own executable.

Its current analysis is deterministic and path-based. A human or external agent must inspect the report, decide whether the recommendation makes sense, and make any documentation edits separately.

### Seven terms that are easy to confuse

| Term | Plain-language meaning | Mutates repository state? |
| --- | --- | --- |
| **Analysis** | Resolves one explicit Git scope and applies path heuristics to return `update`, `create`, or `no-impact`. | Creates or updates only `.docmanager/ledger.db`; never edits tracked documentation or Git state. |
| **Receipt** | JSON proof binding the selected scope, changed paths, content identity, considered documentation digests, outcome, and schema/tool versions. | The object itself is data; successful analysis stores it in the ledger. |
| **Ledger** | A local, rebuildable SQLite cache of successful receipts at `.docmanager/ledger.db`. Git remains authoritative. | Yes, local `.docmanager` metadata only. |
| **Verification** | Re-resolves the same scope and checks the supplied receipt digest, current evidence, documentation bytes, outcome, and stored ledger copy. | Read-only when the ledger already exists. |
| **MCP** | A stdio server exposing the two read-only product operations to an MCP client: analyze and verify. | Analysis still writes ledger metadata; neither tool writes documentation or Git state. |
| **Lifecycle** | CLI-only `install`, `doctor`, and `uninstall` management of owned repository-local assets. | `install` and `uninstall` change `.docmanager`; `doctor` is read-only. |
| **Hook** | Optional pre-push receipt enforcement. The installed asset is a JSON declaration; a real `.git/hooks/pre-push` executable must be wired manually. | Verification is read-only; manual wiring creates a Git hook file. |

> **Read-only means documentation/Git read-only.** Analysis persists a receipt in `.docmanager/ledger.db`. It does not change tracked files, the index, commits, refs, or remotes.

## Requirements

### Runtime requirements

| Requirement | Required? | Current evidence and notes |
| --- | --- | --- |
| Git executable | Yes | Tested locally for this guide with Git 2.39.5. The code declares no minimum version. Git must be on `PATH`. |
| Exact repository root | Yes | `--repo` and `--target` must identify the top-level Git directory, not a nested directory. Absolute paths are strongly recommended and required by the MCP schema. |
| Writable repository root | For analysis/install | The process needs permission to create or update `.docmanager`. Verification needs an existing readable ledger. |
| SQLite runtime or CLI | No | SQLite is embedded through pure-Go `modernc.org/sqlite` v1.55.0. No `sqlite3` program or CGo runtime is required. |
| MCP client with stdio transport | Only for MCP | The client must be able to launch a local command and exchange MCP messages over standard input/output. |
| `jq` | Optional | Used only to extract and inspect JSON in examples. Tested here with jq 1.6. |
| Bash | Optional | Used by the copy/paste Unix-like tutorials and optional hook wiring. The executable itself does not require Bash. |

### Build, development, and testing requirements

| Requirement | Required? | Authoritative version or use |
| --- | --- | --- |
| Go | Yes to build/test | `go.mod` declares Go 1.26.0. CI uses Go 1.26.5; this guide was rehearsed with Go 1.26.5. |
| Git | Yes | Build metadata can be disabled with `-buildvcs=false`; tests and product behavior still exercise Git. |
| C compiler / CGo | No | CI builds with `CGO_ENABLED=0`; the SQLite driver is pure Go. |
| `go test`, race detector, `go vet`, `gofmt` | For contributors | CI runs `go test ./... -count=1`, focused race tests, `go vet ./...`, and formatting checks. |

### Platform support demonstrated by CI

CI performs native CGo-free package smoke builds on:

- Linux, amd64.
- macOS, arm64.
- Windows, amd64.

It also cross-builds those same target combinations from Linux. This is the tested matrix, not a claim that every operating system and architecture works. The Bash tutorials and POSIX hook wrapper below target Unix-like shells. Windows users can run the executable natively but must translate tutorial file operations and optional hook wiring to their environment.

## From Zero: First Repository and Installation

This first example creates a repository with app code and documentation, commits a baseline, builds `docmanager` from source, and installs owned local assets.

### 1. Create and commit a baseline

```bash
DEMO=/absolute/path/to/a/new/docmanager-demo

mkdir -p "$DEMO/cmd"
git -C "$DEMO" init -b main
git -C "$DEMO" config user.name "Onboarding Learner"
git -C "$DEMO" config user.email "learner@example.invalid"

cat > "$DEMO/cmd/server.go" <<'EOF'
package main

import "fmt"

func main() {
	fmt.Println("API version 1")
}
EOF

cat > "$DEMO/README.md" <<'EOF'
# Demo API

Run the server to print the current API version.
EOF

git -C "$DEMO" add README.md cmd/server.go
git -C "$DEMO" commit -m "Create baseline app and documentation"
```

The baseline commit matters. `staged` identity is calculated relative to `HEAD`, so an unborn repository is not a useful starting point for the everyday CLI workflow.

### 2. Build from the source checkout

```bash
SOURCE=/absolute/path/to/repository-documentation-manager
mkdir -p "$HOME/.local/bin"
CGO_ENABLED=0 go -C "$SOURCE" build -trimpath -buildvcs=false -o "$HOME/.local/bin/docmanager" ./cmd/docmanager
```

Optionally add `$HOME/.local/bin` to your shell `PATH`. Every guide command also works with the full executable path.

### 3. Install repository-local assets

```bash
"$HOME/.local/bin/docmanager" install --target "$DEMO"
"$HOME/.local/bin/docmanager" doctor --target "$DEMO"
```

Expected output:

```json
{"git":true}
```

> **Install before the first analysis.** A direct analysis can create `.docmanager/ledger.db` without the lifecycle ownership marker. A later `install` will correctly refuse that unowned directory with `invalid_target`. Installing first creates the ownership boundary needed by `doctor` and safe `uninstall`.

## What `install` Owns

`install --target <root>` requires the exact root of an existing Git repository. It creates `.docmanager` with restrictive permissions where the platform supports POSIX modes, then writes these embedded assets:

```text
.docmanager/
|-- .owned
|-- config.json
|-- guidance/
|   |-- AGENTS.md
|   `-- skills/
|       `-- docmanager/
|           `-- SKILL.md
`-- hooks/
    `-- pre-push.json
```

After the first successful analysis, the directory also contains:

```text
.docmanager/ledger.db
```

| Path | Exact purpose |
| --- | --- |
| `.docmanager/.owned` | Contains `docmanager/1` plus a newline. It proves lifecycle ownership for reinstall, doctor validation, and uninstall. |
| `.docmanager/config.json` | Contains schema `docmanager/config/v1` and default `hook_mode` value `warn`. |
| `.docmanager/guidance/AGENTS.md` | Embedded guidance telling an agent to invoke explicit feature-close analysis and apply any documentation edits itself. |
| `.docmanager/guidance/skills/docmanager/SKILL.md` | Embedded skill guidance for explicit document-change analysis. |
| `.docmanager/hooks/pre-push.json` | Declarative hook specification with command `docmanager hook-verify`, input type `pre-push-ref-updates`, mode `warn`, and schema `docmanager/hook/v1`. |
| `.docmanager/ledger.db` | Pure-Go SQLite database. Its `receipts` table stores the full receipt JSON keyed by receipt digest. It is created lazily by analysis, not by `install`. |

The exact embedded hook declaration is:

```json
{"command":["docmanager","hook-verify"],"input":"pre-push-ref-updates","mode":"warn","schema":"docmanager/hook/v1"}
```

### What installation does not do

Installation does not:

- Create or modify `.git/hooks/pre-push`.
- Set `core.hooksPath`.
- Put `docmanager` on `PATH`.
- Configure an MCP client.
- Add `.docmanager` to `.gitignore` or commit it.
- Create the SQLite ledger before an analysis needs it.
- Overwrite changed, symlinked, non-regular, or unowned state.

Re-running `install` is safe only when the existing `.docmanager` has the exact ownership marker and every existing managed asset still matches its embedded content. It adds missing managed assets, but refuses altered managed files rather than overwriting them.

## Everyday Analysis Workflows

Every analysis requires exactly one non-empty Git scope.

| Scope | Exact command shape | Use it when | What it sees |
| --- | --- | --- | --- |
| `worktree` | `docmanager document-change --repo "$REPOSITORY" --scope worktree` | You are exploring tracked edits before staging. | Modified or deleted **tracked** files compared with the index. It does not include untracked files or staged-only differences. |
| `staged` | `docmanager document-change --repo "$REPOSITORY" --scope staged` | You want to analyze exactly what is currently staged for the next commit. | Index versus `HEAD`; requires a useful `HEAD` baseline. |
| `range` | `docmanager document-change --repo "$REPOSITORY" --scope range --range 'main..HEAD'` | You want to analyze committed work, usually a branch or candidate push range. | Exactly two resolved commit endpoints using a two-dot range. Three-dot ranges, missing endpoints, and non-commit revisions are invalid. |

> `--repo` must be the exact repository root. `--range` is required only for `range` and must be absent for `staged` and `worktree`.

### Analyze a realistic API behavior change

Change `cmd/server.go` from API version 1 to API version 2, then stage it:

```bash
REPOSITORY=/absolute/path/to/docmanager-demo

cat > "$REPOSITORY/cmd/server.go" <<'EOF'
package main

import "fmt"

func main() {
	fmt.Println("API version 2")
}
EOF

git -C "$REPOSITORY" add cmd/server.go

report="$(docmanager document-change --repo "$REPOSITORY" --scope staged)"
printf '%s\n' "$report" | jq .
```

A representative response is:

```json
{
  "Outcome": "create",
  "Candidates": [
    "README.md"
  ],
  "Rationale": "A repository-facing path changed without an existing documentation update.",
  "Confidence": 0.7,
  "Evidence": {
    "Root": "/absolute/path/to/docmanager-demo",
    "Scope": {
      "kind": "staged",
      "range": ""
    },
    "Identity": "sha256:<64 hexadecimal characters>",
    "ChangedPaths": [
      "cmd/server.go"
    ]
  },
  "Receipt": {
    "digest": "sha256:<64 hexadecimal characters>",
    "scope": {
      "kind": "staged",
      "range": ""
    },
    "evidence": "sha256:<64 hexadecimal characters>",
    "changed_paths": [
      "cmd/server.go"
    ],
    "documentation": [
      {
        "path": "README.md",
        "digest": "sha256:<64 hexadecimal characters>"
      }
    ],
    "outcome": "create",
    "versions": {
      "schema": "receipt/v1",
      "tool": "docmanager/1"
    }
  }
}
```

Hashes and the root path vary by repository and content.

### Interpret the report

| Field | Meaning |
| --- | --- |
| `Outcome` | One of `update`, `create`, or `no-impact`. This is a heuristic recommendation, not an edit or policy verdict. |
| `Candidates` | Documentation paths to inspect. It may be absent or `null` for `no-impact`. |
| `Rationale` | Fixed plain-language explanation for the selected heuristic. |
| `Confidence` | Fixed heuristic score: currently 0.95 for `update`, 0.70 for `create`, and 0.85 for `no-impact`. It is not statistical or AI-generated confidence. |
| `Evidence` | Canonical Git root, resolved scope, SHA-256 content identity, and changed paths. Internal documentation digest data is intentionally omitted here. |
| `Receipt` | Complete, portable JSON proof. Top-level report fields use Go's exported names; nested receipt fields intentionally use lowercase JSON names. |

### Understand the three outcomes

| Outcome | Current trigger | Correct human response |
| --- | --- | --- |
| `update` | At least one selected path is `README.md`, any `.md`/`.mdx`, or under `docs/` (except `README.sh`). | Inspect the reported changed documentation path and confirm it is accurate and complete. |
| `create` | No selected documentation path changed, but a path heuristic marks a repository-facing candidate. Paths under `cmd/`, `requirements.txt`, `CMakeLists.txt`, `README.sh`, and many non-Go files qualify. | Inspect `README.md`. The name does **not** mean README is absent; it means the selected scope did not include a recognized documentation update. |
| `no-impact` | No selected path matches either documentation or candidate heuristics. A typical example is a `.go` file outside `cmd/`. | Confirm the path-only heuristic is reasonable for this change; override it with human judgment when behavior changed. |

The analyzer stops at the first matching category after sorting changed paths. It is deliberately simple. It does not parse APIs, compare behavior, understand prose, or know whether an existing README already documents the change.

### When analysis writes metadata

On successful analysis, the CLI and `document_change` MCP tool save the complete receipt in `.docmanager/ledger.db`. They do not edit tracked documentation.

If `.docmanager` is untracked, `git status --short` may show:

```text
?? .docmanager/
```

That is local ledger/guidance state, not a documentation edit. Decide as a team whether to ignore or version selected assets; the product does not make that decision. Failed repository-root validation occurs before ledger creation, but a valid analysis can create `.docmanager` if it is absent. Prefer `install` first so lifecycle ownership remains safe.

## Extract and Verify a Full Receipt

Verification requires the **entire** nested `Receipt` object, not only its digest.

```bash
REPOSITORY=/absolute/path/to/docmanager-demo

report="$(docmanager document-change --repo "$REPOSITORY" --scope staged)"
receipt="$(printf '%s' "$report" | jq -c '.Receipt')"

printf '%s\n' "$receipt" | jq .
docmanager verify --repo "$REPOSITORY" --scope staged --receipt "$receipt"
```

Expected successful verification:

```json
{"verified":true}
```

The jq path is `.Receipt`, while fields inside it are lowercase, for example:

```bash
printf '%s\n' "$report" | jq -r '.Receipt.digest'
printf '%s\n' "$report" | jq -r '.Receipt.scope.kind'
printf '%s\n' "$report" | jq -r '.Receipt.changed_paths[]'
```

Verification succeeds only when all of these still agree:

- The receipt's own SHA-256 digest is valid.
- The supplied CLI scope resolves to the same canonical scope and content.
- Changed paths are identical, sorted, and unique.
- Considered documentation path digests still match the selected scope.
- The current heuristic outcome is identical.
- Receipt schema and tool versions match.
- The exact receipt is present in the local ledger.

If staged content changes after analysis, the staged receipt becomes stale. Re-run analysis and use the new complete receipt. For a `range` receipt, unrelated dirty worktree edits do not change committed range evidence.

## Pre-Push Verification from Zero

Pre-push support verifies existing receipts. It does not run analysis and does not create receipts.

### What Git sends to a pre-push hook

Git invokes a pre-push hook and writes one update line per ref to the hook's standard input:

```text
<local-ref> <local-object-id> <remote-ref> <remote-object-id>
```

`docmanager hook-verify` reads those lines from stdin. It accepts at most 32 lines, each at most 4096 bytes, validates refs with Git, and determines the repository's object-ID length dynamically. This supports both SHA-1 and SHA-256 object formats rather than hard-coding 40-character IDs.

### How update modes resolve

| Push situation | Git update shape | Scope verified by `docmanager` |
| --- | --- | --- |
| Tracking push, such as `git push` | Normal four-field update supplied by Git. | `<remote-object-id>..<local-object-id>` |
| Explicit refspec, such as `git push origin HEAD:main` | Also a normal four-field update; local and remote ref names may differ. | `<remote-object-id>..<local-object-id>` |
| First push of a remote ref | Remote object ID is all zeroes. | Internal `initial` scope bound to all paths reachable from the local commit. |
| Deletion | Local ref is `(delete)` and local object ID is all zeroes. | No receipt is required for that deletion line. |
| Ambiguous or malformed input | Empty input, invalid fields/refs/IDs, overlong input, or more than 32 lines. | `warn` returns a warning; `fail` rejects with `unsupported_request`. |

Normal receipt failures are not ambiguity. A missing, stale, tampered, or mismatched receipt fails with `receipt_mismatch` in both `warn` and `fail` modes.

> **First-push limitation:** the hook can verify internal `initial` receipts, but the public CLI and MCP scope parsers expose only `range`, `staged`, and `worktree`. There is currently no supported public command that creates an `initial` receipt. Do not enable hook enforcement for a branch's first remote push unless your workflow already has a supported way to seed that receipt. This is a product gap, not something to bypass with a fabricated receipt.

### Prepare a receipt before a normal push

Fetch first so your local remote-tracking ref reflects the remote object ID Git is expected to report:

```bash
REPOSITORY=/absolute/path/to/docmanager-demo

git -C "$REPOSITORY" fetch origin
docmanager document-change \
  --repo "$REPOSITORY" \
  --scope range \
  --range 'origin/main..HEAD'
```

For another upstream, replace `origin/main` with the fetched remote-tracking ref that matches the destination. The receipt is stored automatically. If the remote advances before your push, fetch/rebase as appropriate and analyze the new exact range.

### Default warn mode versus fail mode

The embedded declarations default to `warn`:

- `warn` allows only **ambiguous hook input** to continue and prints `{"status":"warning"}`.
- `fail` rejects ambiguous hook input with `unsupported_request`.
- Both modes reject a valid update whose exact receipt is missing or stale.
- A fully verified batch prints `{"status":"valid"}`.

This is stricter than a general advisory mode. Changing `config.json` alone does not wire or dynamically configure a hook; an executable wrapper must pass the intended `--mode`.

### Declarative asset versus executable hook

These are different files with different jobs:

| File | Created by `install`? | Executed by Git? |
| --- | --- | --- |
| `.docmanager/hooks/pre-push.json` | Yes | No. It is a declarative embedded asset. |
| `.git/hooks/pre-push` or the active hooks-path equivalent | No | Yes, when executable and selected by Git. |

### Safe optional manual wiring

Only wire the hook after local analysis and verification work reliably. The script below refuses to overwrite any existing hook, resolves the actual Git hooks directory, and uses `$HOME/.local/bin/docmanager` explicitly.

```bash
set -eu

REPOSITORY=/absolute/path/to/docmanager-demo
DOCMANAGER="$HOME/.local/bin/docmanager"

test -x "$DOCMANAGER"
git -C "$REPOSITORY" rev-parse --show-toplevel >/dev/null

git_dir="$(git -C "$REPOSITORY" rev-parse --absolute-git-dir)"
hook="$git_dir/hooks/pre-push"

if [ -e "$hook" ] || [ -L "$hook" ]; then
  printf '%s\n' "Refusing to overwrite existing hook: $hook" >&2
  exit 1
fi

mkdir -p "$(dirname "$hook")"
cat > "$hook" <<'EOF'
#!/bin/sh
set -eu
repo="$(git rev-parse --show-toplevel)"
exec "$HOME/.local/bin/docmanager" hook-verify --repo "$repo" --mode warn
EOF
chmod 700 "$hook"
```

The wrapper inherits Git's update lines on stdin and passes them unchanged to `hook-verify`. If you intentionally adopt fail mode, change `warn` to `fail` in the executable wrapper after team agreement. The product does not auto-synchronize the wrapper with `.docmanager/config.json`.

To test parsing without contacting a remote, pipe a syntactically real update line whose objects exist locally. This still requires a matching range receipt in the ledger:

```bash
REPOSITORY=/absolute/path/to/docmanager-demo
old="$(git -C "$REPOSITORY" rev-parse HEAD^)"
new="$(git -C "$REPOSITORY" rev-parse HEAD)"

printf '%s %s %s %s\n' \
  refs/heads/main "$new" refs/heads/main "$old" |
  docmanager hook-verify --repo "$REPOSITORY" --mode warn
```

Do not overwrite an existing team hook. Integrate the `exec ... hook-verify` invocation into the existing hook manager or wrapper according to that repository's conventions.

## MCP Onboarding

### Start the stdio server

```bash
docmanager mcp
```

The process speaks MCP on stdin/stdout and normally remains running until the client closes the transport. Do not expect a startup banner on stdout; protocol output must remain machine-readable.

### Exact MCP tools

| Tool | Input | Structured success output |
| --- | --- | --- |
| `document_change` | Absolute `repository` and explicit `scope`. | `{"report": <same report shape as CLI>}` |
| `verify_receipt` | Same repository/scope plus the complete receipt object. | `{"verified": true}` |

There are no MCP lifecycle or mutation tools. `install`, `doctor`, `uninstall`, and `hook-verify` remain CLI-only.

### OpenCode daily review guidance

`docmanager install --agent opencode` adds only an owned local MCP entry and the absolute `.docmanager/guidance/opencode.md` instruction reference to OpenCode; it preserves unrelated OpenCode configuration. The guidance is useful only for an explicit documentation-impact review:

1. Select exactly one `worktree`, `staged`, or two-dot `base..head` scope and make one `document_change` call.
2. Review the report yourself before requesting receipt verification.
3. Verify at most once while the selected scope and considered documentation bytes are unchanged; otherwise re-analyze and review again.

`docmanager opencode status` and `doctor` are read-only inspection commands for this OpenCode pair. This integration does not infer scope, edit documentation, modify user instructions or `AGENTS.md`, manage MCP lifecycle, enable hooks, configure other agents, or perform release work. To roll it back, remove only the exact owned MCP/instruction pair and retain `.docmanager` plus unrelated user configuration.

### `document_change` input

```json
{
  "repository": "/absolute/path/to/docmanager-demo",
  "scope": {
    "kind": "range",
    "range": "main..HEAD"
  }
}
```

For staged changes:

```json
{
  "repository": "/absolute/path/to/docmanager-demo",
  "scope": {
    "kind": "staged",
    "range": ""
  }
}
```

For worktree changes:

```json
{
  "repository": "/absolute/path/to/docmanager-demo",
  "scope": {
    "kind": "worktree",
    "range": ""
  }
}
```

### `verify_receipt` input

Pass the complete lowercase-field receipt returned inside `report.Receipt`:

```json
{
  "repository": "/absolute/path/to/docmanager-demo",
  "scope": {
    "kind": "staged",
    "range": ""
  },
  "receipt": {
    "digest": "sha256:<64 hexadecimal characters>",
    "scope": {
      "kind": "staged",
      "range": ""
    },
    "evidence": "sha256:<64 hexadecimal characters>",
    "changed_paths": ["cmd/server.go"],
    "documentation": [
      {
        "path": "README.md",
        "digest": "sha256:<64 hexadecimal characters>"
      }
    ],
    "outcome": "create",
    "versions": {
      "schema": "receipt/v1",
      "tool": "docmanager/1"
    }
  }
}
```

### Generic stdio client configuration

MCP client configuration formats differ. Map this generic shape to your client's documented stdio-server format:

```json
{
  "mcpServers": {
    "repository-documentation-manager": {
      "command": "/absolute/path/to/docmanager",
      "args": ["mcp"]
    }
  }
}
```

The product does not install this configuration. Restart or reload the client if that client requires it, then list tools and confirm that only `document_change` and `verify_receipt` appear.

### MCP security boundary

- The repository argument is explicit and must be its exact absolute Git root.
- Tool calls invoke bounded, shell-free Git subprocesses with five-second timeouts and sanitized Git configuration/environment inputs.
- Git output is bounded to 1 MiB for evidence resolution; hook-specific Git output is bounded to 4096 bytes.
- External diff and text-conversion behavior is disabled for evidence resolution.
- Worktree documentation symlinks resolving outside the repository are rejected.
- MCP cannot install assets, uninstall state, write documentation, stage, commit, push, or configure hooks.
- `document_change` does write local ledger metadata. `verify_receipt` requires and reads an existing ledger.

MCP does not make the heuristic semantic or AI-powered. Any client or agent remains responsible for interpreting the report and making separately authorized edits.

## Doctor, Uninstall, and Recovery

### Doctor

```bash
docmanager doctor --target /absolute/path/to/repository
```

Expected success:

```json
{"git":true}
```

With a target, `doctor` validates the exact Git root. If `.docmanager` exists, it also validates ownership and exact embedded asset contents. It does not validate receipt semantics or run an analysis. Without `--target`, `doctor` only checks that Git is available:

```bash
docmanager doctor
```

### Safe uninstall

```bash
docmanager uninstall --target /absolute/path/to/repository
```

Successful uninstall is silent. It removes the entire `.docmanager` directory only when the exact ownership marker is present. That includes assets and `ledger.db`. It does not remove:

- An executable Git hook that you wired manually.
- MCP client configuration.
- The `docmanager` executable.
- Documentation, commits, refs, remotes, or Git configuration.

Remove a manually wired hook only after verifying that it is the wrapper you created and not a shared or pre-existing hook:

```bash
REPOSITORY=/absolute/path/to/repository
git_dir="$(git -C "$REPOSITORY" rev-parse --absolute-git-dir)"
hook="$git_dir/hooks/pre-push"

printf '%s\n' "Inspect before removing: $hook"
```

Inspection and removal are intentionally not automated here. Ownership of `.git/hooks/pre-push` is outside the product lifecycle.

### Typed errors

CLI failures print one typed error to stderr and exit non-zero. MCP failures return an error result containing `{"error":"<type>"}`.

| Error | Meaning | Recovery |
| --- | --- | --- |
| `invalid_scope` | Missing/unsupported scope, missing range value, range supplied to staged/worktree, or malformed public scope flags. | Supply exactly one of `range`, `staged`, or `worktree`; use `--range A..B` only with `range`. |
| `not_repository` | The requested path is not inside a Git repository. | Initialize Git or select an existing repository. |
| `outside_repository` | `--repo` points inside a repository rather than at its exact top level. | Use `git -C <path> rev-parse --show-toplevel` and pass that result. |
| `git_unavailable` | Git cannot be found on `PATH`. | Install Git or correct the process environment. |
| `ledger_failure` | `.docmanager`/ledger is inaccessible, unsafe, corrupt, over capacity for hook enumeration, or SQLite failed. | Run `doctor`; if state is owned and disposable, uninstall, reinstall, and re-run analysis. Preserve evidence externally if needed first. |
| `receipt_mismatch` | Receipt is malformed, tampered, stale, absent from ledger, or bound to different content/scope/outcome. | Re-run analysis for the exact current scope and verify the newly returned complete receipt. |
| `invalid_target` | Lifecycle target is missing, not the exact root, traverses `..`, is symlinked/unsafe, or contains unowned/altered managed state. | Use the exact real repository root. Do not bypass ownership or symlink checks; inspect conflicting state manually. |
| `unsupported_request` | Unknown command, unsupported mode/request, ambiguous fail-mode hook input, or an internal error not exposed as another public type. | Validate command spelling and flags against this guide; inspect hook stdin and mode. |

Other internal conditions such as empty scopes, invalid revisions, bounded output, or content-read failure are currently collapsed by the CLI/MCP public classifier to `unsupported_request`.

### Common situations

| Situation | Why it happens | Safe response |
| --- | --- | --- |
| Empty staged/worktree/range result | The selected scope has no changed paths. | Select the correct scope; do not invent an unscoped analysis. |
| Stale staged receipt | Staged paths or bytes changed after analysis. | Re-run `document-change --scope staged`; use the new receipt. |
| Dirty worktree with range receipt | Range reads committed objects, not dirty files. | This is expected; separately analyze worktree changes if they matter. |
| Untracked file absent from worktree report | Worktree scope enumerates tracked index entries only. | Add/stage the file and use staged scope, or commit it and use range scope. |
| `install` fails after analysis-first use | Analysis created unowned `.docmanager` state without `.owned`. | Inspect and preserve anything needed, then remove that local cache manually only if you own it; run `install` before rebuilding receipts. |
| Symlinked target or `.docmanager` rejected | Lifecycle and ledger code refuse symlink traversal at ownership boundaries. | Use a real, exact repository root and regular managed files. Do not replace safety checks with symlinks. |
| Hook fails after remote advanced | Stored range uses different endpoint IDs. | Fetch, reconcile the branch, analyze the new exact push range, and retry. |
| More than 64 stored receipts | Hook reads a bounded receipt catalog and treats overflow as `ledger_failure`. | Rebuild owned local state and regenerate only currently needed receipts. |

## Team Adoption Without Surprise Blocks

Everything in this section is a **recommendation**, not implemented automation.

### Stage 1: local-only trial

- Build from source and install assets in one volunteer repository.
- Do not wire a hook yet.
- Run staged analysis at feature close and review false positives/negatives.
- Confirm that `.docmanager` handling matches the team's ignore/versioning policy.
- Generate a fresh receipt only after the intended scope is stable.

### Stage 2: CI visibility

CI can build `docmanager` and run explicit `range` analysis, but the repository currently provides no ready-made CI integration or policy script. A CI job must choose exact commit endpoints available in its clone and decide how to preserve/use the local ledger during that job.

Recommended policy:

- Start by publishing the report as evidence, not failing based on `Outcome` alone.
- Do not treat `create` or `no-impact` as proof of missing or unnecessary documentation.
- Keep receipt generation and verification in the same workspace unless you intentionally transfer the full ledger.
- Fetch enough history for both range endpoints; shallow clones may not contain them.

### Stage 3: warn-mode hook trial

- Wire the executable hook manually only for volunteers.
- Prepare exact range receipts after the branch is ready and the remote endpoint is fetched.
- Remember that `warn` still blocks missing or stale receipts; it only tolerates ambiguous input.
- Exclude first-push branches until the public initial-receipt gap is resolved.

### Stage 4: deliberate fail mode

- Move to `fail` only after every supported push path supplies valid update records and the team understands receipt timing.
- Document recovery before enforcement.
- Coordinate with existing hook managers instead of overwriting hooks.
- Roll out consistently; a local ledger is not shared automatically with teammates.

### Avoid blocking teammates incorrectly

- Never assume your local `.docmanager/ledger.db` exists on another clone.
- Do not commit a policy requiring receipts before agreeing how each developer or CI job creates its own ledger.
- Do not enable first-push enforcement while public clients cannot create `initial` receipts.
- Re-analyze after rebases, amended commits, staging changes, documentation edits, or remote endpoint changes.
- Treat analysis as evidence for review, not an automatic correctness gate.

## Architecture and Stack for New Contributors

The repository follows a clean-ish dependency direction: domain rules at the center, application use cases around them, adapters at the edge, and one CLI composition root.

```text
cmd/docmanager/main.go
        |
        v
internal/app/                 use cases and ports
        |
        v
internal/domain/              scopes, evidence, heuristics, receipts, typed errors
        ^
        |
internal/adapters/git/        bounded Git evidence resolver
internal/adapters/sqlite/     pure-Go receipt ledger
internal/adapters/mcp/        official Go MCP SDK stdio server

assets/                       embedded guidance and hook declaration
```

| Area | Responsibility |
| --- | --- |
| `cmd/docmanager/main.go` | Parses subcommands/flags, wires adapters and use cases, emits JSON, classifies public errors. |
| `internal/domain` | Validates scopes, defines evidence/report/receipt types, runs deterministic path heuristics, calculates receipt digests. |
| `internal/app/document_change.go` | Orchestrates resolve, analyze, receipt creation, and ledger save. |
| `internal/app/verify_receipt.go` | Reconstructs expected evidence/receipt and compares it with supplied and stored data. |
| `internal/app/lifecycle.go` | Safely installs, validates, and removes only owned `.docmanager` state. |
| `internal/app/hook.go` | Parses real pre-push update lines and verifies exact stored receipts. |
| `internal/adapters/git` | Executes Git directly without a shell, disables configurable diff filters, bounds output, and hashes canonical evidence. |
| `internal/adapters/sqlite` | Uses `modernc.org/sqlite` to store complete receipt JSON in a rebuildable local database. |
| `internal/adapters/mcp` | Uses official `github.com/modelcontextprotocol/go-sdk` v1.7.0 and exposes two stdio tools. |
| `assets` | Uses Go `embed` for local guidance, config, and declarative hook data. |

### Test and CI shape

- Unit tests cover domain analysis and receipt behavior.
- Filesystem/Git integration tests use temporary repositories.
- MCP has a stdio integration test and skips it under `go test -short`.
- CLI tests prove lifecycle, analysis, verification, typed errors, and non-mutation.
- Hook tests cover normal ranges, explicit differing refs, first pushes, deletions, ambiguity modes, stale/tampered receipts, and dirty worktrees.
- SQLite tests cover persistence and receipt matching.
- CI runs all tests, selected race tests, vet, formatting, native package smokes, and CGo-free cross-builds.

Contributor commands:

```bash
go test ./... -count=1
go test -race ./internal/app ./internal/adapters/mcp ./cmd/docmanager -count=1
go vet ./...
test -z "$(gofmt -l cmd internal spike)"
```

## Complete Disposable Tutorial

This script is intended for Bash on a Unix-like system. It creates one directory under the system temporary directory, builds the current source into that directory, rehearses `install`, staged analysis, receipt extraction, verification, range/worktree scopes, non-mutation checks, `doctor`, and `uninstall`, then removes the sandbox. It does not execute destructive commands outside the created sandbox.

Set `SOURCE` to your source checkout before running it.

```bash
#!/usr/bin/env bash
set -euo pipefail

SOURCE=${SOURCE:-/absolute/path/to/repository-documentation-manager}
test -f "$SOURCE/go.mod"

sandbox="$(mktemp -d "${TMPDIR:-/tmp}/docmanager-onboarding.XXXXXX")"
cleanup() {
  rm -rf -- "$sandbox"
}
trap cleanup EXIT

repo="$sandbox/demo"
bin="$sandbox/bin/docmanager"
mkdir -p "$repo/cmd" "$repo/internal" "$(dirname "$bin")"

CGO_ENABLED=0 go -C "$SOURCE" build \
  -trimpath \
  -buildvcs=false \
  -o "$bin" \
  ./cmd/docmanager

git -C "$repo" init -b main
git -C "$repo" config user.name "Onboarding Learner"
git -C "$repo" config user.email "learner@example.invalid"

cat > "$repo/cmd/server.go" <<'EOF'
package main

import "fmt"

func main() {
	fmt.Println("API version 1")
}
EOF

cat > "$repo/internal/service.go" <<'EOF'
package internal

const ServiceName = "demo"
EOF

cat > "$repo/README.md" <<'EOF'
# Demo API

The server prints API version 1.
EOF

git -C "$repo" add README.md cmd/server.go internal/service.go
git -C "$repo" commit -m "Create baseline"

"$bin" install --target "$repo"
"$bin" doctor --target "$repo" | jq -e '.git == true' >/dev/null
test -f "$repo/.docmanager/.owned"
test -f "$repo/.docmanager/guidance/AGENTS.md"
test -f "$repo/.docmanager/hooks/pre-push.json"
test ! -e "$repo/.docmanager/ledger.db"

cat > "$repo/cmd/server.go" <<'EOF'
package main

import "fmt"

func main() {
	fmt.Println("API version 2")
}
EOF
git -C "$repo" add cmd/server.go

status_before="$(git -C "$repo" status --porcelain=v1)"
index_before="$(git -C "$repo" write-tree)"
readme_before="$(git -C "$repo" hash-object --no-filters README.md)"

report="$($bin document-change --repo "$repo" --scope staged)"
printf '%s\n' "$report" | jq -e \
  '.Outcome == "create" and .Candidates == ["README.md"] and (.Receipt.digest | startswith("sha256:"))' \
  >/dev/null
receipt="$(printf '%s' "$report" | jq -c '.Receipt')"
"$bin" verify --repo "$repo" --scope staged --receipt "$receipt" |
  jq -e '.verified == true' >/dev/null

test -f "$repo/.docmanager/ledger.db"
test "$(git -C "$repo" status --porcelain=v1)" = "$status_before"
test "$(git -C "$repo" write-tree)" = "$index_before"
test "$(git -C "$repo" hash-object --no-filters README.md)" = "$readme_before"

cat > "$repo/README.md" <<'EOF'
# Demo API

The server prints API version 2.
EOF
git -C "$repo" add README.md

updated_report="$($bin document-change --repo "$repo" --scope staged)"
printf '%s\n' "$updated_report" | jq -e \
  '.Outcome == "update" and .Candidates == ["README.md"]' \
  >/dev/null
updated_receipt="$(printf '%s' "$updated_report" | jq -c '.Receipt')"
"$bin" verify --repo "$repo" --scope staged --receipt "$updated_receipt" |
  jq -e '.verified == true' >/dev/null

git -C "$repo" commit -m "Change API behavior and documentation"

range_report="$($bin document-change \
  --repo "$repo" \
  --scope range \
  --range 'HEAD^..HEAD')"
printf '%s\n' "$range_report" | jq -e '.Outcome == "update"' >/dev/null
range_receipt="$(printf '%s' "$range_report" | jq -c '.Receipt')"
"$bin" verify \
  --repo "$repo" \
  --scope range \
  --range 'HEAD^..HEAD' \
  --receipt "$range_receipt" |
  jq -e '.verified == true' >/dev/null

cat > "$repo/internal/service.go" <<'EOF'
package internal

const ServiceName = "renamed-demo"
EOF

worktree_report="$($bin document-change --repo "$repo" --scope worktree)"
printf '%s\n' "$worktree_report" | jq -e \
  '.Outcome == "no-impact" and .Evidence.ChangedPaths == ["internal/service.go"]' \
  >/dev/null
worktree_receipt="$(printf '%s' "$worktree_report" | jq -c '.Receipt')"
"$bin" verify --repo "$repo" --scope worktree --receipt "$worktree_receipt" |
  jq -e '.verified == true' >/dev/null

"$bin" uninstall --target "$repo"
test ! -e "$repo/.docmanager"
test -f "$repo/README.md"
test -d "$repo/.git"

printf '%s\n' "Disposable onboarding tutorial passed: $repo"
```

Expected final line before automatic cleanup:

```text
Disposable onboarding tutorial passed: /tmp/docmanager-onboarding.<random>/demo
```

The script intentionally snapshots after staging the API change and after installing local assets. It proves that analysis and verification leave Git status, the index tree, and tracked README bytes unchanged. The ledger appears as the one expected metadata mutation.

## Checklists

### Installation complete

- [ ] Built from the current source checkout with Go 1.26 or newer.
- [ ] `docmanager doctor --target <exact-root>` prints `{"git":true}`.
- [ ] `.docmanager/.owned` exists.
- [ ] `.docmanager/guidance/AGENTS.md` exists at that exact path.
- [ ] `.docmanager/hooks/pre-push.json` exists and is understood as declarative only.
- [ ] No executable Git hook was assumed or silently overwritten.

### First successful analysis

- [ ] Repository has a baseline commit.
- [ ] Selected scope is explicit and non-empty.
- [ ] Report outcome, candidates, rationale, confidence, and changed paths were reviewed.
- [ ] Human judgment was applied; the heuristic was not treated as semantic truth.
- [ ] `.docmanager/ledger.db` now exists.
- [ ] Tracked documentation and Git state were not changed by the command.

### Receipt verification

- [ ] Extracted the complete object with `jq -c '.Receipt'`.
- [ ] Used the same scope kind and range value as the analysis.
- [ ] Ran verification before changing selected content.
- [ ] Received `{"verified":true}`.
- [ ] Re-analyzed after any rebase, amend, staging change, or documentation edit.

### MCP connected

- [ ] Client launches the absolute executable path with argument `mcp` over stdio.
- [ ] Tool list contains exactly `document_change` and `verify_receipt`.
- [ ] Requests use an absolute repository root and explicit scope object.
- [ ] No lifecycle or mutation-capable MCP tools are expected.
- [ ] Team understands that analysis writes only ledger metadata.

### Optional hook enforcement

- [ ] Local analysis and verification are reliable before wiring.
- [ ] Existing hooks were inspected and not overwritten.
- [ ] Executable wrapper points to a known `docmanager` binary.
- [ ] Team understands `warn` versus `fail` ambiguity behavior.
- [ ] Exact normal-push range receipts are created after fetching the remote endpoint.
- [ ] First-push enforcement is excluded because public clients cannot create `initial` receipts.
- [ ] Recovery steps are documented before enforcing teammates' pushes.

### Safe uninstall

- [ ] Confirmed `.docmanager/.owned` belongs to `docmanager`.
- [ ] Preserved any receipt evidence that must be retained outside the rebuildable local cache.
- [ ] Ran `docmanager uninstall --target <exact-root>`.
- [ ] Confirmed tracked documentation and Git history remain intact.
- [ ] Separately inspected any manually wired Git hook and MCP client configuration.

## Current Limitations and Honest Boundaries

- No documentation generation or editing.
- No staging, commits, pushes, ref changes, or remote operations.
- No unscoped analysis; empty scopes fail.
- No semantic code or prose understanding; outcomes are path heuristics.
- No claim that `create` means the candidate file is absent.
- No published installer or package-registry assumption in this repository.
- No automatic executable Git hook installation or `core.hooksPath` configuration.
- No public CLI/MCP `initial` analysis even though first-push hook verification requires an initial receipt.
- No untracked-file coverage in `worktree` scope.
- No three-dot range syntax; use exactly `A..B` with commit endpoints.
- No automatic ledger sharing; receipts are local to `.docmanager/ledger.db`.
- Hook receipt enumeration is limited to 64 stored receipts and 32 update lines.
- Git evidence subprocesses have five-second timeouts and bounded output; very large or slow repositories can fail.
- Lifecycle state is intentionally strict: symlinked, altered, or unowned targets are rejected rather than repaired.
- Tested platform coverage is Linux/amd64, macOS/arm64, and Windows/amd64; Bash examples are Unix-oriented.
- MCP is stdio-only in the current implementation.
- Review mode, SDD state, and repository governance workflows are unrelated to ordinary product use. `docmanager` does not read or change them.

## Glossary

| Term | Definition |
| --- | --- |
| Candidate | Documentation path suggested for human inspection. |
| Canonical evidence | Stable representation of selected Git objects or tracked worktree bytes used to calculate an identity hash. |
| Content-bound | Tied to exact scope endpoints, paths, and relevant bytes; changing them invalidates the receipt. |
| Evidence identity | SHA-256 digest of the canonical selected Git evidence. |
| Explicit scope | One caller-selected `range`, `staged`, or `worktree`; never inferred by the tool. |
| Initial scope | Internal hook-only representation of all paths reachable from a first-pushed commit. Not exposed by public CLI/MCP analysis. |
| Ledger | Local pure-Go SQLite receipt cache at `.docmanager/ledger.db`. |
| Lifecycle | Owned-asset operations `install`, `doctor`, and `uninstall`. |
| MCP | Model Context Protocol transport allowing a compatible client to call the two stdio tools. It does not imply AI behavior. |
| Outcome | Path-heuristic result: `update`, `create`, or `no-impact`. |
| Receipt | Full JSON proof binding evidence, scope, paths, documentation digests, outcome, and versions. |
| Verification | Rebuilding expected evidence and checking it against the complete supplied and stored receipt. |

## What to Do Next

1. Run the complete disposable tutorial against your current source checkout.
2. Install assets in one real repository and use staged analysis without a hook.
3. Review several reports with your team to understand path-heuristic limits.
4. Decide how your team treats local `.docmanager` state.
5. Add MCP only if a compatible client needs explicit analyze/verify calls.
6. Consider manual pre-push wiring last, after normal range receipts and first-push limitations are understood.

The durable workflow is simple: **select scope, analyze, apply human judgment, make any documentation change yourself, re-analyze if content changes, and verify the exact receipt.**
