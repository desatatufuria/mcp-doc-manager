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

- [ ] 1.1 Create `spike/go-mcp-sqlite/{go.mod,main.go,main_test.go}`; prove MCP stdio, pure-Go SQLite, binaries, and package smoke.
- [ ] 1.2 Add `.github/workflows/spike.yml` macOS/Linux/Windows tests and pass/fail criteria in `spike/go-mcp-sqlite/README.md`.
- [ ] 1.3 If the spike fails, record Rust reassessment and stop before `cmd/` or `internal/` core creation.
- [ ] 1.4 On success, create `go.mod`, `cmd/docmanager/main.go`, `.gitignore`, and test conventions.

## Phase 2: Git Evidence and Outcomes

- [ ] 2.1 RED: test `requirements.txt`, `CMakeLists.txt`, executable Markdown/MDX, and `README.sh` in `internal/domain/analysis_test.go`: classify, never execute.
- [ ] 2.2 Implement `internal/domain/{scope,evidence,analysis,proposal}.go`: three outcomes, candidates, rationale, confidence, no write/apply API.
- [ ] 2.3 RED: temp-repo test `git -C`, relative/absolute roots, staged, `commit -a`, and empty index; reject outside/non-repo and type empty scope.
- [ ] 2.4 Implement `internal/adapters/git/` shell-free bounded one-scope resolver with root validation, NUL-safe evidence, and typed failures.
- [ ] 2.5 Test `internal/app/document_change.go` update/create/no-impact, ambiguous scope, Git unavailable, unsupported request, and non-mutation.
- [ ] 2.6 Implement the use case: preserve content; failures return neither outcome nor receipt.

## Phase 3: Deterministic Receipts, Ledger, and CLI

- [ ] 3.1 RED: test canonical receipts: equal semantics has one digest; changed scope/content/version rejects.
- [ ] 3.2 Implement `internal/domain/receipt.go` SHA-256 binding scope, evidence, docs blobs, outcome, and versions.
- [ ] 3.3 Test `internal/adapters/sqlite/` with `t.TempDir()` save/rebuild/verify and typed ledger failure.
- [ ] 3.4 Implement removable `.docmanager/` SQLite cache/ledger; Git remains authoritative.
- [ ] 3.5 Test CLI document-change/verify, invalid scope, and non-mutation contracts.
- [ ] 3.6 Implement CLI plus `internal/app/{verify_receipt,install,doctor,uninstall}.go` contained lifecycle and typed errors.

## Phase 4: MCP, Assets, and Optional Hook

- [ ] 4.1 Test `document_change` CLI/MCP field, receipt, and failure parity with `internal/adapters/mcp/mcp_test.go` stdio fixtures.
- [ ] 4.2 Implement read-only MCP tools requiring explicit invocation.
- [ ] 4.3 RED: hook fixtures cover tracking, first push, explicit refspec; ambiguity warns or fails closed by mode.
- [ ] 4.4 Add `assets/{AGENTS.md,skills/docmanager/SKILL.md,.githooks/pre-push}`; argument arrays only; no shell/LLM/mutation.

## Phase 5: Release Verification and Documentation

- [ ] 5.1 Create `.github/workflows/ci.yml` for format, tests, race checks, cross-platform builds, and package smoke.
- [ ] 5.2 Add `README.md` quick path, scopes, receipt semantics, lifecycle rollback, and non-goals.
- [ ] 5.3 Run `go test ./...`, `go vet ./...`, matrix/package smoke, and mutation snapshots; record results in `README.md`.
