# Apply Progress: Repository Documentation Manager

## Work Unit 1: Go Compatibility Spike

**Status**: Passed.

### Dependency Selection (Verified)

The compatibility module pins the selected dependencies and the focused tests and static builds passed. The Go path is accepted for the next work unit.

| Dependency | Candidate version | Rationale | Official source checked |
|---|---:|---|---|
| `github.com/modelcontextprotocol/go-sdk` | `v1.7.0` | Official MCP Go SDK; `mcp.StdioTransport`, server, client, and `CommandTransport` prove a real stdio request/response. | Pinned in `spike/go-mcp-sqlite/go.mod`; verified 2026-07-29 |
| `modernc.org/sqlite` | `v1.55.0` | CGo-free SQLite driver used through `database/sql` with driver name `sqlite`. | Pinned in `spike/go-mcp-sqlite/go.mod`; verified with `CGO_ENABLED=0` builds 2026-07-29 |

### Previous Environment Blocker Resolved

The prior attempt could not invoke Go. The active attempt verified the available toolchain:

```text
$ go version
go version go1.26.5 linux/amd64
```

The default `GOPATH=/go` could not write the checksum database. The harness used the writable, command-scoped `GOPATH=/tmp/opencode/gopath`, `GOMODCACHE=/tmp/opencode/gomodcache`, and `GOCACHE=/tmp/opencode/gocache`. This was an environment permission issue, not a Go MCP or SQLite technology failure.

### Task Status

- [x] 1.1 Create and prove `spike/go-mcp-sqlite`.
- [x] 1.2 Add the cross-platform spike workflow and documented criteria.
- [x] 1.3 Rust reassessment condition evaluated: not applicable because the spike passed.
- [x] 1.4 Create the root module and entrypoint after the successful spike.

### Rust Reassessment

Not recorded. The Go MCP/SQLite combination passed the required local compatibility evidence; the temporary checksum-cache permission issue was environmental and did not justify a technology reassessment.

## Work Unit Evidence

| Evidence | Required value |
|---|---|
| Focused test command and exact result | `GOPATH=/tmp/opencode/gopath GOMODCACHE=/tmp/opencode/gomodcache GOCACHE=/tmp/opencode/gocache go test ./... -count=1` from `spike/go-mcp-sqlite` — exit 0; 2 tests passed. Root convention check: `go test ./... -count=1 && go vet ./...` from the repository root — exit 0; `cmd/docmanager` has no tests or vet findings. |
| Runtime harness command/scenario and exact result | `GOPATH=/tmp/opencode/gopath GOMODCACHE=/tmp/opencode/gomodcache GOCACHE=/tmp/opencode/gocache go test ./... -run TestMCPStdioSQLiteRoundTrip -count=1` — exit 0; 1 test passed. It starts the spike as an MCP stdio subprocess, calls `sqlite_round_trip`, and reads the persisted row through a separate SQLite connection. |
| Static cross-build/package smoke | Native: `CGO_ENABLED=0 go build -o /tmp/go-mcp-sqlite-spike . && /tmp/go-mcp-sqlite-spike --package-smoke` — exit 0; emitted `go-mcp-sqlite spike package smoke passed`. Static builds: `CGO_ENABLED=0 GOOS=darwin GOARCH=arm64`, `GOOS=linux GOARCH=amd64`, and `GOOS=windows GOARCH=amd64` `go build` commands — all exit 0. |
| Rollback boundary | Remove `spike/go-mcp-sqlite/`, `.github/workflows/spike.yml`, root `go.mod`, `cmd/docmanager/main.go`, `TESTING.md`, and the `.docmanager/` ignore entry. This removes only the compatibility gate and empty foundation; no Phase 2+ behavior is affected. |

### Native Attempt Finalization

The corrective finalization reused the completed harness evidence without executing it again. Active ordinal 2 finished as `passed` using operation request ID `repository-documentation-manager-spike-finish-20260729-v2`, evidence revision `sha256:f11d80593994fe0885ee851cb229a131b0633104eee2aeccc44cdb31a5d510e7`, and terminal runtime revision `sha256:ac006a81e26190be25d6d8555ab066ea152bc03121674897d244b698afabe3a7`. The runtime ledger reports `next_action: complete`.

## Delivery Boundary

- Strategy: `auto-chain`, `stacked-to-main`.
- Slice: Gate / Go compatibility spike only.
- Out of scope: Phase 2 and later; no `internal/` product code was created.

## Corrective Work Unit: Phase 1 Contract Corrections

**Status**: Passed. This bounded correction preserves Phase 1 task completion and leaves Phase 2 task 2.2 pending.

### Corrections

- Aligned the design and task 2.2 with the normative read-only analysis contract: results contain candidates, rationale, confidence, and evidence; they do not contain patch output or a write/apply API.
- Updated the spike workflow so every native matrix leg builds `go-mcp-sqlite-spike` and executes `--package-smoke`, using `./go-mcp-sqlite-spike` on macOS/Linux and `.\go-mcp-sqlite-spike.exe` on Windows.
- Added `testing.Short()` guards before both external `go run` integration commands.

### Work Unit Evidence

| Evidence | Required value |
|---|---|
| Artifact consistency checks | `python3 -c '...'` checked that design/task have no `PatchProposal`, unified-diff, or proposal-file behavior; the normative spec retains its read-only requirement; and the workflow builds and uses native Unix/Windows paths — exit 0; printed `artifact consistency checks passed`. |
| Focused test command and exact result | Root: `go test -short ./... -count=1` — exit 0; `cmd/docmanager` has no test files. Spike: `go test -short ./... -count=1` — exit 0; 2 integration tests skipped. Guard proof: compiled spike test binary executed with `PATH=/nonexistent` and `-test.short -test.v` — exit 0; both external-command tests reported `SKIP`, proving neither launched `go run`. |
| Runtime harness command/scenario and exact result | `go test ./... -count=1` from `spike/go-mcp-sqlite` — exit 0; 2 tests passed, including MCP stdio SQLite persistence/readback and package smoke. Native: `CGO_ENABLED=0 go build -o /tmp/go-mcp-sqlite-spike . && /tmp/go-mcp-sqlite-spike --package-smoke` — exit 0; printed `go-mcp-sqlite spike package smoke passed`. CGo-free Darwin/arm64, Linux/amd64, and Windows/amd64 builds — all exit 0. A Windows workflow-path check confirmed `GOOS=windows ... -o /tmp/go-mcp-sqlite-workflow-check` creates `.exe`. |
| Rollback boundary | Revert only the read-only wording in `design.md` and task 2.2, the three workflow build/smoke steps, and the two short-mode guards in `spike/go-mcp-sqlite/main_test.go`; no Phase 2 implementation or Phase 1 checkbox is removed. |

