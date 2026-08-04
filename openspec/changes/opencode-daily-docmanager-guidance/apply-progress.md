# Apply Progress: OpenCode Daily DocManager Guidance

## Result Contract

- **status**: success
- **executive_summary**: Work Unit 1 adds the versioned OpenCode guidance asset, catalogued predecessor upgrade handling, and ownership-safe paired OpenCode MCP plus additive instructions configuration. Only tasks 1.1–1.5 were implemented; Units 2–4 remain untouched.
- **artifacts**: `assets/opencode.md`; `assets/assets.go`; `internal/app/lifecycle.go`; `internal/adapters/agent/{opencode,registry}.go`; focused tests; `tasks.md`; this file.
- **next_recommended**: Apply Work Unit 2 only.
- **risks**: The paired guidance path is supplied through `OpenCodeOptions.Guidance`; CLI lifecycle orchestration and observability remain intentionally deferred to Unit 2.
- **skill_resolution**: paths-injected — `sdd-apply`, `go-testing`, and `work-unit-commits` were read before task work.

## Completed Tasks

- [x] 1.1 RED guidance workflow/prohibition contract
- [x] 1.2 RED asset upgrade, drift, and user-file preservation contract
- [x] 1.3 GREEN versioned guidance asset catalog and lifecycle
- [x] 1.4 RED paired configuration ownership/refusal contract
- [x] 1.5 GREEN atomic paired MCP/instructions configuration

## TDD Cycle Evidence

| Task | Test File | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|---|---|---|---|---|---|---|---|
| 1.1 | `internal/app/lifecycle_test.go` | Unit | `go test ./internal/app ./internal/adapters/agent` passed before edits (2 packages) | Initial focused run failed to compile: `assets.OpenCodeGuidance` undefined | Focused suite passed: 2 packages | Required workflow plus 22 prohibition assertions | Guidance wording adjusted after a real failing assertion; focused suite passed |
| 1.2 | `internal/app/lifecycle_test.go` | Integration | Same baseline | Initial focused run failed to compile: catalog/snapshots undefined | Focused suite passed: fresh, known-old upgrade, edited/substituted/missing refusal, user-file preservation | Multiple lifecycle state transitions | Extracted `knownOpenCodeGuidance`; focused suite passed |
| 1.3 | `internal/app/lifecycle_test.go` | Integration | Same baseline | Covered by 1.1/1.2 RED asset contract | Focused suite passed after asset/catalog/lifecycle implementation | Current and catalogued predecessor contents exercise distinct paths | Clean extraction retained |
| 1.4 | `internal/adapters/agent/core_opencode_test.go` | Integration | Same baseline | Initial focused run failed to compile: `NewGuidanceIdentity` and `OpenCodeOptions.Guidance` undefined | Focused suite passed: pair append/no-op/exact unconfigure plus four refusal cases | Preserved existing instruction order and tested unsupported, incomplete, and conflicting states | Shared test helper retained; focused suite passed |
| 1.5 | `internal/adapters/agent/core_opencode_test.go` | Integration | Same baseline | Pair inspection RED failed with edited guidance returning nil; it required drift refusal | Focused suite passed after paired inspection validates MCP, instruction, and guidance identity | Fresh pair, idempotent pair, exact removal, and edited-guidance inspection exercise distinct branches | Isolated identity and instruction validation helpers; focused suite passed |

## Work Unit Evidence

