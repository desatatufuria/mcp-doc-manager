# Tasks: Repository Documentation Manager

## Review Workload Forecast

| Field | Value |
|---|---|
| Estimated changed lines | 2,400–3,400 |
| 800-line budget risk | High |
| Chained PRs recommended | Yes |
| Suggested split | Gate → core → ledger/CLI → MCP/assets → release |
| Delivery strategy | auto-forecast |
| Chain strategy | stacked-to-main |

Decision needed before apply: Yes
Chained PRs recommended: Yes
Chain strategy: stacked-to-main
800-line budget risk: High

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|---|---|---|---|---|---|
| 1 | Go compatibility spike | Gate | `go test ./spike/...` | matrix build + stdio ping | `spike/`, workflow |
| 2 | Domain and Git resolver | PR 1 | `go test ./internal/domain ./internal/adapters/git` | temp Git repo | domain + Git |
| 3 | Receipts, ledger, CLI | PR 2 | `go test ./internal/app ./internal/adapters/{sqlite,cli}` | CLI temp repo | app, ledger, CLI |
| 4 | MCP, assets, hook | PR 3 | `go test ./internal/adapters/mcp ./internal/app` | stdio + hook fixture | MCP + assets |
| 5 | Release packaging/docs | PR 4 | `go test ./...` | package smoke | CI + docs |

## Phase 1: Compatibility Gate and Foundation

- [x] 1.1 Create `spike/go-mcp-sqlite/{go.mod,main.go,main_test.go}`; prove MCP stdio, pure-Go SQLite, binaries, and package smoke.
- [x] 1.2 Add `.github/workflows/spike.yml` macOS/Linux/Windows tests and pass/fail criteria in `spike/go-mcp-sqlite/README.md`.
- [x] 1.3 If the spike fails, record Rust reassessment and stop before `cmd/` or `internal/` core creation. (Not applicable: the Go spike passed.)
- [x] 1.4 On success, create `go.mod`, `cmd/docmanager/main.go`, `.gitignore`, and test conventions.

## Phase 2: Git Evidence and Outcomes

- [x] 2.1 RED: test `requirements.txt`, `CMakeLists.txt`, executable Markdown/MDX, and `README.sh` in `internal/domain/analysis_test.go`: classify, never execute.
- [x] 2.2 Implement `internal/domain/{scope,evidence,analysis}.go`: three outcomes, candidates, rationale, confidence, and no patch proposal, write, or apply API.
- [x] 2.3 RED: temp-repo test `git -C`, relative/absolute roots, staged, `commit -a`, and empty index; reject outside/non-repo and type empty scope.
- [x] 2.4 Implement `internal/adapters/git/` shell-free bounded one-scope resolver with root validation, NUL-safe evidence, and typed failures.
- [x] 2.5 Test `internal/app/document_change.go` update/create/no-impact, ambiguous scope, Git unavailable, unsupported request, and non-mutation.
- [x] 2.6 Implement the use case: preserve content; failures return neither outcome nor receipt.

Phase 2 ordinals 5–6 exposed and partially corrected Git authority gaps. Ordinals 7–10 passed scope isolation plus exact initial-root, identity-, and post-root-discovery overflow classification, so tasks 2.1–2.6 remain checked. Phase 3+ remains out of scope.

## Phase 3: Deterministic Receipts, Ledger, and CLI

- [x] 3.1 RED: test canonical receipts: equal semantics has one digest; changed scope/content/version rejects.
- [x] 3.2 Implement `internal/domain/receipt.go` SHA-256 binding scope, evidence, docs blobs, outcome, and versions.
- [x] 3.3 Test `internal/adapters/sqlite/` with `t.TempDir()` save/rebuild/verify and typed ledger failure.
- [x] 3.4 Implement removable `.docmanager/` SQLite cache/ledger; Git remains authoritative.
- [x] 3.5 Test CLI document-change/verify, invalid scope, and non-mutation contracts.
- [x] 3.6 Implement CLI plus `internal/app/{verify_receipt,install,doctor,uninstall}.go` contained lifecycle and typed errors.

Phase 3 safety/proof correction terminal: ordinal 13 passed at `363/500` native lines. Ordinal 14 then passed at `51/80` native lines, rejecting traversal-shaped lifecycle targets before normalization. It hardens ownership marker lifecycle, prevalidates CLI repository roots before SQLite creation, types ledger operational failures, proves canonical receipts with an independent oracle, and expands CLI tamper/invalid-scope proof. Phase 4+ remains out of scope.

## Phase 4: MCP, Assets, and Optional Hook

- [x] 4.1 Test `document_change` CLI/MCP field, receipt, and failure parity with `internal/adapters/mcp/mcp_test.go` stdio fixtures.
- [x] 4.2 Implement read-only MCP tools requiring explicit invocation.
- [x] 4.3 RED: hook fixtures cover tracking, first push, explicit refspec; ambiguity warns or fails closed by mode.
- [x] 4.4 Add `assets/{AGENTS.md,skills/docmanager/SKILL.md,.githooks/pre-push}`; argument arrays only; no shell/LLM/mutation.

Ordinal 17 passed the bounded task 4.3 correction: Git-equivalent local/remote ref validation and complete root/multi-commit first-push receipt evidence are covered. Phase 5 remains out of scope.

## Phase 5: Release Verification and Documentation

- [x] 5.1 Create `.github/workflows/ci.yml` for format, tests, race checks, cross-platform builds, and package smoke.
- [x] 5.2 Add `README.md` quick path, scopes, receipt semantics, lifecycle rollback, and non-goals.
- [x] 5.3 Run `go test ./...`, `go vet ./...`, matrix/package smoke, and mutation snapshots; record results in `README.md`.

Ordinal 19 corrected the package-smoke installed asset assertion and the README receipt example/assertion. Tasks 5.1–5.3 remain checked: CI now proves the actual embedded guidance path, the documented complete lowercase-field receipt verifies, malformed digest-only input fails, and the full quality/runtime/non-mutation evidence passed.