### Delivery Boundary

- Strategy: `auto-chain`, `stacked-to-main`.
- Slice: `phase1-contract-corrections` only; no Phase 2 implementation.
- Review impact: native attempt recorded 53 changed lines at finish, below the 200-line native cap and the 800-line session budget.

### Native Attempt Finalization

Active ordinal 3 finished as `passed` with request ID `repository-documentation-manager-phase1-corrections-finish-20260729-v1`, evidence revision `sha256:9ac91106a3746b125ba39d093c1fe96ee253930d9d36de6fdf669db30e568d39`, terminal runtime revision `sha256:5343f9f327495204c870bfd93faf373af059fb46843929997236f4d36b10081d`, and harness disposition `reused`. The runtime ledger reports `next_action: complete`.

## Work Unit 2: Domain and Git Resolver

**Status**: Passed.

### Task Status

- [x] 2.1 RED classifier cases for untrusted documentation-like paths.
- [x] 2.2 Read-only domain scope, evidence, analysis, and outcomes.
- [x] 2.3 RED temporary-repository scope and root cases.
- [x] 2.4 Bounded shell-free Git resolver with typed failures.
- [x] 2.5 `document_change` outcome, failure, and non-mutation tests.
- [x] 2.6 Read-only `document_change` use case.

### RED → GREEN Evidence

| Contract case | RED | GREEN |
|---|---|---|
| 2.1 untrusted filenames | `go test ./internal/domain ./internal/adapters/git ./internal/app -count=1` before production files — exit 1; undefined domain contracts. | Same focused command after implementation — exit 0; domain package passed all five classifier cases. |
| 2.3 Git roots and scopes | Same RED command — exit 1; resolver package could not compile without contracts. | Focused command — exit 0; temporary repository covers absolute, relative, `git -C`-style root resolution, staged paths, `commit -a`, empty index, outside root, and non-repository root. |

### Work Unit Evidence

| Evidence | Required value |
|---|---|
| Focused test command and exact result | `GOPATH=/tmp/opencode/gopath GOMODCACHE=/tmp/opencode/gomodcache GOCACHE=/tmp/opencode/gocache go test ./internal/domain ./internal/adapters/git ./internal/app -count=1` — exit 0; 3 packages passed. |
| Runtime harness command/scenario and exact result | `GOPATH=/tmp/opencode/gopath GOMODCACHE=/tmp/opencode/gomodcache GOCACHE=/tmp/opencode/gocache go test ./internal/app -run TestDocumentChangePreservesTemporaryRepository -count=1 -v` — exit 0; 1 test with 3 temporary Git-repository scenarios passed: `README.md` → update, `cmd/server.go` → create, and `internal/format.go` → no-impact. |
| Repository-wide quality | `go test ./... -count=1` and `go vet ./...` with the same command-scoped Go caches — both exit 0; root suite passed and vet emitted no findings. |
| Non-mutation proof | Each runtime scenario snapshots `git status --porcelain=v1 -z` before/after analysis and reads the changed file afterward; all snapshots and contents matched. Resolver and use case expose no patch, write, apply, stage, commit, or sync API. |
| Rollback boundary | Remove only `internal/domain/{scope,evidence,analysis}.go`, `internal/adapters/git/resolver.go`, `internal/app/document_change.go`, and their Phase 2 tests. This removes the autonomous resolver work unit without touching Phase 1 or Phase 3+. |

### Delivery Boundary

- Strategy: `auto-chain`, `stacked-to-main`.
- Slice: PR 1 / work unit 2, `domain-and-git-resolver` only.
- Out of scope: receipt, ledger, CLI, MCP, assets, hooks, and all Phase 3+ work.
- Review impact: authored Phase 2 source and tests remain within the 800 changed-line cap.

### Native Ordinal 4 Terminal Facts (Preserved)

- Status: `passed`.
- Evidence: `sha256:b744ffe8684aace44aace3075eba510e12ab261d488e7fa1b8a7967873dc8efa`.
- Revision: `sha256:b035400b1e1f45b0121ca9d3906ba0491d88a1e9d00e0deeb6a479fbf8b3261d`.
- Review impact: `575/800` lines.

## Corrective Work Unit: Phase 2 Git Authority Retry

**Status**: Failed at native ordinal 5. This was the one authorized bounded corrective retry for Phase 2 PR 1 only; Phase 3+ was not edited.

### Corrections

- Range endpoints are independently resolved with `git rev-parse --verify <endpoint>^{commit}` and evidence exposes only the canonical object-ID range.
- Ordinal 5 attempted to hash deterministic `git diff --binary` content output while disabling external diff, text conversion, and inherited global/system configuration. Repository-local attributes still influenced it.
- Git stdout and stderr share a bounded streaming buffer. Crossing 1 MiB stops buffering immediately and returns typed `bounded_output` / `ErrOutputLimit`.
- Real temporary-Git tests covered range canonicalization, `commit -a`, staged/worktree scope, a newline-containing filename parsed from NUL-delimited output, same-path different-content identities, oversized adapter output, and missing-Git classification. They did not prove isolation from repository-local attributes/configuration.

### RED → GREEN Evidence

| Contract case | RED | GREEN |
|---|---|---|
| Canonical range, content identity, NUL-safe worktree paths, bounded output, unavailable Git | `go test ./internal/adapters/git -run 'TestResolver(CanonicalizesRangeAndBindsIdentityToContent|WorktreeAndNULDelimitedPaths|BoundsStreamingOutputAndClassifiesUnavailableGit)' -count=1` — exit 1 before production correction: missing `strings`, `boundedBuffer`, and `ErrOutputLimit` contracts. | `go test ./internal/domain ./internal/adapters/git ./internal/app -count=1` — exit 0; all 3 packages passed after correction. |

