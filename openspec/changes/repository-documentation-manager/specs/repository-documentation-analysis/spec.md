# Repository Documentation Analysis Specification

## Purpose

Define read-only Git analysis and deterministic receipts.

## Requirements

### Requirement: Explicit Git Scope

The system MUST require exactly one supported, caller-selected Git scope: a revision range, staged changes, or worktree changes. It MUST report the resolved scope and content identity as evidence and MUST reject empty, unsupported, or ambiguous scopes.

#### Scenario: Analyze a revision range

- GIVEN a repository and a valid caller-selected revision range
- WHEN analysis is requested
- THEN the result identifies the resolved revisions and content identity

#### Scenario: Reject an ambiguous scope

- GIVEN a request with multiple or no Git scopes
- WHEN analysis is requested
- THEN it fails without an outcome or receipt

### Requirement: Evidence-Backed Outcome

The system MUST return exactly one outcome: `update`, `create`, or `no-impact`. It MUST include candidate repository paths, rationale, evidence, and a confidence value for every successful outcome; candidate paths MAY be empty only for `no-impact`.

#### Scenario: Documentation requires update

- GIVEN a scoped change that affects existing documentation
- WHEN analysis succeeds
- THEN it returns `update` with candidate paths and evidence

#### Scenario: Documentation requires creation

- GIVEN a scoped change requiring documentation
- WHEN analysis succeeds
- THEN it returns `create` with candidate paths and evidence

#### Scenario: Documentation is unaffected

- GIVEN a scoped change with no documentation impact
- WHEN analysis succeeds
- THEN it returns `no-impact` with rationale, evidence, and no candidate paths

### Requirement: Read-Only Repository Boundary

The system MUST inspect only the selected repository and its Git-visible content. Analysis and receipt verification MUST NOT write, modify, stage, commit, apply, synchronize, or propose patches to repository documentation or other files.

#### Scenario: Analyze without mutation

- GIVEN a repository with a valid scope
- WHEN analysis completes
- THEN its tracked, staged, and worktree content remains unchanged

#### Scenario: Exclude external content

- GIVEN a request that references an external provider or another repository
- WHEN analysis is requested
- THEN it fails without accessing or changing that content

### Requirement: Canonical Receipt Verification

For a successful analysis, the system MUST provide a deterministic receipt bound to the resolved scope, analyzed content, outcome, evidence, documentation content identity, and receipt schema/tool versions. Verification MUST accept only an exactly matching successful analysis and MUST NOT invoke an LLM or mutate files.

#### Scenario: Verify a matching receipt

- GIVEN an unaltered receipt and matching repository content
- WHEN receipt verification is requested
- THEN verification succeeds deterministically

#### Scenario: Reject a stale receipt

- GIVEN a receipt whose bound scope or content differs
- WHEN receipt verification is requested
- THEN verification fails without producing a new receipt

### Requirement: Equivalent Invocation Contract

The CLI and MCP `document_change` operation MUST expose equivalent scope inputs, successful result fields, receipt-verification behavior, and failure classifications. Both MUST require explicit invocation; neither SHALL infer feature completion from Git state.

#### Scenario: Match CLI and MCP results

- GIVEN identical repository content and scope
- WHEN CLI and MCP analysis are requested
- THEN they return equivalent observable results

#### Scenario: Require explicit invocation

- GIVEN repository activity without an analysis request
- WHEN Git state changes or a merge occurs
- THEN no analysis is performed automatically

### Requirement: Failure and Unsupported-Case Reporting

The system MUST fail clearly when Git is unavailable, the target is not a repository, scope resolution fails, content cannot be read, or the request is unsupported. A failure MUST identify its classification and MUST NOT return an outcome or successful receipt.

#### Scenario: Report unavailable Git

- GIVEN Git cannot be used for the target repository
- WHEN analysis is requested
- THEN it returns a Git-unavailable failure classification

#### Scenario: Reject unsupported request

- GIVEN a request outside the supported repository-only operation
- WHEN it is submitted through CLI or MCP
- THEN it returns an unsupported-request failure classification
