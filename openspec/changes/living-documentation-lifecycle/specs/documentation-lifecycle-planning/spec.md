# Documentation Lifecycle Planning Specification

## Purpose

Define approval-gated documentation plans, policy, and action boundaries from confirmed radiography.

## Requirements

### Requirement: Approval-Gated Plan

The system MUST propose, rather than author or persist, an information architecture and coherent document batch from radiography. It MUST NOT authorize visible-document authoring before plan approval.

#### Scenario: Approve a proposed plan

- GIVEN confirmed radiography and a proposed document batch
- WHEN an authorized user approves the plan
- THEN the batch becomes the approved authoring boundary

#### Scenario: Refuse a proposed plan

- GIVEN a proposed plan has not been approved or is refused
- WHEN a caller requests document authoring
- THEN the system denies authorization and preserves the plan as unapproved

### Requirement: Discovered Storage and Existing Documentation

The system MUST include the confirmed visible storage, audience, language, ownership candidates, and treatment for existing documents in a plan. It MUST import existing documents in place and MUST NOT duplicate, relocate, or replace them by default.

#### Scenario: Plan around existing documents

- GIVEN confirmed existing Markdown documentation in a discovered location
- WHEN a plan is approved
- THEN it includes those documents as in-place inventory and states their planned treatment

#### Scenario: Reject a fixed-name assumption

- GIVEN no repository evidence or confirmation selects a document name or location
- WHEN a plan is proposed
- THEN it does not prescribe a fixed filename or fabricated storage location

### Requirement: Project-Selected Automation Policy

The system MUST support `approval-required` and `automatic-after-approved-plan` policies selected by the project. Approval-required mode MUST require approval for each coherent feature or change batch; automatic mode MAY authorize create or update only inside the approved plan. Policy changes MUST require approval and produce an audit record.

#### Scenario: Approve a coherent batch

- GIVEN the project uses approval-required policy
- WHEN a feature proposes multiple related documentation changes
- THEN one explicit approval accepts or refuses the coherent batch

#### Scenario: Apply automatic policy boundaries

- GIVEN automatic-after-approved-plan policy and an approved plan
- WHEN a proposed content update is inside that plan
- THEN it is eligible for automatic authorization

#### Scenario: Change policy

- GIVEN an established lifecycle policy
- WHEN an authorized user approves a policy change
- THEN the new policy and its approval are recorded for audit

### Requirement: Structural and Destructive Safeguards

The system MUST require explicit approval for moves, renames, archival, deletion, and any structural or destructive action regardless of automation policy.

#### Scenario: Block unapproved structural action

- GIVEN automatic-after-approved-plan policy
- WHEN a caller proposes moving or deleting a document
- THEN the system requires explicit approval before authorization