### Work Unit Evidence

| Evidence | Required value |
|---|---|
| Focused test command and exact result | `GOPATH=/tmp/opencode/gopath GOMODCACHE=/tmp/opencode/gomodcache GOCACHE=/tmp/opencode/gocache go test ./internal/domain ./internal/adapters/git ./internal/app -count=1` — exit 0; 3 packages passed. |
| Runtime harness command/scenario and exact result | `... go test ./internal/app -run TestDocumentChangePreservesTemporaryRepository -count=1 -v` — exit 0; one real temporary-Git test passed all 3 update/create/no-impact scenarios. Resolver tests also executed real Git range, worktree, staged, `commit -a`, newline-path, and oversized-diff scenarios. |
| Quality and formatting | `... go test ./... -count=1` — exit 0; `... go vet ./...` — exit 0; `gofmt -d` over corrected Go files and `git diff --check` produced no output. |
| Non-mutation proof | The runtime harness snapshots NUL-delimited `git status --porcelain=v1 -z` before/after and rereads the changed content; all three temporary repositories matched. The resolver uses only Git read commands and exposes no mutation API. |
| Rollback boundary | Revert only `internal/domain/scope.go`, `internal/adapters/git/resolver.go`, and `internal/adapters/git/resolver_test.go` correction behavior; Phase 1 and Phase 3+ remain independent. |

### Native Ordinal 5 Terminal Facts (Corrected)

- Outcome: `failed`; terminal revision: `sha256:e762c814cd75d42148e94bf40247fd8f64a6f364c25b0798243bfa4c8772ee61`.
- Evidence revision: `sha256:395b64b55f43fe0c849a928c176d634d4ec8b7a8ee7a991163f051dec7ca5973`; changed lines: `243`.
- Exact gate diagnosis: final Phase 2 gate failed because repository-local attributes and local Git configuration can still influence canonical identity, no isolation proof test exists, and apply-progress falsely recorded ordinal 5 as terminal before native finish.
- Harness disposition: `invalidated`. Cleanup evidence: no additional runtime harness was launched by the final gate; validation was read-only and created no temporary artifacts requiring cleanup.
- Process evidence: focused tests, full tests, vet, formatting, and non-mutation checks passed, but fresh-context validation failed on repository attributes/local-config isolation and native/artifact terminal-state mismatch.

### Native Ordinal 6 Isolation Objective

**Status**: Passed by authoritative native finish status.

- Terminal revision: `sha256:f4a5970fc85b13e8d0492c06d26e5e49949115f93067dbac74adcfbc7f3e8351`; evidence revision: `sha256:d332b12c83f67245dd65e2bab4cf34ddb32ebe6b4d8ddb7c0c62a294e78be87c`; changed lines: `136/200`.
- Canonical range/staged identity now hashes raw Git object metadata; worktree identity hashes each path with `git hash-object --no-filters`, so attributes and diff machinery cannot alter canonical bytes.
- The adversarial RED test failed before the fix under `.gitattributes`, `.git/info/attributes`, local/global/environment diff configuration, and passed after it; same-path content still changed identity.
- Focused Phase 2 tests, the three-scenario real non-mutation harness, `go test ./... -count=1`, `go vet ./...`, `gofmt -d`, and `git diff --check` passed. Harness disposition: `reused`; all temporary fixtures used `t.TempDir()` cleanup.

## Native Ordinal 7: Scope Discovery Isolation (Terminal)

**Status**: Passed by authoritative readback. Range and staged discovery use explicit no-rename/no-filter flags; worktree discovery compares index object IDs to `hash-object --no-filters`, so mode-only changes do not create content evidence.

| Scope | Adversarial proof |
|---|---|
| Range | `.gitattributes`, `.git/info/attributes`, local/global/system/inherited diff/textconv/external-diff settings, and rename poison preserve paths and identity; same-path content changes identity. |
| Staged | The same poison matrix preserves paths and identity; same-path staged content changes identity. |
| Worktree | The same poison matrix preserves paths and identity; mode-only/local `core.filemode` changes yield `empty_scope`, while content changes identity. |

Typed resolver failures: invalid refs return `invalid_revision`; content/hash reads return `content_read_failed`; resolver-level oversized output returns `bounded_output`.

### RED → GREEN and Work Unit Evidence

- RED: `go test ./internal/adapters/git -run 'TestResolver(RangeScopeIgnoresGitConfiguration|StagedScopeIgnoresGitConfiguration|WorktreeScopeIgnoresGitConfiguration|ClassifiesRevisionContentAndOutputFailures)' -count=1 -v` exited 1 before implementation (missing typed contracts).
- GREEN/focused: the same command exited 0; all four resolver tests passed.
- Phase 2: `go test ./internal/domain ./internal/adapters/git ./internal/app -count=1` — exit 0; 3 packages passed.
- Runtime non-mutation: `go test ./internal/app -run TestDocumentChangePreservesTemporaryRepository -count=1 -v` — exit 0; 3 temporary-repository scenarios passed.
- Quality: `go test ./... -count=1`, `go vet ./...`, `gofmt -d`, and `git diff --check` — exit 0/no output.
- Rollback: revert only `internal/domain/scope.go`, `internal/adapters/git/resolver.go`, and `internal/adapters/git/resolver_test.go`; no Phase 3+ behavior is involved.

### Native Ordinal 7 Terminal Facts

- Outcome: `passed`; request ID: `repository-documentation-manager-phase2-scope-isolation-finish-20260729-v1`.
- Evidence revision: `sha256:311189cf9a80a4000c74a45a6768f8621dbb6bc63edf244267b50dffcae636a3`; terminal runtime revision: `sha256:1c3f9de45fb9d9094b03a8ab775786f0574509132a59841a8bbdc28404a30959`; native impact: `184/200` lines.
- Harness disposition: `invalidated`; cleanup/process evidence are persisted in the native ledger. Authoritative status readback reports `complete: true`, `next_action: complete`.

## Native Ordinal 8: Identity-Stage Output Classification (Terminal)

**Status**: Passed by authoritative native readback after recovery from the interrupted editor session.

