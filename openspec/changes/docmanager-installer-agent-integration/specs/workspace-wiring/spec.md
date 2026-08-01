# workspace-wiring Specification

## Purpose

Define workspace-facing lifecycle commands, compatibility aliases, and explicit hook consent.

## Requirements

### Requirement: Workspace Command Hierarchy

The system MUST expose workspace lifecycle operations under the `workspace` hierarchy and preserve `install --target`, `doctor --target`, and `uninstall --target` as deprecated human-output aliases for one major version.

#### Scenario: Current workspace command

- GIVEN a caller requests a workspace lifecycle action
- WHEN it invokes the corresponding `workspace` command
- THEN the system MUST perform the documented equivalent lifecycle action

#### Scenario: Compatibility alias period

- GIVEN the release is within the next major-version compatibility window
- WHEN a caller invokes `install --target`
- THEN the system SHALL complete the equivalent action and identify the alias as deprecated in human output

#### Scenario: Machine-readable alias request

- GIVEN a caller requests JSON output through a compatibility alias
- WHEN the alias executes
- THEN the system MUST return the standard JSON result without human-only deprecation text

### Requirement: Opt-In Repository Hooks

The system MUST NOT create, enable, or alter repository hooks unless the caller explicitly opts in. Hook status MUST distinguish absent, opted-in, and drifted state.

#### Scenario: Default workspace setup

- GIVEN a repository has no explicit hook opt-in
- WHEN workspace setup runs
- THEN the system MUST NOT create or enable a hook

#### Scenario: Explicit hook opt-in

- GIVEN a caller explicitly requests hook installation
- WHEN workspace setup succeeds
- THEN the system MUST report the hook as opted in

#### Scenario: Drifted hook

- GIVEN an opted-in hook has been externally changed
- WHEN workspace status or removal is requested
- THEN the system MUST report drift and refuse destructive replacement or removal