| Evidence | Exact result |
|---|---|
| Focused test | `go test ./internal/app ./internal/adapters/agent` → exit 0; both packages passed. |
| Full configured runner | `go test ./...` → exit 0; 9 tested packages passed and `assets` reported no test files. |
| Runtime harness | `go test ./internal/adapters/agent -run TestOpenCode -v` → exit 0; 9 top-level OpenCode tests passed, including the paired guidance lifecycle and 4 no-write refusal subcases. Parent `sdd-attempt settle` completed with state `complete`; runtime-attempt token: `sha256:dd096b52b4fe939ffc9052c1a072ef5d442f41d0bf36f1b6d4a2f6d2d906e760`. |
| Cleanup/process evidence | All Go test commands exited 0; `git diff --check` exited 0. The harness uses temporary directories and completed without a persistent external process. |
| Changed lines | 343 authored lines: 336 additions and 7 deletions (325/7 tracked source/test lines plus 11-line new guidance asset). This is within the 650-line Work Unit cap; overall delivery remains maintainer-approved `size:exception`. |
| Rollback boundary | Reverse order: first revert the paired configuration identity/ownership code and its adapter tests (`internal/adapters/agent/{opencode,registry}.go`, `core_opencode_test.go`); then remove the guidance asset/catalog/lifecycle upgrade changes and lifecycle tests (`assets/opencode.md`, `assets/assets.go`, `internal/app/lifecycle.go`, `internal/app/lifecycle_test.go`). Retain `.docmanager` state and unrelated user configuration. |

## Implementation Deviation

- No `internal/adapters/agent/merge.go` change was required: existing helpers sufficed for the atomic paired configuration behavior. The implemented and rollback-scoped adapter files are `opencode.go`, `registry.go`, and `core_opencode_test.go`.

## Scope Boundary

- Delivery: `exception-ok`, explicit maintainer-approved `size:exception`.
- Work unit: 1 — guidance asset and owned configuration pair.
- No Unit 2, 3, or 4 production, test, documentation, CLI, MCP metadata, status/doctor, receipt, acceptance, Git authority, commit, push, tag, or release work was started.
- No source-mutating normalizer was run after final source verification.

## Work Unit 2 Result Contract

- **status**: success
- **executive_summary**: Work Unit 2 implements ordered workspace → OpenCode pair configuration → bounded probe handling, compensating unconfigure only for a pair created by the current attempt, OpenCode-only inspection commands, bounded MCP guidance, and receipt-review eligibility primitives.
- **artifacts**: `cmd/docmanager/{install.go,install_test.go,main.go,acceptance_test.go}`; `internal/adapters/mcp/{mcp.go,mcp_test.go}`; `internal/app/{receipt_snapshot.go,document_change_test.go}`; `tasks.md`; this merged progress record.
- **next_recommended**: Apply Work Unit 3 only; do not start Phase 4.
- **delivery**: `exception-ok`; maintainer-approved `size:exception`; no commit, push, tag, release, or Git review-authority mutation.
- **risks**: This corrective rerun changes phase-contract evidence only; it does not alter product behavior, tests, prior Unit 2 evidence, or runtime execution. The recorded Unit 2 evidence remains the basis for subsequent verification.
- **skill_resolution**: paths-injected — `sdd-apply` and `work-unit-commits` were read before this evidence-only correction.
- **parent_attempt_settlement**: Parent Unit 2 `sdd-attempt settle` completed with state `complete`.
- **runtime_attempt_token**: `sha256:5fd1be79fc6cb3ad8f17a6939201900b256fd84e1616cbb106dde6e2a708eed5`

### Completed Tasks

- [x] 2.1 Installer retained-workspace failure reporting RED coverage
- [x] 2.2 Probe-failure rollback ordering and pre-existing-pair retention RED coverage
- [x] 2.3 Ordered installer orchestration and compensating rollback GREEN implementation
- [x] 2.4 OpenCode inspection boundary coverage
- [x] 2.5 OpenCode-only `status` / `doctor` command boundary
- [x] 3.1 MCP explicit-scope, one-call, read-only, review metadata RED coverage
- [x] 3.2 Bounded MCP metadata GREEN implementation
- [x] 3.3 Receipt scope/bytes/consumption eligibility RED coverage
- [x] 3.4 Receipt snapshot, consumption, and explicit-review gate GREEN implementation

### TDD Cycle Evidence

