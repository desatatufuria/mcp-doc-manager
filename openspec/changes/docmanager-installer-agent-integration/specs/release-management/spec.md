# release-management Specification

## Purpose

Define trusted Docmanager distribution and recovery on supported platforms.

## Requirements

### Requirement: Verified Supported Distribution

The system MUST install or upgrade only an artifact authorized by a valid, unexpired signed manifest from a trusted key ID. The supported matrix SHALL be Linux and macOS on amd64 and arm64; Windows distribution MUST be rejected as unsupported.

#### Scenario: Verified supported install

- GIVEN a valid trusted manifest for Linux arm64
- WHEN an install is requested for Linux arm64
- THEN the system MUST install the manifest-authorized artifact

#### Scenario: Unsupported platform

- GIVEN a Windows target
- WHEN distribution is requested
- THEN the system MUST return a stable unsupported-platform error and make no install

#### Scenario: Expired or invalid manifest

- GIVEN a manifest that is expired or fails signature verification
- WHEN install or upgrade is requested
- THEN the system MUST refuse the operation without replacing the current binary

### Requirement: Trust Rotation, Compromise, and Recovery

The system MUST accept a rotated trusted key only during an explicitly authorized overlap, MUST reject revoked or compromised keys, and MUST provide offline/manual recovery instructions when trust cannot be established.

#### Scenario: Authorized key rotation

- GIVEN an unexpired manifest signed by an overlapping replacement key
- WHEN verification occurs
- THEN the system SHALL accept the manifest

#### Scenario: Compromised key

- GIVEN a manifest signed by a revoked key ID
- WHEN verification occurs
- THEN the system MUST reject it and report recovery guidance

### Requirement: Safe Release Lifecycle

The system MUST expose install, upgrade, rollback, status, and doctor behavior. It MUST preserve a recoverable prior owned installation until a replacement is verified and healthy.

#### Scenario: Failed upgrade recovery

- GIVEN an owned healthy installation and a replacement that fails health validation
- WHEN upgrade runs
- THEN the system MUST restore the prior installation

#### Scenario: Status inspection

- GIVEN no supported installation is present
- WHEN status is requested
- THEN the system MUST report the absence without modifying state