- Behavior: identity-stage output overflow now preserves the exact `bounded_output` classification instead of being wrapped as `content_read_failed`; actual content-read failures remain unchanged.
- Regression proof: `go test ./internal/adapters/git -run 'TestResolverClassifiesIdentityStageOutputOverflow' -count=1 -v` — exit 0; the resolver-level identity-stage overflow test passed.
- Full verification: `go test ./... -count=1` and `go vet ./...` — exit 0; `gofmt -d` and `git diff --check` produced no output.
- Outcome: `passed`; changed lines: `17/80`.
- Evidence revision: `sha256:ca203a6b07f61edc6e99f749f2a043ce103bab9b9dae64c3516bfdcb2827c3c5`.
- Terminal runtime revision: `sha256:011742752f6f406acf7069abc428659d2456d0eff1a0bbbce54d0e85c7eb3df4`.
- Harness disposition: `reused`; the interrupted worker left no active process or repository temporary files, and verification used command-scoped Go caches under `/tmp/opencode`.

## Native Ordinal 9: Post-Root Discovery Overflow Classification (Terminal)

**Status**: Passed by authoritative native finish and status readback.

- `Resolve` returns exactly `domain.ErrOutputLimit` when post-root path discovery exceeds the shared output bound; other discovery failures remain `resolve scope` wrapped.
- RED: discovery overflow after successful `show-toplevel` returned `resolve scope: bounded_output`; identity overflow still passed.
- GREEN: focused discovery + identity overflow tests passed (2 tests); Phase 2 tests, three-scenario non-mutation harness, root tests, vet, `gofmt -d`, and `git diff --check` passed.
- Rollback: `internal/adapters/git/resolver.go` and `internal/adapters/git/resolver_test.go` only.
- Request: `repository-documentation-manager-phase2-discovery-overflow-finish-20260729-v1`; evidence: `sha256:4cc4c55e2512717a13d9532ec7eeac054911e85ac1f69624f7ced78323e80e19`; terminal revision: `sha256:7d95bb6007fadda73b33241f90123961432b51835bca51fa80068613e74a6375`; impact: `17/60`; disposition: `reused`; status: `complete: true`, `next_action: complete`.

## Native Ordinal 10: Initial Root Discovery Exact Overflow Test (Terminal)

**Status**: Passed by authoritative native finish and status readback.

- The initial `rev-parse --show-toplevel` overflow regression now asserts `err == domain.ErrOutputLimit`, independently of identity-stage and post-root-discovery cases; production behavior was already correct.
- Three focused overflow tests passed; all Phase 2 tests, the three-scenario non-mutation harness, `go test ./... -count=1`, `go vet ./...`, `gofmt -d`, and `git diff --check` passed.
- Request: `repository-documentation-manager-phase2-root-overflow-test-finish-20260729-v1`; evidence: `sha256:2b441735dd1355bfe37be28dc0b8ff191d9aef1b5cfed707499e87d125605925`; terminal revision: `sha256:11388950d612d42112325161f95a73d22e8baef5eb52777eba1ac4418341cba1`; impact: `6/30`; disposition: `reused`; status: `complete: true`, `next_action: complete`.

## Work Unit 3: Receipts, Ledger, and CLI

**Status**: Corrected contracts passed local verification and authoritative native finish/readback.

### Task Status

- [x] 3.1 Canonical receipt RED test and semantic/content/version rejection cases.
- [x] 3.2 SHA-256 receipt canonicalization binding scope, evidence, paths, documentation blobs, outcome, and versions.
- [x] 3.3 SQLite `t.TempDir()` save/rebuild/verify and typed failure tests.
- [x] 3.4 Rebuildable, removable `.docmanager/ledger.db` SQLite receipt ledger.
- [x] 3.5 CLI document-change, verify, invalid-scope, and documentation non-mutation tests.
- [x] 3.6 CLI and contained install/doctor/uninstall/verify use cases with typed errors.

### RED → GREEN Evidence

| Contract | RED | GREEN |
|---|---|---|
| Canonical receipts | `go test ./internal/domain -run TestReceiptDigestIsCanonicalAndContentBound -count=1` — exit 1; receipt contracts were undefined. | Same test — exit 0; equal semantic inputs share one digest, while scope, evidence/content, and schema version changes differ. |
| SQLite ledger | `go test ./internal/adapters/sqlite -count=1` — exit 1; `Open` was undefined. | Same package — exit 0; save/rebuild/verify, mismatch, and invalid root pass. |

### Work Unit Evidence

| Evidence | Exact result |
|---|---|
| Focused test | `GOPATH=/tmp/opencode/gopath GOMODCACHE=/tmp/opencode/gomodcache GOCACHE=/tmp/opencode/gocache go test ./internal/domain ./internal/adapters/sqlite ./internal/app ./cmd/docmanager -count=1 -v` — exit 0; 4 packages passed, including canonical receipt, ledger, existing non-mutation, CLI verify, and invalid-scope cases. |
| Runtime harness | `... go test ./cmd/docmanager -run TestCLIDocumentChangeVerifyAndLifecycleWithoutDocumentationMutation -count=1 -v` — exit 0; one temporary Git repository exercised install, doctor, staged document-change, SQLite save, receipt verify, and uninstall. The test reread `README.md` before/after and it was unchanged. |
| Quality | `... go test ./... -count=1`; `... go vet ./...`; `gofmt -d`; and `git diff --check` — all exit 0/no output. |
| Rollback | Revert `internal/domain/receipt.go`, `internal/adapters/sqlite/`, `internal/app/{document_change,verify_receipt,lifecycle}.go`, `cmd/docmanager/`, root SQLite module dependency, and their tests. This removes only receipts/ledger/CLI behavior; Git evidence remains authoritative and Phase 1–2 remain intact. |

### Delivery Boundary

- Strategy: `auto-chain`, `stacked-to-main`.
- Slice: PR 2 / work unit 3, `receipts-ledger-cli` only.
- Out of scope: MCP, assets, hook, release documentation, commits, and PR creation.

## Native Ordinal 11 Facts (Preserved Exactly)

- Status: `passed`.
- Evidence: `sha256:4dab43102ab11b11ceb6217c6533698a39f634097d56134e095aecfeedf815a7`.
- Revision: `sha256:8505c1d6f51a6e6f52ad7ed21d3427bb729e5c7850ac08c927131ee1671ad86e`.
- Review impact: `633/800`.