| Task | Test File | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|---|---|---|---|---|---|---|---|
| 2.1 | `cmd/docmanager/install_test.go` | Integration | `go test ./cmd/docmanager ./internal/adapters/mcp ./internal/app ./internal/adapters/agent` → exit 0 | `WorkspaceState undefined` compile failure | retained workspace states pass | configuration and probe failure | focused suite green |
| 2.2 | `cmd/docmanager/install_test.go` | Integration | same baseline | expected `workspace,configure,probe,unconfigure`; got no rollback | focused installer tests pass | newly created pair rolls back; pre-existing pair remains | rollback conditioned on initial inspection |
| 2.3 | `cmd/docmanager/install_test.go` | Integration | same baseline | covered by 2.1/2.2 failure seams | focused installer tests pass | failure and successful configure/probe paths | compact runtime seam retained |
| 2.4 | `cmd/docmanager/install_test.go` | Integration | same baseline | OpenCode inspection helper absent | focused command test passes | status/doctor share the one bounded Status probe path | OpenCode-only helper retained |
| 2.5 | `cmd/docmanager/install_test.go` | Integration | same baseline | `runOpenCode undefined` compile failure | JSON inspection output passes | both status and doctor accepted without generic composition | no generic resource path added |
| 3.1 | `internal/adapters/mcp/mcp_test.go` | Unit | same baseline | `toolDescription undefined` compile failure | required scope/read-only/review text passes | both tools validated | descriptions centralized |
| 3.2 | `internal/adapters/mcp/mcp_test.go` | Unit | same baseline | covered by 3.1 metadata contract | MCP package passes | document and verify variants | shared base description |
| 3.3 | `internal/app/document_change_test.go` | Unit | same baseline | `ReceiptSnapshot` / `ReceiptEligibility undefined` compile failure | table test passes | unchanged, scope, considered bytes, unrelated bytes, and consumed paths | pure eligibility function extracted |
| 3.4 | `internal/app/document_change_test.go`, `internal/adapters/mcp/mcp_test.go` | Unit | same baseline | covered by 3.3 contract | app/MCP focused suites pass | review refusal and consumption included | snapshot remains caller-owned and side-effect bounded |

### Work Unit Evidence

| Evidence | Exact result |
|---|---|
| Focused tests | `go test ./internal/adapters/mcp -run 'Test(ToolDescriptionsRequireExplicitBoundedReviewWorkflow|VerifyReceiptRefusesWithoutExplicitHumanReview)' -count=1 && go test ./internal/app -run TestReceiptEligibilityRequiresReviewAndUnchangedConsideredBytes -count=1 && go test ./cmd/docmanager -run 'TestInstall|TestOpenCodeInspection' -count=1` → exit 0; all three packages passed. |
| Runtime harness | `go test ./cmd/docmanager -run TestAcceptance -v` → exit 0; 8 acceptance tests passed, 1 macOS-only lexical-path test skipped. `go build -o /tmp/docmanager ./cmd/docmanager && /tmp/docmanager --help` → exit 0. |
| Full configured runner | `go test ./...` → exit 0; 9 tested packages passed and `assets` reported no test files. |
| Cleanup/process evidence | `git diff --check` → exit 0. The acceptance harness uses temporary repositories and MCP stdio sessions, which close through deferred cleanup; no persistent external process remains. |
| Changed lines | Work Unit 2 authored delta: 232 additions and 19 deletions, 251 changed lines. This remains below the 850-line work-unit limit; the aggregate change remains the approved `size:exception`. |
| Rollback boundary | Revert Unit 2 only: `cmd/docmanager/{install.go,main.go,install_test.go,acceptance_test.go}`, `internal/adapters/mcp/{mcp.go,mcp_test.go}`, and `internal/app/{receipt_snapshot.go,document_change_test.go}`. This removes installer orchestration, OpenCode-only inspection, MCP metadata, and receipt eligibility while retaining Unit 1 guidance/pair lifecycle and unrelated workspace state. |

