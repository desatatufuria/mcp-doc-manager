# Compatibility and Workspace Lifecycle

Use top-level `install` for guided OpenCode onboarding. The `workspace` hierarchy remains the low-level repository lifecycle surface; `doctor` and `uninstall` remain compatibility aliases.

## Workspace commands and aliases

| Current command | Compatibility alias | Behavior |
| --- | --- | --- |
| `workspace install --target /repo` | none | Creates only owned repository-local state and initializes its ledger. |
| `workspace doctor --target /repo` | `doctor --target /repo` | Read-only validation. |
| `workspace uninstall --target /repo` | `uninstall --target /repo` | Removes only matching owned state. |
| `workspace status --target /repo --json` | none | Reports workspace and hook state. |

The `doctor` and `uninstall` aliases print a deprecation notice only in human output. With `--json`, they return the normal result without extra deprecation text. Top-level `install` is not an alias: it detects and configures OpenCode after repository initialization.

The target must be the exact Git root. Guided `install --json` requires `--yes` or `--dry-run` so it never prompts. `workspace install`, `workspace uninstall`, and compatibility aliases accept `--dry-run` to plan without writing; inspection commands (`status` and `doctor`) are read-only and do not accept `--dry-run`.

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