## Corrective Work Unit: Phase 3 Receipt, Ledger, CLI, and Lifecycle Contracts

**Status**: Passed by authoritative native finish and status readback.

### Corrected Contracts

- Receipts retain canonical changed paths, scope, evidence, exact per-document SHA-256 digests, outcome, and schema/tool versions. Their digest is length-delimited and independently recomputed; duplicate paths are rejected.
- Verification first rejects any receipt whose submitted fields do not reproduce its digest, then independently re-resolves Git evidence, re-reads candidate bytes without following an escaping symlink, recomputes document digests, and compares every field against the current canonical receipt.
- SQLite persists the full receipt JSON, can query it, and remains a removable cache: deletion causes verification to fail typed, while a new authoritative analysis rebuilds and re-verifies it.
- CLI classifies unsupported scope values as `invalid_scope`, rejects tampered receipts, and the temporary Git harness proves content and Git status stay unchanged across verification.
- Lifecycle operations validate a non-symlink repository root, reject traversal and unowned/symlink `.docmanager`, write an ownership marker, and remove newly-created state when marker creation fails.

### RED → GREEN Evidence

| Contract | RED | GREEN |
|---|---|---|
| Receipt self-verification and canonical field binding | Focused test compilation failed before `ChangedPaths` and `ValidateDigest` existed. | `go test ./internal/domain ./internal/adapters/sqlite ./internal/app ./cmd/docmanager -count=1 -v` exited 0; canonical field/tamper cases passed. |
| Ledger rebuild and typed receipt query | Focused test compilation failed before `Ledger.Receipt` existed. | Same focused command exited 0; delete/rebuild/reverify and typed failure cases passed. |

### Work Unit Evidence

| Evidence | Exact result |
|---|---|
| Focused test command and exact result | `GOPATH=/tmp/opencode/gopath GOMODCACHE=/tmp/opencode/gomodcache GOCACHE=/tmp/opencode/gocache go test ./internal/domain ./internal/adapters/sqlite ./internal/app ./cmd/docmanager -count=1 -v` — exit 0; 4 packages passed. |
| Runtime harness command/scenario and exact result | `... go test ./cmd/docmanager -run TestCLIDocumentChangeVerifyAndLifecycleWithoutDocumentationMutation -count=1 -v` — exit 0; temporary Git+SQLite install, staged analysis, verify, receipt tamper rejection, status/content snapshots, and uninstall passed. |
| Full quality | `... go test ./... -count=1`, `... go vet ./...`, `gofmt -l cmd internal`, and `git diff --check` — exit 0/no output. |
| Rollback boundary | Revert only Phase 3 receipt, app, SQLite, CLI, lifecycle, and corresponding test changes; Phase 1–2 Git evidence and Phase 4+ remain untouched. |

### Native Ordinal 12 Terminal Facts

- Finish ID: `repository-documentation-manager-phase3-corrections-finish-20260729-v1`.
- Outcome: `passed`; impact: `503/800`; disposition: `reused`.
- Evidence: `sha256:dee39ac088fd5589335c136528bbe76dc8fbe6250d8100dcf949b9040b7277fe`.
- Terminal revision: `sha256:a2e68825d344bbf58370a212be473e9c2a04254abda4a21d1add392e2ff1990f`.
- Authoritative status: `complete: true`, `next_action: complete`.

## Native Ordinal 13: Phase 3 Safety and Proof Corrections (Terminal)

**Status**: Passed by authoritative finish and readback.

### Corrections

- Lifecycle install creates only a fresh `.docmanager` marker with the explicit `docmanager/1` ownership token. Existing unowned state, symlinked state or marker, non-regular marker, and mismatched marker content are rejected by install, doctor, and uninstall without following the marker symlink. New-state marker-write failure rolls back the directory.
- `document-change` validates the requested path as the exact Git root before opening SQLite. SQLite rejects symlinked or non-directory `.docmanager` state and never creates `ledger.db` through it.
- Ledger verification checks retrieval errors before receipt comparison, so closed, corrupt, query, and save failures are `ledger_failure`; only a missing/different stored receipt is `receipt_mismatch`.
- Receipt tests use a separate length-delimited SHA-256 oracle and fixed golden digest; all digest-bound fields, `Scope.Kind`, documentation path, and duplicate changed paths are covered. CLI verification covers the same tamper matrix plus invalid scope without Git/content mutation.

### RED → GREEN Evidence

| Contract | RED | GREEN |
|---|---|---|
| Lifecycle, SQLite, receipt, and CLI safety/proof cases | `go test ./internal/domain ./internal/adapters/sqlite ./internal/app ./cmd/docmanager -count=1` exited 1 after the new safety/proof tests were added and before production corrections. | `go test ./internal/domain ./internal/adapters/sqlite ./internal/app ./cmd/docmanager -count=1 -v` exited 0; all 4 packages passed. |

### Work Unit Evidence

| Evidence | Exact result |
|---|---|
| Focused test command and exact result | `go test ./internal/domain ./internal/adapters/sqlite ./internal/app ./cmd/docmanager -count=1 -v` — exit 0; all 4 packages passed. |
| Runtime harness command/scenario and exact result | `go test ./cmd/docmanager -run 'TestCLI(DocumentChangeVerifyAndLifecycleWithoutDocumentationMutation|PrevalidatesRepositoryBeforeOpeningLedger|RejectsInvalidScope)$' -count=1 -v` — exit 0; 3 temporary Git/SQLite CLI scenarios passed, including before/after Git-status and documentation-content assertions. |
| Full quality | `go test ./... -count=1`, `go vet ./...`, `gofmt -l cmd internal`, and `git diff --check` — all exit 0/no output. |
| Rollback boundary | Revert only Phase 3 changes in `internal/app/lifecycle.go`, `internal/adapters/{git,sqlite}/`, `internal/domain/receipt*`, `cmd/docmanager/`, and their tests; Phase 1–2 Git evidence and Phase 4+ remain independent. |

### Native Ordinal 13 Terminal Facts

