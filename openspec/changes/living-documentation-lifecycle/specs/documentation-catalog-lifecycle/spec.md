# Documentation Catalog Lifecycle Specification

## Purpose

Define internal lifecycle records, freshness assessment, and verification of caller-authored documentation.

## Requirements

### Requirement: Complete Catalog and Provenance

The system MUST maintain internal catalog entries with path, purpose, audience, owner, related areas, lifecycle state, evidence, last verification, and pending actions. It MUST record internal provenance for lifecycle decisions and MUST NOT add visible provenance markers to documents.

#### Scenario: Record an imported document

- GIVEN an approved plan imports an existing document in place
- WHEN lifecycle state is persisted
- THEN its catalog entry contains every required field and internal provenance

#### Scenario: Detect stale catalog evidence

- GIVEN a catalog entry's evidence or verification basis no longer matches repository evidence
- WHEN the entry is assessed
- THEN the system marks it stale or uncertain rather than claiming freshness

### Requirement: Impact and Freshness Audits

The system MUST perform change-triggered and on-demand audits. It MUST report evidence and rationale for `update`, `review`, `orphan`, `conflict`, or `no-action` outcomes; it MUST NOT represent probabilistic mapping as a correctness verdict.

#### Scenario: Audit a related repository change

- GIVEN changed repository evidence maps to a catalog entry
- WHEN a change-triggered audit runs
- THEN it reports the candidate outcome with evidence and rationale

#### Scenario: Audit without a change

- GIVEN a user requests an on-demand audit and evidence conflicts
- WHEN the audit runs
- THEN it reports conflict or uncertainty and the evidence that caused it

### Requirement: Protected Orphan Handling

The system MUST retain evidence-marked orphan records until an explicit approval authorizes archival or removal. It MUST NOT silently remove an orphan.

#### Scenario: Identify an orphan

- GIVEN an audit finds an evidence-supported orphan
- WHEN the audit completes
- THEN it records the orphan and a pending action without removing the document

#### Scenario: Approve orphan removal

- GIVEN an orphan has recorded evidence
- WHEN an authorized user approves its archival or removal
- THEN the approved structural action may proceed and is recorded

### Requirement: Caller-Authored Outcome Verification

The calling agent MUST author visible-document changes through normal repository edits. The system MUST validate reported outcomes against the approved plan and record verification; it MUST reject mismatched paths, actions, or evidence.

#### Scenario: Verify an approved caller edit

- GIVEN a caller reports an edit within an approved plan
- WHEN verification finds matching paths and evidence
- THEN the system records a verified lifecycle outcome

#### Scenario: Reject a verification mismatch

- GIVEN a caller reports an edit outside the approved plan or with changed evidence
- WHEN verification runs
- THEN the system rejects verification and records the mismatch without accepting the outcome

### Requirement: Legacy Idempotency Migration Integrity

The system MUST preserve all historical v1 provenance and idempotency data for audit. When v1 evidence cannot reliably reconstruct a provenance-to-idempotency relation, migration MUST mark the legacy key replay unavailable and MUST NOT invent a mapping. A request using that key MUST return a stable typed refusal and MUST NOT execute, replay an unrelated result, overwrite audit history, or create a duplicate successful action. New v2 records MUST retain exact idempotent replay. Migration failure MUST be atomic and leave v1 unchanged.

#### Scenario: Invalidate an ambiguous multi-record legacy key

- GIVEN one v1 idempotency key has insufficient evidence to relate it to multiple provenance records
- WHEN the v1 data migrates to v2
- THEN all historical records are preserved and that key is marked replay unavailable

#### Scenario: Refuse a migrated legacy key

- GIVEN a legacy key is marked replay unavailable after migration
- WHEN a request reuses that key
- THEN the system returns the stable typed refusal without executing or changing audit history

#### Scenario: Replay a new v2 request exactly

- GIVEN a successful v2 request and its idempotency key
- WHEN an identical request reuses that key
- THEN the system replays its exact recorded result without a duplicate action

#### Scenario: Roll back a failed legacy migration

- GIVEN v1 lifecycle records and a migration failure
- WHEN the migration transaction ends
- THEN v1 records remain unchanged and no partial v2 state exists
