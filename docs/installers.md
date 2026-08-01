# Install Docmanager Safely

The standalone signed bootstrap installs only verified Linux or macOS archives. It does not require a checkout and does not configure agents, Git hooks, or your shell.

## Quick path

```sh
curl -fsSL https://raw.githubusercontent.com/desatatufuria/mcp-doc-manager/main/scripts/install.sh | sh
"$HOME/.local/bin/docmanager" workspace status --target /absolute/repository --json
```

The tokenless command is available after the GitHub repository is public and at least one release exists. `install.sh` embeds the trusted key ID and public key; it never fetches its trust root. By default it uses GitHub's latest-release download route. For a reproducible install, pin an explicit published tag:

```sh
curl -fsSL https://raw.githubusercontent.com/desatatufuria/mcp-doc-manager/main/scripts/install.sh | DOCMANAGER_VERSION=vX.Y.Z sh
```

The script verifies the signed manifest, selected archive size, and SHA-256 before atomically placing `docmanager` in `$HOME/.local/bin` by default. It never evaluates downloaded shell text.

## Prerequisites and supported targets

| Item | Requirement |
| --- | --- |
| Platforms | Linux or macOS on amd64 or arm64. |
| Excluded platform | Windows distribution is explicitly unsupported; no Windows archive is published by this installer. |
| Commands | `curl`, `openssl`, `python3`, `tar`, and `sha256sum` or `shasum`. |
| Install location | `$HOME/.local/bin/docmanager` by default; set `DOCMANAGER_INSTALL_DIR` to choose another regular, non-symlinked directory. |
| Release version | The default installs the latest release; set `DOCMANAGER_VERSION=vX.Y.Z` to pin an explicit published tag. |
| PATH | The script does not edit `PATH` or shell profiles. Use the full path or update your shell configuration manually. |

The source tree also supports a local build when a published release is unavailable:

```sh
CGO_ENABLED=0 go build -trimpath -buildvcs=false -o "$HOME/.local/bin/docmanager" ./cmd/docmanager
```

## Trust and release publishing

Trust metadata is committed at `assets/release/trust.json`. The current trusted key ID is `docmanager-2026-01`; its matching public key is `assets/release/public-key.pem`. A release manifest must use that key ID, have a valid signature, be unexpired, and authorize the exact platform archive.

Release maintainers configure the GitHub Actions secret named `DOCMANAGER_RELEASE_SIGNING_KEY` with the private signing material corresponding to the committed public key. Never put that secret, a private-key file, or its value in the repository, shell history, issues, or documentation.

The bootstrap has no command-line upgrade, status, doctor, or rollback interface. Release lifecycle services exist in the application layer, but CLI composition for those operations is not exposed yet. Re-run the bootstrap only after confirming the target location and keep a known-good executable for manual rollback; see [Recovery](recovery.md).

## Current availability versus future integration

`workspace`, `release`, and `agent` are headless design boundaries intended to be consumable by a future TUI. No TUI is present. Gentle AI/community-tool orchestration is also future work: Docmanager has no runtime dependency on it and it owns no Gentle AI configuration.

## Verify an installation

Use ordinary product commands after installation:

```sh
DOCMANAGER="$HOME/.local/bin/docmanager"
"$DOCMANAGER" document-change --repo /absolute/repository --scope staged
"$DOCMANAGER" workspace doctor --target /absolute/repository --json
```

`workspace doctor` validates owned repository-local state; it does not validate a downloaded release. The current release CLI hierarchy must not be treated as an operational upgrade tool.

## Checklist

- [ ] The platform is Linux/macOS amd64/arm64.
- [ ] The installer prerequisites are on `PATH`.
- [ ] The default or chosen install directory is appropriate and not a symlink.
- [ ] The executable is invoked by full path until `PATH` is deliberately updated.
- [ ] A known-good executable or trusted offline copy is retained before replacement.
