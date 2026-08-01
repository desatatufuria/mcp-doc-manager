# agent-integration Specification

## Purpose

Define safe managed Docmanager MCP and guidance lifecycle for supported agents.

## Requirements

### Requirement: Agent State Detection

The system MUST detect and report `installed`, `supported`, `configured`, and `healthy` independently for OpenCode, Codex, Claude Code, GitHub Copilot, and Pi. It MUST reject unknown or unsupported agent/configuration shapes safely.

#### Scenario: Healthy configured agent

- GIVEN OpenCode is installed and has a valid managed Docmanager entry
- WHEN agent status is requested
- THEN the system MUST report it as installed, supported, configured, and healthy

#### Scenario: Malformed agent configuration

- GIVEN a supported agent has malformed configuration
- WHEN status or configuration is requested
- THEN the system MUST return a stable error and preserve the file

### Requirement: Managed MCP and Guidance Lifecycle

The system MUST configure a supported agent with a managed MCP entry and managed guidance/skill, and MUST unconfigure only entries it owns. It MUST preserve existing MCP tools, receipt behavior, and repository-local ledger semantics.

#### Scenario: Configure an unconfigured supported agent

- GIVEN Codex is installed, supported, and its configuration is valid
- WHEN managed configuration is requested
- THEN the system MUST add its managed MCP entry and guidance/skill

#### Scenario: Unconfigure owned entries

- GIVEN Claude Code contains managed Docmanager entries and unrelated user entries
- WHEN unconfigure is requested
- THEN the system MUST remove only the managed entries and retain unrelated entries

#### Scenario: Externally changed managed state

- GIVEN a managed entry no longer matches its recorded ownership state
- WHEN configure or unconfigure is requested
- THEN the system MUST refuse destructive change and report drift

### Requirement: Safe Configuration Mutation

The system MUST serialize concurrent mutations, create recoverable backups before changing owned configuration, and make each accepted configuration change atomically visible.

#### Scenario: Concurrent mutation

- GIVEN another Docmanager mutation holds the target configuration lock
- WHEN a second mutation is requested
- THEN the second request MUST return a stable lock error without changing configuration
