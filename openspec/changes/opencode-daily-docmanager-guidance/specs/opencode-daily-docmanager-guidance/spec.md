# OpenCode Daily DocManager Guidance

### Requirement: Additive Owned Configuration

System MUST install versioned guidance in `.docmanager/guidance/` and add its absolute path through additive `instructions`. It MUST preserve user instructions and unrelated config.

#### Scenario: Fresh configuration
- GIVEN supported config without owned entries
- WHEN configuration runs
- THEN it adds owned MCP entry and instruction reference

#### Scenario: Unsupported or conflict
- GIVEN unknown config or a conflicting identity
- WHEN configuration is requested
- THEN it fails before mutation

### Requirement: Ownership-Safe Lifecycle

Ownership requires exact matching MCP entry, instruction reference, and guidance identity. Reconfiguration MUST be a no-op; drift MUST be rejected; upgrades MUST replace older owned guidance only; unconfigure MUST remove exact owned entries and retain `.docmanager` state.

#### Scenario: No-op and upgrade
- GIVEN current owned config, then older owned guidance
- WHEN configure is requested
- THEN it is a no-op, then upgrades owned guidance only

#### Scenario: Drift and unconfigure
- GIVEN moved, edited, missing, or substituted owned guidance
- WHEN configure or unconfigure runs
- THEN it rejects without mutation; exact ownership permits removal

### Requirement: Capability and Lifecycle Observability

Pinned fixture MUST validate additive shape. Status/doctor MUST report support, owned entries, guidance version/readability, and bounded binary probe; MUST report unsupported, missing, unreadable, or drifted states without repair.

#### Scenario: Healthy inspection
- GIVEN supported config and readable guidance
- WHEN status or doctor runs
- THEN it reports all required healthy fields

#### Scenario: Degraded inspection
- GIVEN unsupported, unreadable, moved, or missing guidance
- WHEN status or doctor runs
- THEN it reports the precise state without mutation

### Requirement: Versioned Bounded Guidance

Guidance MUST be readable, versioned, and review-first. It MUST NOT mutate user instructions, `AGENTS.md`, documentation, workspace/hook behavior, auto-enable hooks, manage MCP lifecycle, other agents, generic composition, release CLI work, or historical installer-plan evidence.

#### Scenario: Guidance consumption
- GIVEN OpenCode loads the owned reference
- WHEN documentation-impact review is appropriate
- THEN it provides the versioned workflow without file modification

### Requirement: Explicit Read-Only Checkpoints

Guidance and MCP metadata MUST require one explicit `worktree`, `staged`, or two-dot `range`. Worktree MUST consider unstaged changes; staged MUST consider index changes; range MUST use `base..head`. Descriptions and schemas MUST state scope, one-call use, and bounded purpose.

#### Scenario: Explicit range
- GIVEN `base..head` is supplied
- WHEN analysis is requested
- THEN it uses one checkpoint with that recorded scope

#### Scenario: Ambiguous scope
- GIVEN absent scope or a non-two-dot range
- WHEN analysis is considered
- THEN it makes no tool call and requests explicit selection

### Requirement: Negative Triggers and Human Review

Guidance and MCP metadata MUST prohibit calls for routine coding, unrelated questions, inferred scope, automatic editing, or unavailable evidence. Tools MUST remain read-only; human review of `document_change` MUST precede receipt verification.

#### Scenario: Negative trigger
- GIVEN routine coding without explicit documentation-impact review
- WHEN OpenCode follows guidance
- THEN it makes no DocManager call

#### Scenario: Review boundary
- GIVEN `document_change` completed
- WHEN verification is considered
- THEN it waits for explicit human review

### Requirement: Receipt Eligibility and Re-analysis

At most one `verify_receipt` MUST follow human review when recorded scope and considered documentation bytes are unchanged. Changed scope or considered bytes MUST invalidate eligibility and require new explicit analysis.

#### Scenario: Eligible receipt
- GIVEN reviewed output with unchanged scope and bytes
- WHEN verification is requested once
- THEN it verifies the receipt

#### Scenario: Invalidated receipt
- GIVEN scope or considered bytes changed after analysis
- WHEN verification is requested
- THEN it refuses until re-analysis completes

### Requirement: End-to-End Acceptance

Acceptance MUST cover configuration, invocation/non-invocation, no-op, drift, upgrade, status/doctor, exact unconfigure, reviewed verification, and invalidation/re-analysis.

#### Scenario: Managed workflow
- GIVEN supported configuration and documentation-impact staged changes
- WHEN OpenCode configures, analyzes, receives review, and verifies unchanged output
- THEN it succeeds while unrelated configuration remains unchanged