- Finish ID: `repository-documentation-manager-phase3-safety-corrections-finish-20260730-v1`.
- Outcome: `passed`; native impact: `363/500`; disposition: `reused`.
- Computed evidence SHA: `sha256:fb87a48ada931d99b78b71de43744524ab97594e6bc5dffc5717abd168aca9ac`.
- Expected finish revision supplied: `sha256:084944438d6fc98a1016a7f343b3b6499df58feb5692b7ce3ecb669fbde537bd`.
- Authoritative readback: `complete: true`, `next_action: complete`, generation `13`.

## Native Ordinal 14: Traversal-Shaped Lifecycle Target Containment (Terminal)

**Status**: Passed by authoritative settle and readback.

- `lifecycleRoot` rejects a raw `..` component before `filepath.Abs` can normalize it. This blocks a target such as `<valid-repository>/nested/..` for install, doctor, and uninstall, while exact-root lifecycle behavior remains covered.
- RED: `go test ./internal/app -run TestLifecycleRejectsTraversalAndNonRepositoryTargets -count=1 -v` failed before the validation change: all three lifecycle operations returned nil for the traversal-shaped target.
- GREEN/focused lifecycle: `go test ./internal/app -run 'TestLifecycle(ContainsOnlyOwnedRepositoryState|RejectsUnsafeOwnershipMarkers|RejectsTraversalAndNonRepositoryTargets)|TestInstallRollsBackNewStateWhenOwnershipWriteFails' -count=1 -v` — exit 0; 4 top-level tests passed, including three traversal operations.
- Phase 3 suite: `go test ./internal/domain ./internal/adapters/sqlite ./internal/app ./cmd/docmanager -count=1 -v` — exit 0; 4 packages passed.
- CLI non-mutation harness: `go test ./cmd/docmanager -run TestCLIDocumentChangeVerifyAndLifecycleWithoutDocumentationMutation -count=1 -v` — exit 0; 1 temporary Git/SQLite lifecycle scenario passed without documentation mutation.
- Quality: `go test ./... -count=1`, `go vet ./...`, `gofmt -l internal/app/lifecycle.go internal/app/lifecycle_test.go`, and `git diff --check` — exit 0/no output.
- Rollback boundary: revert only `internal/app/lifecycle.go` and `internal/app/lifecycle_test.go`; this removes traversal-shape rejection without changing ownership, symlink, rollback, receipt, or Phase 4+ behavior.

### Native Ordinal 14 Terminal Facts

- Settle ID: `repository-documentation-manager-phase3-traversal-settle-20260730-v1`; outcome: `passed`; native impact: `51/80`; disposition: `reused`.
- Computed evidence revision: `sha256:37b6c0b34d0af84432f106b5b62e69046d3193607919fbe4abc280cc1d7586cb`; terminal runtime revision: `sha256:8ffd29c69945400de568fb823f640cb9837dc0065f8e64bd9fc8f4bb0c99c634`.
- Authoritative readback: `complete: true`, `next_action: complete`, generation `14`; temporary fixtures used `t.TempDir()` and left no persistent process or state.

## Work Unit 4: MCP, Assets, and Optional Hook

**Status**: Passed by authoritative native settle and readback.

### Task Status

- [x] 4.1 MCP stdio fixture proves CLI/MCP report and receipt parity plus typed failure output.
- [x] 4.2 Official Go SDK exposes explicitly-invoked `document_change`, `verify_receipt`, `install`, `doctor`, and `uninstall` tools.
- [x] 4.3 Hook input validation covers tracking, first push, explicit refspec, warn ambiguity, and fail-closed ambiguity.
- [x] 4.4 Embedded deterministic guidance, skill, and declarative pre-push configuration install only below owned `.docmanager/` state.

### Requirement Evidence

- MCP handlers validate explicit scope before Git/SQLite access, call the same application use cases as the CLI, and serialize only typed classifications in structured failures.
- `TestMCPStdioDocumentChangeReceiptAndLifecycle` starts `docmanager mcp` over `mcp.CommandTransport`, runs the five lifecycle/analysis tools, compares CLI and MCP reports exactly, verifies the receipt, and rejects invalid scope as `invalid_scope`.
- Install writes deterministic embedded assets and configuration under `.docmanager/`, rejects unsafe/symlinked/tampered state, doctor validates the payloads read-only, and uninstall removes only owned state.
- The declarative hook asset contains an argument array; `ValidatePrePush` reads standard ref-update records without shell execution and returns warning or typed failure for ambiguous input.

### Work Unit Evidence

| Evidence | Exact result |
|---|---|
| Focused tests | `GOPATH=/tmp/opencode/gopath GOMODCACHE=/tmp/opencode/gomodcache GOCACHE=/tmp/opencode/gocache go test ./internal/adapters/mcp ./internal/app ./cmd/docmanager -count=1 -v` — exit 0; MCP stdio, hook, lifecycle, and CLI suites passed. |
| Runtime harness | `go test ./internal/adapters/mcp -run TestMCPStdioDocumentChangeReceiptAndLifecycle -count=1 -v` — exit 0; one temporary Git repository completed MCP install, doctor, staged analysis, CLI parity, receipt verification, typed invalid scope, and uninstall. README bytes and NUL-delimited Git status matched before/after. |
| Full quality | `go test ./... -count=1`, `go vet ./...`, `gofmt -l cmd internal assets`, and `git diff --check` — all exit 0/no output. |
| Rollback | Revert Phase 4 files only: `assets/`, `internal/adapters/mcp/`, `internal/app/hook*`, Phase 4 portions of `internal/app/lifecycle*`, plus MCP module/main/scope wiring. This removes MCP/assets/hook behavior without touching Phase 1–3 evidence contracts. |

### Native Ordinal 15 Terminal Facts

- Settle ID: `repository-documentation-manager-phase4-settle-20260801-v1`; outcome: `passed`; native impact: `482/800` lines.
- Evidence revision: `sha256:0e8b71897a82e663af0ff5e7edc257bdab2d329c8061f35f54ae251c3cd6a736`; terminal runtime revision: `sha256:6ee7cc561f1a11c8f3dca488cb35fd69297aabebf8f855b8a1e2da5a94077030`.
- Harness disposition: `invalidated` (new runtime proof executed); cleanup uses `t.TempDir`, deferred MCP session close, uninstall of owned state, and before/after README plus Git-status assertions.
- Authoritative readback: `complete: true`, `next_action: complete`, generation `15`.

