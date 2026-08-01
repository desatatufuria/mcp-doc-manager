# Repository Documentation Manager

`docmanager` is a repository-local, read-only documentation-impact analyzer. At feature close, you select one Git scope; it returns `update`, `create`, or `no-impact` with evidence and a content-bound receipt. It never writes, stages, commits, or proposes documentation changes.

## Quick path

```sh
go build -o docmanager ./cmd/docmanager
./docmanager document-change --repo /absolute/path/to/repository --scope staged
```

The command requires exactly one scope:

| Scope | Command |
| --- | --- |
| Revision range | `document-change --repo /repo --scope range --range main..HEAD` |
| Staged changes | `document-change --repo /repo --scope staged` |
| Worktree changes | `document-change --repo /repo --scope worktree` |

Build from source with Go 1.26. Native CI validates Linux/amd64, macOS/arm64, and Windows/amd64; release artifacts are built with `CGO_ENABLED=0` for Linux/amd64, macOS/arm64, and Windows/amd64.

## CLI and receipts

`document-change` emits JSON containing the outcome, candidate paths, rationale, confidence, canonical evidence, and receipt. Pass the complete lowercase-field receipt object from that response to `verify`:

```sh
report="$(./docmanager document-change --repo /repo --scope staged)"
receipt="$(printf '%s' "$report" | jq -c '.Receipt')"
./docmanager verify --repo /repo --scope staged --receipt "$receipt"
```

Verification accepts only the exact, successful analysis whose selected Git content and documentation bytes still match. The SQLite ledger in `.docmanager/` is rebuildable cache state; Git remains authoritative.

| Command | Purpose |
| --- | --- |
| `docmanager document-change ...` | Explicitly analyze one selected scope. |
| `docmanager verify ...` | Recheck a content-bound receipt without an LLM or repository mutation. |
| `docmanager mcp` | Start the stdio MCP server. |
| `docmanager install --target /repo` | Install owned local guidance and optional hook configuration. |
| `docmanager doctor --target /repo` | Validate Git and owned local assets without changing them. |
| `docmanager uninstall --target /repo` | Remove only owned `.docmanager/` state. |

## Read-only MCP

Run `docmanager mcp` over stdio and configure the client to launch that executable. The MCP surface is deliberately small and read-only:

| Tool | Inputs | Result |
| --- | --- | --- |
| `document_change` | Absolute repository root and one explicit `range`, `staged`, or `worktree` scope | Analysis report and receipt. |
| `verify_receipt` | Same repository/scope plus a receipt | `verified: true` only when it still matches. |

MCP does not expose install, uninstall, patch, commit, or documentation-write tools. Lifecycle commands remain CLI-only.

## Lifecycle and pre-push receipts

`install` creates an owned `.docmanager/` directory containing embedded `AGENTS.md`, skill guidance, and declarative pre-push configuration. It refuses symlinked, unowned, or altered state. `doctor` is read-only; `uninstall` removes only state carrying the ownership marker.

The installed pre-push configuration invokes `docmanager hook-verify`. It reads normal Git pre-push update records, resolves each non-deletion update, and requires a stored receipt that exactly verifies current content. The embedded default is `warn`: ambiguous hook input warns, while a missing, stale, tampered, or mismatched receipt fails verification. Use `--mode fail` only when your team intends to enforce ambiguous-input failure as well.

## Disposable acceptance walkthrough

This Bash walkthrough creates and removes a temporary repository. It requires `git`, `go`, and `jq`; it does not touch the repository that contains `docmanager`.

```sh
set -eu
root="$(mktemp -d)"
trap 'rm -rf "$root"' EXIT
git -C "$root" init
git -C "$root" config user.email demo@example.invalid
git -C "$root" config user.name demo
printf 'before\n' > "$root/README.md"
git -C "$root" add README.md && git -C "$root" commit -m initial
printf 'after\n' > "$root/README.md"
git -C "$root" add README.md
go build -o "$root/docmanager" ./cmd/docmanager
"$root/docmanager" install --target "$root"
report="$("$root/docmanager" document-change --repo "$root" --scope staged)"
printf '%s\n' "$report" | jq -e '.Outcome == "update" and (.Receipt.digest? | type == "string" and length > 0)'
receipt="$(printf '%s' "$report" | jq -c '.Receipt')"
"$root/docmanager" verify --repo "$root" --scope staged --receipt "$receipt"
"$root/docmanager" doctor --target "$root"
"$root/docmanager" uninstall --target "$root"
test ! -e "$root/.docmanager"
```

## Troubleshooting and uninstall

| Symptom | Check |
| --- | --- |
| `invalid_scope` | Supply exactly one supported scope; `range` also needs `--range`. |
| `outside_repository` or `not_repository` | Use the exact Git root as `--repo` or `--target`. |
| `receipt_mismatch` | Re-run the explicit analysis for the current selected content, then verify that new receipt. |
| `ledger_failure` | Run `doctor`; remove only owned state with `uninstall`, then re-run analysis to rebuild the ledger. |
| Hook warns or fails | Ensure the pushed range has a matching stored receipt; use the hook's configured mode intentionally. |

To remove local integration, run `docmanager uninstall --target /absolute/repository/root`. This removes owned `.docmanager/` cache, guidance, and hook configuration only. It does not modify repository documentation, Git history, or external MCP client configuration.

## Release verification record

The Phase 5 release work unit was verified on 2026-08-01 with command-scoped Go caches under `/tmp/opencode`:

- `go test ./... -count=1` passed: six tested packages passed; `assets` has no tests.
- `go test -race ./internal/app ./internal/adapters/mcp ./cmd/docmanager -count=1` passed: all three packages passed.
- `go vet ./...`, `gofmt -l cmd internal spike`, and `git diff --check` passed with no output.
- A native CGo-free build and the Linux/amd64, macOS/arm64, and Windows/amd64 CGo-free cross-builds produced nonempty executables.
- The executable acceptance harness installed and validated embedded assets, analyzed and verified a range receipt, accepted a matching fail-mode pre-push update, and uninstalled its owned state.
- Before/after disposable-repository snapshots matched: Git status SHA-256 `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`, index tree `d8f3f0f0156415c2455efe27a139445641b602fd`, and `README.md` SHA-256 `7b9a72466d3960eb2aacccfc848939453490db0678bd4725def3f789b891c919`.

The authoritative SDD apply-progress artifact records the exact commands, outcomes, and cleanup boundary.
