#!/usr/bin/env sh
# Bootstrap verifies a signed manifest as data; it never evaluates downloaded text.
set -eu

release_base=${DOCMANAGER_RELEASE_BASE:-https://github.com/desatatufuria/mcp-doc-manager/releases}
version=${DOCMANAGER_VERSION:-latest}
install_dir=${DOCMANAGER_INSTALL_DIR:-"$HOME/.local/bin"}
os_name=${DOCMANAGER_OS:-$(uname -s | tr '[:upper:]' '[:lower:]')}
arch=${DOCMANAGER_ARCH:-$(uname -m)}
case "$os_name/$arch" in
  linux/x86_64) os_name=linux; arch=amd64 ;;
  linux/aarch64) os_name=linux; arch=arm64 ;;
  linux/amd64|linux/arm64) ;;
  darwin/x86_64) os_name=darwin; arch=amd64 ;;
  darwin/arm64|darwin/amd64) ;;
  *) printf '%s\n' "unsupported platform: $os_name/$arch" >&2; exit 1 ;;
esac
case "$release_base" in https://github.com/*|https://objects.githubusercontent.com/*) ;; *) printf '%s\n' 'release URL is not allowlisted HTTPS' >&2; exit 1 ;; esac
case "$version" in
  latest) manifest_base="$release_base/latest/download" ;;
  v*)
    case "$version" in *[!A-Za-z0-9._-]*) printf '%s\n' 'release version is invalid' >&2; exit 1 ;; esac
    manifest_base="$release_base/download/$version"
    ;;
  *) printf '%s\n' 'release version must be latest or a v-prefixed tag' >&2; exit 1 ;;
esac
command -v curl >/dev/null 2>&1 && command -v openssl >/dev/null 2>&1 && command -v python3 >/dev/null 2>&1 || { printf '%s\n' 'curl, openssl, and python3 are required' >&2; exit 1; }

tmp=$(mktemp -d "${TMPDIR:-/tmp}/docmanager-install.XXXXXX") || exit 1
trap 'rm -rf "$tmp"' EXIT HUP INT TERM
trust_file="$tmp/trust.json"
public_key="$tmp/public-key.pem"
cat > "$trust_file" <<'TRUST_JSON'
{"schema":1,"key_id":"docmanager-2026-01","public_key":"public-key.pem"}
TRUST_JSON
cat > "$public_key" <<'PUBLIC_KEY'
-----BEGIN PUBLIC KEY-----
MCowBQYDK2VwAyEAPZPPg6utUP67tINU9eniIPpe5yApudZmUejC+tsVuTI=
-----END PUBLIC KEY-----
PUBLIC_KEY
download() {
  destination=$1
  url=$2
  max_size=$3
  effective=$(curl --fail --silent --show-error --location --proto '=https' --proto-redir '=https' --max-filesize "$max_size" --output "$destination" --write-out '%{url_effective}' "$url") || return 1
  case "$effective" in https://github.com/*|https://objects.githubusercontent.com/*) ;; *) return 1 ;; esac
  test -s "$destination"
}
sha256() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  else
    shasum -a 256 "$1" | awk '{print $1}'
  fi
}

download "$tmp/manifest.json" "$manifest_base/manifest.json" 1048576 || { printf '%s\n' 'unable to fetch trusted manifest' >&2; exit 1; }
download "$tmp/manifest.sig" "$manifest_base/manifest.sig" 1048576 || { printf '%s\n' 'unable to fetch manifest signature' >&2; exit 1; }
openssl pkeyutl -verify -rawin -pubin -inkey "$public_key" -in "$tmp/manifest.json" -sigfile "$tmp/manifest.sig" >/dev/null 2>&1 || { printf '%s\n' 'manifest signature verification failed' >&2; exit 1; }
read_artifact=$(python3 - "$tmp/manifest.json" "$os_name" "$arch" "$trust_file" <<'PY'
import datetime, hashlib, json, sys
manifest = json.load(open(sys.argv[1], encoding="utf-8"))
if manifest.get("schema") != 1 or manifest.get("key_id") != json.load(open(sys.argv[4], encoding="utf-8")).get("key_id"):
    raise SystemExit(1)
now = datetime.datetime.now(datetime.timezone.utc)
if datetime.datetime.fromisoformat(manifest["expires"].replace("Z", "+00:00")) <= now:
    raise SystemExit(1)
for artifact in manifest.get("artifacts", []):
    if artifact.get("os") == sys.argv[2] and artifact.get("arch") == sys.argv[3]:
        if artifact.get("size", -1) < 0 or len(artifact.get("sha256", "")) != 64:
            raise SystemExit(1)
        print(artifact["url"])
        print(artifact["size"])
        print(artifact["sha256"])
        raise SystemExit(0)
raise SystemExit(1)
PY
)
artifact_url=$(printf '%s\n' "$read_artifact" | sed -n '1p')
artifact_size=$(printf '%s\n' "$read_artifact" | sed -n '2p')
artifact_sha=$(printf '%s\n' "$read_artifact" | sed -n '3p')
test -n "$artifact_url" && test -n "$artifact_size" && test -n "$artifact_sha" || { printf '%s\n' 'manifest does not authorize this platform' >&2; exit 1; }
case "$artifact_url" in https://github.com/*|https://objects.githubusercontent.com/*) ;; *) printf '%s\n' 'artifact URL is not allowlisted HTTPS' >&2; exit 1 ;; esac
download "$tmp/archive.tar.gz" "$artifact_url" "$artifact_size" || { printf '%s\n' 'unable to fetch authorized archive' >&2; exit 1; }
test "$(wc -c < "$tmp/archive.tar.gz" | tr -d ' ')" = "$artifact_size" && test "$(sha256 "$tmp/archive.tar.gz")" = "$artifact_sha" || { printf '%s\n' 'artifact digest verification failed' >&2; exit 1; }
tar -xzf "$tmp/archive.tar.gz" -C "$tmp" docmanager || { printf '%s\n' 'archive is invalid' >&2; exit 1; }
test -f "$tmp/docmanager" && test ! -L "$tmp/docmanager" || { printf '%s\n' 'archive binary is unsafe' >&2; exit 1; }
mkdir -p "$install_dir" && test ! -L "$install_dir" || { printf '%s\n' 'install directory is unsafe' >&2; exit 1; }
install_tmp=$(mktemp "$install_dir/.docmanager.XXXXXX") || exit 1
cp "$tmp/docmanager" "$install_tmp"
chmod 755 "$install_tmp"
mv -f "$install_tmp" "$install_dir/docmanager"
printf '%s\n' "installed docmanager to $install_dir/docmanager"