### Delivery Boundary

- Strategy: `auto-chain`, `stacked-to-main`.
- Slice: PR 3 / work unit 4, `mcp-assets-hook` only; Phase 5, commits, pushes, and PR creation are out of scope.

## Native Ordinal 16: Phase 4 Read-Only MCP and Receipt-Hook Correction

**Status**: Passed by authoritative native settle and readback.

### Corrected Contracts

- MCP exposes exactly `document_change` and `verify_receipt`; lifecycle operations remain CLI-only. The stdio integration fixture enumerates the exact exposed set and rejects mutation-capable lifecycle exposure.
- Pre-push input accepts bounded, structurally valid standard update records only. It resolves tracking and explicit-refspec updates as `remote..local`; a first push resolves `local^..local`.
- Each resolved scope must have a stored receipt that passes canonical digest, current Git evidence, documentation-content, and ledger equality checks. Missing, stale, mismatched, and tampered receipts return `receipt_mismatch`; malformed or over-limit update input remains warn-or-fail ambiguous by mode.

### Work Unit Evidence

| Evidence | Exact result |
|---|---|
| Focused test command and exact result | `GOPATH=/tmp/opencode/gopath GOMODCACHE=/tmp/opencode/gomodcache GOCACHE=/tmp/opencode/gocache go test ./internal/adapters/mcp ./internal/app ./cmd/docmanager -count=1 -v` — exit 0; MCP exact-tool-set, hook adversarial receipt, real temporary-Git update, lifecycle, and CLI suites passed. |
| Runtime harness command/scenario and exact result | `go test ./internal/app -run TestPrePushVerifierRequiresMatchingStoredReceipt -count=1 -v` — exit 0; real temporary Git tracking, first-push, explicit-refspec, warn/fail ambiguity, missing, tampered, and stale receipt scenarios passed. `go test ./internal/adapters/mcp -run TestMCPStdioDocumentChangeAndReceipt -count=1 -v` — exit 0; stdio server exposed only the two read-only tools and preserved README/Git status. |
| Full quality | `go test -race ./internal/adapters/mcp ./internal/app ./cmd/docmanager -count=1`, `go test ./... -count=1`, `go vet ./...`, `gofmt -l cmd internal`, and `git diff --check` — all exit 0/no output. |
| Rollback boundary | Revert only `internal/adapters/mcp/mcp.go`, `internal/adapters/mcp/mcp_test.go`, `internal/app/hook.go`, `internal/app/hook_test.go`, `internal/adapters/sqlite/ledger.go`, and the hook CLI wiring in `cmd/docmanager/main.go`. This restores the prior Phase 4 surface without changing Phase 5 or repository documentation content. |

### Native Ordinal 16 Terminal Facts

- Settle ID: `repository-documentation-manager-phase4-contract-correction-settle-20260801-v1`; outcome: `passed`; native impact: `304/400` lines.
- Evidence revision: `sha256:44d6caa44e4163069ddd052381a6b590ef7ddaf95724640e8e64f29960a0173c`; terminal runtime revision: `sha256:3907f1f715108fa69037f1ba72f0b36fb513c02eb2fb5a8ccc787817529328ce`.
- Harness disposition: `invalidated`; temporary Git repositories used `t.TempDir`, the MCP session closed, and no hook process or documentation mutation persisted.
- Authoritative readback: `complete: true`, `next_action: complete`, generation `16`.

## Native Ordinal 17: Phase 4 Git Ref and First-Push Evidence Correction

**Status**: Passed by authoritative native settle and readback.

### Corrected Contracts

- The pre-push parser validates local and remote refs through bounded shell-free `git check-ref-format` invocations; traversal, double-dot, malformed, and unsafe names fail through the existing warn/fail ambiguity behavior. Valid deletion updates retain their zero-local-OID semantics and create no evidence scope.
- Object-ID length is derived from the repository's Git object format. A zero remote OID derives an `initial` scope anchored to the verified local commit, not `local^..local`.
- Initial-scope resolution uses Git's complete reachable history with `--root`; it deduplicates and sorts all introduced paths and binds raw history identity. Receipt re-verification requires the exact canonical initial scope and current content.

### Work Unit Evidence

| Evidence | Exact result |
|---|---|
| Focused test command and exact result | `GOPATH=/tmp/opencode/gopath GOMODCACHE=/tmp/opencode/gomodcache GOCACHE=/tmp/opencode/gocache go test ./internal/app -run 'TestPrePush(ScopeRejectsArbitraryTokens|ScopeRejectsGitInvalidRefs|VerifierRequiresMatchingStoredReceipt|VerifierInitialPushBindsCompleteReachableEvidence|VerifierRootInitialPushRequiresExactReceipt)$' -count=1 -v` — exit 0; 5 tests passed. |
| Runtime harness command/scenario and exact result | The focused temporary-Git tests — exit 0 — used `t.TempDir()` repositories to prove Git-invalid local/remote refs fail, a root first push passes only with its exact receipt, a multi-commit first push binds `README.md` and `cmd/server.go`, and missing/tampered/stale/mismatched receipts fail without repository mutation. |
| CLI and race regression | `... go test ./cmd/docmanager -count=1 -v` — exit 0; 3 tests passed. `... go test -race ./internal/app ./cmd/docmanager -count=1` — exit 0. |
| Full quality | `... go test ./... -count=1`, `... go vet ./...`, `gofmt -l internal/domain/scope.go internal/adapters/git/resolver.go internal/app/hook.go internal/app/hook_test.go`, and `git diff --check` — all exit 0/no output. |
| Rollback boundary | Revert only `internal/domain/scope.go`, `internal/adapters/git/resolver.go`, `internal/app/hook.go`, and `internal/app/hook_test.go`; this removes task 4.3 ref/first-push hardening without changing MCP, assets, lifecycle, or Phase 5. |

### Native Ordinal 17 Terminal Facts