### Unit 2 Scope Boundary

- No Phase 4 task was started.
- No source-mutating normalizer was run after the final source verification.
- The reverse-order rollback is safe: revert Unit 2 first, then (only if required) Unit 1 guidance/pair changes; retain `.docmanager` workspace state and unrelated user configuration.

## Work Unit 3 Result Contract

- **status**: success
- **executive_summary**: Work Unit 3 adds a deterministic OpenCode guidance transcript acceptance oracle, documents the bounded daily workflow in the three requested OpenCode sections only, and completes the final formatting, test, build, runtime-help, cleanup, and scope-boundary evidence.
- **artifacts**: `cmd/docmanager/acceptance_test.go`; `README.md`; `docs/agents.md`; `ONBOARDING.md`; `tasks.md`; this merged progress record.
- **next_recommended**: Parent-owned `sdd-verify`; do not archive or settle Git review authority from apply.
- **delivery**: `exception-ok`; maintainer-approved `size:exception`; Work Unit 3 is below its 650-line cap; no commit, push, tag, release, archive, or Git review-authority mutation.
- **runtime_attempt_token**: `sha256:6ec3190f5670ffcad1a6cb0f0bb8e69ca03d9f44df43c05011ba0d7e2595a56c`
- **parent_attempt_settlement**: Parent Unit 3 `sdd-attempt settle` completed with state `complete`.
- **risks**: The transcript oracle is intentionally deterministic test-double evidence, not proof of an LLM runtime. It binds the embedded guidance contract to actual configuration/no-op/drift/unconfigure, owned-asset upgrade, and OpenCode-only inspection seams; parent verification remains responsible for final independent acceptance.
- **skill_resolution**: paths-injected — `sdd-apply`, `go-testing`, `work-unit-commits`, and `cognitive-doc-design` exact SKILL.md files, plus `sdd-apply/strict-tdd.md`, were read before task work.

### Completed Tasks

- [x] 4.1 RED acceptance transcript oracle for lifecycle, explicit scope, negative no-call, review, invalidation/re-analysis, and one-time verification conditions
- [x] 4.2 GREEN deterministic fakes plus real owned-pair lifecycle checks preserving unrelated configuration
- [x] 4.3 OpenCode-only daily workflow documentation in README, agent guide, and onboarding guide
- [x] 4.4 Final gofmt, focused/full tests, build, runtime smoke, diff check, cleanup, and scope audit

### TDD Cycle Evidence

| Task | Test File | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|---|---|---|---|---|---|---|---|
| 4.1 | `cmd/docmanager/acceptance_test.go` | Acceptance/integration | Existing Work Unit 2 full suite evidence: `go test ./...` exit 0; final suite repeated below | `go test ./cmd/docmanager -run TestAcceptanceOpenCodeGuidanceTranscript -count=1` → exit 1, compile failure: `runOpenCodeTranscript` and `assertTranscript` undefined | Same focused command → exit 0 after the oracle and assertions were implemented | Config/no-op/upgrade/drift, status/doctor, exact unconfigure, staged/range, five negative no-call triggers, review/invalidation/re-analysis, and consumed verification branches | The oracle checks embedded guidance clauses first, then executes owned-pair lifecycle and read-only inspection fakes; focused test remained green |
| 4.2 | `cmd/docmanager/acceptance_test.go` | Acceptance/integration | Same baseline | Covered by 4.1's missing-oracle RED seam before any implementation support existed | Focused command exit 0; actual adapter configuration preserves `theme`, user instruction, and unrelated MCP; exact removal preserves them | Current and predecessor guidance bytes exercise upgrade; matching bytes exercise no-op; altered bytes exercise drift refusal; status/doctor fake records only bounded probes | Compact test-only oracle/fake support retained; no production behavior was added |
| 4.3 | `README.md`, `docs/agents.md`, `ONBOARDING.md` | Documentation | N/A — documentation-only task; preceding focused test passed | N/A — no production behavior; docs were constrained by the already-green guidance acceptance contract | Final focused command exit 0 after documentation edits | Three reader contexts cover the same explicit scopes, human review, invalidation, rollback, and non-goals | Documentation remains additive and limited to OpenCode sections |
| 4.4 | `cmd/docmanager/acceptance_test.go` and repository sources | Integration | `go test ./cmd/docmanager -run TestAcceptanceOpenCodeGuidanceTranscript -count=1` exit 0 before final normalization | N/A — verification/cleanup task; no production behavior added | After `gofmt -w cmd internal`: focused test exit 0, `go test ./...` exit 0, and `go build ./cmd/docmanager` exit 0 | Focused acceptance plus complete suite exercise distinct boundaries; runtime help independently proves the built CLI path | Final source-mutating `gofmt` ran before final verification; no further source mutation followed |

