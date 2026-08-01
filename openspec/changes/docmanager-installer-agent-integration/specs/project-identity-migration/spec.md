# project-identity-migration Specification

## Purpose

Define the canonical public identity of Docmanager without changing its MCP or ledger behavior.

## Requirements

### Requirement: Canonical Module Identity

The system MUST publish and report `github.com/desatatufuria/mcp-doc-manager` as its canonical Go module and import identity. New public references MUST use that identity.

#### Scenario: Canonical consumer reference

- GIVEN a consumer obtains a current Docmanager reference
- WHEN the reference identifies the module or import path
- THEN it SHALL identify `github.com/desatatufuria/mcp-doc-manager`

#### Scenario: Legacy identity is encountered

- GIVEN an input names the previous module identity
- WHEN Docmanager validates a canonical identity request
- THEN it MUST reject or diagnose the mismatch without claiming it is canonical

### Requirement: Behavioral Preservation During Migration

The identity migration MUST NOT change existing read-only MCP tool contracts, receipt behavior, or the repository-local ledger location.

#### Scenario: Existing MCP client

- GIVEN a client invokes an existing read-only MCP tool
- WHEN it uses a release with the canonical identity
- THEN the tool's observable result contract SHALL remain compatible

#### Scenario: Existing repository ledger

- GIVEN a repository contains its existing local ledger
- WHEN Docmanager runs after the migration
- THEN it MUST continue to use that ledger location
