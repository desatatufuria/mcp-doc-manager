# Safe Recovery and Troubleshooting

Stop on trust, ownership, drift, or probe failures. These failures are safeguards; do not bypass them with symlinks, forced deletion, or destructive cleanup commands.

## Release recovery

| Situation | Safe response |
| --- | --- |
| Bootstrap cannot fetch the manifest | Check network/proxy policy and the allowlisted GitHub release URL. The script exits before installation. Use a manually verified offline binary if recovery cannot wait. |
| Signature, key ID, expiry, size, or digest fails | Do not install the archive. Obtain a fresh signed manifest or a separately verified trusted binary. |
| Wrong platform | Do not force it. Only Linux/macOS amd64/arm64 are supported by distribution; Windows distribution is excluded. |
| Replacement is unwanted | Restore the known-good executable you retained, after verifying its provenance and executable path. The bootstrap itself does not expose rollback. |

An offline/manual recovery copy must be verified out of band against trusted release metadata and checksum before use. It is not permission to skip manifest trust checks on a new download.

### Key compromise and rotation

The committed trust metadata currently names key ID `docmanager-2026-01` and public key `assets/release/public-key.pem`. If that key is suspected compromised, stop using manifests signed by it, obtain updated trust metadata through a separately trusted channel, and wait for a release signed by an authorized replacement key. Rotation is valid only during an explicit authorized overlap; revoked keys must be rejected. Never solve a compromise by editing local trust data to accept an unknown key.

Maintainers rotate by publishing reviewed trust metadata and configuring `DOCMANAGER_RELEASE_SIGNING_KEY` only in GitHub Actions. Private signing material must remain outside the repository.

## Workspace and agent recovery

| Symptom | Safe response |
| --- | --- |
| `drift`, ownership, malformed configuration, or unsafe route | Inspect the file and preserve a copy. Restore the expected owned content only if you can prove ownership; otherwise leave it unchanged. |
| Lock error | Wait for the known operation to finish and retry. Do not remove a lock you cannot attribute. |
| Managed hook is drifted | Inspect it manually. Do not use uninstall as a forced hook remover. |
| Agent configuration is unexpected | Do not use a guessed CLI command. Current public CLI wiring does not configure adapters; see [Agents](agents.md). |
| Repository-local lifecycle state is corrupt | Run `docmanager workspace doctor --target /absolute/repository --json`; remove only state that is proven owned through the documented workspace lifecycle. |

## Native CI evidence boundary

Linux native package smoke passed locally. Linux/amd64, macOS/arm64, and Windows/amd64 CGo-free cross-builds passed locally. Actual native macOS and Windows smoke reruns are still pending until GitHub Actions confirms them; cross-build success is not native-runtime evidence.

## Before retrying

- [ ] Keep the failing manifest, checksum, error output, or configuration copy for diagnosis.
- [ ] Confirm the target is a real path, not a symlink.
- [ ] Verify that no other Docmanager operation owns the lock.
- [ ] Prefer read-only status/doctor and inspection before any owned rollback.
- [ ] Never delete a user-managed agent entry, Git hook, or key to make a command pass.