- Acquire ID: `repository-documentation-manager-phase4-ref-firstpush-acquire-20260801`; settle ID: `repository-documentation-manager-phase4-ref-firstpush-settle-20260801-v1`; outcome: `passed`; native impact: `216/300` lines.
- Evidence revision: `sha256:3dca656ec111f86356c1d1784f793e384da32645d650309a113d28a257fee92b`; terminal runtime revision: `sha256:284611e3023f78e321ac4dbb7008b634014f617b4edd667a20cb9824a24de7f4`.
- Harness disposition: `invalidated`; all temporary repositories used `t.TempDir()`, and no hook process, documentation mutation, or persistent temporary state remains.
- Authoritative readback: `complete: true`, `next_action: complete`, generation `17`.

## Work Unit 5: Release CI, Packaging, and Documentation

**Status**: Passed by authoritative native settle and readback.

### Task Status

- [x] 5.1 Release-quality CI covers formatting, full tests, race checks, vet, supported native package smoke, and deterministic CGo-free cross-builds.
- [x] 5.2 Root README documents the explicit read-only concept, build and CLI paths, MCP surface, lifecycle, receipt-backed hook behavior, troubleshooting, uninstall, and disposable acceptance walkthrough.
- [x] 5.3 Local full verification, executable/package smoke, receipt and hook proof, and mutation snapshots completed; results are recorded in the README.

### Work Unit Evidence

| Evidence | Exact result |
|---|---|
| Focused test command and exact result | `GOPATH=/tmp/opencode/gopath GOMODCACHE=/tmp/opencode/gomodcache GOCACHE=/tmp/opencode/gocache go test ./... -count=1` — exit 0; six tested packages passed and `assets` has no test files. `go test -race ./internal/app ./internal/adapters/mcp ./cmd/docmanager -count=1` — exit 0; all three packages passed. |
| Runtime harness command/scenario and exact result | CGo-free `docmanager` executable in a disposable Git repository installed embedded assets, analyzed `base..tip`, verified its receipt, accepted a matching fail-mode pre-push update, passed doctor, and uninstalled — exit 0. It emitted `receipt=verified hook=valid assets=removed`. |
| Package/build verification | Native `CGO_ENABLED=0 go build -trimpath -buildvcs=false` plus `doctor` — exit 0. CGo-free Linux/amd64, macOS/arm64, and Windows/amd64 `go build -trimpath -buildvcs=false` outputs were all nonempty. CI uses the same matrix and runs native executable smoke on Ubuntu, macOS, and Windows. |
| Quality | `go vet ./...`, `test -z "$(gofmt -l cmd internal spike)"`, and `git diff --check` — exit 0/no output. |
| Non-mutation snapshots | Disposable repository before/after Git status SHA-256: `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`; index tree: `d8f3f0f0156415c2455efe27a139445641b602fd`; README SHA-256: `7b9a72466d3960eb2aacccfc848939453490db0678bd4725def3f789b891c919`. All matched. |
| Cleanup | Harness repositories were trap-cleaned; `/tmp/opencode/phase5-bin` and `/tmp/opencode/phase5-cross` were removed after validation. No process, repository state, or documentation mutation persisted. |
| Rollback boundary | Revert only `.github/workflows/ci.yml` and `README.md`; this removes release automation and user documentation without changing runtime, receipt, MCP, lifecycle, or hook semantics. |

### Delivery Boundary

- Strategy: `auto-chain`, `stacked-to-main`.
- Slice: PR 4 / work unit 5, `release-docs-packaging` only; no release publication, commit, push, PR creation, archive, or Phase 6 work.
- Review impact: native authority recorded `212/500` changed lines.

### Native Ordinal 18 Terminal Facts

- Acquire ID: `repository-documentation-manager-phase5-acquire-20260801`; settle ID: `repository-documentation-manager-phase5-settle-20260801-v1`; outcome: `passed`; generation: `18`; impact: `212/500`.
- Evidence revision: `sha256:58a929635fe535747a74166bba78654612ee6d1ad4c5d5f14bfe845da475d711`; terminal runtime revision: `sha256:2a5556ff21d51a88bfb732d5efe18ffa3dfdea602adde39d8ae481aa2027e0fb`.
- Harness disposition: `invalidated`; authoritative readback reports `complete: true`, `next_action: complete`.

## Native Ordinal 19: Phase 5 CI and README Correction

**Status**: Passed by authoritative native settle.

### Correction

- CI package smoke now asserts the actual lifecycle-installed `assets.Files` path: `.docmanager/guidance/AGENTS.md` on Unix and Windows.
- README extracts the complete `.Receipt` object emitted by `document-change` and passes it to `verify`; its assertion checks that `.Receipt.digest` is a nonempty string, so absent, null, and empty values fail.

### Work Unit Evidence

| Evidence | Exact result |
|---|---|
| Focused documentation/runtime command | The changed README walkthrough commands ran in a disposable Git repository — exit 0; `document-change` emitted a complete receipt, `verify` returned `{"verified":true}`, and the lowercase digest assertion passed. A digest-only `{"digest":"sha256:..."}` receipt exited nonzero with `receipt_mismatch`. |
| CI smoke and builds | Native Unix package smoke installed, checked `.docmanager/guidance/AGENTS.md`, doctored, and uninstalled — exit 0. CGo-free Linux/amd64, macOS/arm64, and Windows/amd64 builds were nonempty. Static workflow inspection confirmed both Unix and Windows assertions use the installed path; local `pwsh` was unavailable. |
| Quality and non-mutation | `go test ./... -count=1` (six tested packages), race tests (three packages), `go vet ./...`, `gofmt -l cmd internal spike`, and `git diff --check` all passed. Before/after workspace Git-status SHA-256, index tree, and README SHA-256 snapshots matched. |
| Rollback | Revert only `.github/workflows/ci.yml` and `README.md`; runtime, Phase 1–4, and lifecycle semantics are unchanged. |

### Native Ordinal 19 Terminal Facts

- Acquire ID: `repository-documentation-manager-phase5-ci-readme-acquire-20260801`; settle ID: `repository-documentation-manager-phase5-ci-readme-settle-20260801-v1`; outcome: `passed`; generation: `19`; impact: `12/120` production/documentation changed lines.
- Evidence revision: `sha256:0266e2594209f9a5a18436c088ab8df6f989a4b635e27209158dfde051bc1320`; harness disposition: `invalidated`.
- Cleanup: disposable repositories and `/tmp/opencode/phase5-ci-readme` were removed; no process or repository mutation remained.