### Work Unit Evidence

| Evidence | Exact result |
|---|---|
| RED transcript | `go test ./cmd/docmanager -run TestAcceptanceOpenCodeGuidanceTranscript -count=1` → exit 1; compile errors at `acceptance_test.go:88` and `:105` for undefined `runOpenCodeTranscript` and `assertTranscript`. |
| Focused test | After final `gofmt -w cmd internal`, `go test ./cmd/docmanager -run TestAcceptanceOpenCodeGuidanceTranscript -count=1` → exit 0; `ok github.com/desatatufuria/mcp-doc-manager/cmd/docmanager 0.777s`. |
| Full configured runner | `go test ./...` → exit 0; `assets` reported no test files and all 9 tested packages passed. |
| Build | `go build ./cmd/docmanager` → exit 0. The generated workspace `docmanager` binary was removed during cleanup. |
| Runtime harness | `go build -o /tmp/docmanager ./cmd/docmanager && /tmp/docmanager --help` → exit 0; help listed `mcp`, `document-change`, `verify`, `install`, `workspace`, `release`, and `agent`. `/tmp/docmanager` was removed during cleanup. |
| Formatting and diff | `gofmt -w cmd internal` ran before the final focused/full verification; `git diff --check` → exit 0. |
| Cleanup/process evidence | Both generated binaries (`/workspace/docmanager`, `/tmp/docmanager`) were removed and absence was asserted. Tests use temporary directories and deferred MCP/session cleanup; no persistent external process remains. |
| Scope/process audit | `git status --short` and changed-path audit show no `.git/hooks`, repository/user `AGENTS.md`, other-agent adapter, release adapter, generic composition, installer-foundation, commit, push, tag, or archive mutation. The only Work Unit 3 product paths are the acceptance test and requested OpenCode documentation sections. |
| Changed lines | Work Unit 3 tracked source/documentation delta is 110 changed lines (the aggregate tracked diff is 704 lines; recorded Unit 1 and Unit 2 deltas are 343 and 251 respectively). The OpenSpec task/progress append is planning evidence and excluded from authored product count. This is within the 650-line Work Unit cap. |
| Rollback boundary | Reverse order: (1) revert this Unit 3 acceptance oracle and OpenCode-only documentation sections in `cmd/docmanager/acceptance_test.go`, `README.md`, `docs/agents.md`, and `ONBOARDING.md`; (2) only if required, revert Unit 2 command/inspection/MCP/receipt hunks; (3) only then revert Unit 1 guidance/pair lifecycle hunks. Retain `.docmanager`, user instructions, unrelated configuration, hooks, and Git history. |

### Unit 3 Scope Boundary

- All tasks 1.1–4.4 are visibly checked in `tasks.md`; previous Unit 1 and Unit 2 evidence above is preserved verbatim.
- No source-mutating command ran after the final `gofmt` and before focused/full verification, build, runtime help smoke, cleanup, and diff check.
- No `sdd-verify` was started; parent owns final verification.
