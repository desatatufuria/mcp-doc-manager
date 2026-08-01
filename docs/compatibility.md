# Compatibility and Workspace Lifecycle

Use the `workspace` hierarchy for repository-local lifecycle state. It is the current CLI surface; legacy aliases remain only for one major-version compatibility window.

## Workspace commands and aliases

| Current command | Compatibility alias | Behavior |
| --- | --- | --- |
| `workspace install --target /repo` | `install --target /repo` | Creates only owned repository-local state. |
| `workspace doctor --target /repo` | `doctor --target /repo` | Read-only validation. |
| `workspace uninstall --target /repo` | `uninstall --target /repo` | Removes only matching owned state. |
| `workspace status --target /repo --json` | none | Reports workspace and hook state. |

Aliases print a deprecation notice only in human output. With `--json`, they return the normal result without extra deprecation text.

The target must be the exact Git root. `workspace install`, `workspace uninstall`, and aliases accept `--dry-run` to plan without writing; all resource commands accept `--json` for machine-readable results. Inspection commands (`status` and `doctor`) are read-only and do not accept `--dry-run`.

## Hooks require explicit consent

```sh
docmanager workspace install --target /absolute/repository --json
docmanager workspace install --target /absolute/repository --enable-hook --json
```

The first command leaves the hook absent. Only the second explicitly requests the managed pre-push hook; status distinguishes `absent`, `opted-in`, and `drifted`. A changed hook is never replaced or removed automatically.

## Ownership, backups, and rollback boundary

Workspace mutation uses external XDG state, locks, backup digests/modes, and atomic replacement. It refuses symlinks, route escapes, unowned state, and drift. `workspace uninstall` removes owned workspace artifacts and the managed hook only; it does not alter documentation, history, remotes, agent configuration, or unrelated hooks.

For the older repository-local lifecycle guidance and receipt ledger behavior, keep using the commands and safeguards in [ONBOARDING.md](../ONBOARDING.md). The workspace hierarchy is the preferred spelling for its install/doctor/uninstall lifecycle.

## Versioned agent compatibility

Agent support is configuration-shape compatibility, not a general claim about every version of an agent. See [Agents](agents.md#fixture-policy) before relying on a new agent release or manually edited configuration.
