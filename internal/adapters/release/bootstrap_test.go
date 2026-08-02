package release

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestReleaseBootstrapAssetsDefineSignedSupportedArchives(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	trustPath := filepath.Join(root, "assets", "release", "trust.json")
	trustRaw, err := os.ReadFile(trustPath)
	if err != nil {
		t.Fatalf("read public trust metadata: %v", err)
	}
	var trust struct {
		Schema    int    `json:"schema"`
		KeyID     string `json:"key_id"`
		PublicKey string `json:"public_key"`
	}
	if err := json.Unmarshal(trustRaw, &trust); err != nil {
		t.Fatalf("decode public trust metadata: %v", err)
	}
	if trust.Schema != 1 || trust.KeyID == "" || trust.PublicKey == "" {
		t.Fatalf("invalid deterministic public trust binding: %+v", trust)
	}
	if key, err := os.ReadFile(filepath.Join(filepath.Dir(trustPath), trust.PublicKey)); err != nil || !strings.Contains(string(key), "BEGIN PUBLIC KEY") {
		t.Fatalf("public key metadata is not a PEM public key: %v", err)
	}

	for _, target := range []struct{ os, arch string }{{"linux", "amd64"}, {"linux", "arm64"}, {"darwin", "amd64"}, {"darwin", "arm64"}} {
		workflow, err := os.ReadFile(filepath.Join(root, ".github", "workflows", "release.yml"))
		if err != nil || !strings.Contains(string(workflow), "os: "+target.os+", arch: "+target.arch) || !strings.Contains(string(workflow), "docmanager_${version}_${GOOS}_${GOARCH}.tar.gz") {
			t.Fatalf("release workflow does not package docmanager_<version>_%s_%s.tar.gz: %v", target.os, target.arch, err)
		}
	}

	script, err := os.ReadFile(filepath.Join(root, "scripts", "install.sh"))
	if err != nil {
		t.Fatalf("read bootstrap: %v", err)
	}
	for _, required := range []string{"set -eu", "openssl pkeyutl -verify", "--proto-redir '=https'", "mktemp -d", "mv -f", "shasum -a 256", "eval"} {
		if !strings.Contains(string(script), required) {
			t.Fatalf("bootstrap missing required signed-install behavior %q", required)
		}
	}
	if strings.Contains(string(script), "eval ") || strings.Contains(string(script), "source ") || strings.Contains(string(script), ". ") {
		t.Fatal("bootstrap evaluates downloaded or interpolated content")
	}
	if strings.Contains(string(script), "sudo") || strings.Contains(string(script), ".profile") || strings.Contains(string(script), ".zshrc") {
		t.Fatal("bootstrap escalates privileges or edits a shell profile")
	}
}

func TestReleaseWorkflowPublishesAllAssetsAndFailsClosed(t *testing.T) {
	workflow, err := os.ReadFile(filepath.Join("..", "..", "..", ".github", "workflows", "release.yml"))
	if err != nil {
		t.Fatalf("read release workflow: %v", err)
	}

	publication := `      - uses: softprops/action-gh-release@v2
        with:
          files: |
            out/*.tar.gz
            manifest.json
            manifest.sig
          fail_on_unmatched_files: true`
	if !strings.Contains(string(workflow), publication) {
		t.Fatal("release publication must use newline-delimited asset patterns and fail on unmatched files")
	}
	if strings.Contains(string(workflow), `files: 'out/*.tar.gz\nmanifest.json\nmanifest.sig'`) {
		t.Fatal("release publication uses one escaped-newline asset pattern")
	}
}

func TestReleaseWorkflowInjectsExactTagVersion(t *testing.T) {
	workflow, err := os.ReadFile(filepath.Join("..", "..", "..", ".github", "workflows", "release.yml"))
	if err != nil {
		t.Fatalf("read release workflow: %v", err)
	}

	build := `go build -trimpath -buildvcs=false -ldflags="-s -w -X 'main.version=${version}'" -o docmanager ./cmd/docmanager`
	if !strings.Contains(string(workflow), build) {
		t.Fatal("release build must inject the exact tag into main.version while retaining deterministic and stripping flags")
	}
}

func TestReleaseRepositoryTracksNoPrivateSigningMaterial(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	tracked, err := exec.Command("git", "-C", root, "ls-files", "-z").Output()
	if err != nil {
		t.Fatalf("list tracked files: %v", err)
	}
	for _, path := range strings.Split(string(tracked), "\x00") {
		name := strings.ToLower(filepath.Base(path))
		if strings.Contains(name, "private") || strings.Contains(name, "signing-key") || strings.HasSuffix(name, ".key") {
			t.Fatalf("tracked private signing material detected: %s", path)
		}
	}

	privatePath := filepath.Join("assets", "release", "private-key.pem")
	if err := exec.Command("git", "-C", root, "check-ignore", "--quiet", privatePath).Run(); err != nil {
		t.Fatalf("private signing key path is not ignored: %v", err)
	}
	if err := exec.Command("git", "-C", root, "ls-files", "--error-unmatch", privatePath).Run(); err == nil {
		t.Fatal("private signing key path is tracked")
	}
}

func TestBootstrapEmbeddedTrustMatchesReleaseAssets(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	script, err := os.ReadFile(filepath.Join(root, "scripts", "install.sh"))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name   string
		asset  string
		marker string
	}{
		{name: "metadata", asset: "trust.json", marker: "TRUST_JSON"},
		{name: "public key", asset: "public-key.pem", marker: "PUBLIC_KEY"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			asset, err := os.ReadFile(filepath.Join(root, "assets", "release", tc.asset))
			if err != nil {
				t.Fatal(err)
			}
			if embedded := heredoc(t, string(script), tc.marker); embedded != string(asset) {
				t.Fatalf("embedded %s differs from assets/release/%s", tc.name, tc.asset)
			}
		})
	}
	if strings.Contains(string(script), "$0") || strings.Contains(string(script), "dirname") || strings.Contains(string(script), "DOCMANAGER_TRUST_DIR") {
		t.Fatal("standalone bootstrap inspects its invocation path or adjacent trust assets")
	}
}

func TestBootstrapFromStdinUsesReleaseURLsAndFailsClosed(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	script, err := os.ReadFile(filepath.Join(root, "scripts", "install.sh"))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name          string
		version       string
		effectiveBase string
		wantFailure   string
		wantURLs      []string
	}{
		{name: "latest", effectiveBase: "https://release-assets.githubusercontent.com/github-production-release-asset", wantFailure: "manifest signature verification failed", wantURLs: []string{
			"https://github.com/desatatufuria/mcp-doc-manager/releases/latest/download/manifest.json",
			"https://github.com/desatatufuria/mcp-doc-manager/releases/latest/download/manifest.sig",
		}},
		{name: "explicit version", version: "v1.2.3", effectiveBase: "https://release-assets.githubusercontent.com/github-production-release-asset", wantFailure: "manifest signature verification failed", wantURLs: []string{
			"https://github.com/desatatufuria/mcp-doc-manager/releases/download/v1.2.3/manifest.json",
			"https://github.com/desatatufuria/mcp-doc-manager/releases/download/v1.2.3/manifest.sig",
		}},
		{name: "untrusted effective URL", effectiveBase: "https://release-assets.githubusercontent.com.example.com/github-production-release-asset", wantFailure: "unable to fetch trusted manifest", wantURLs: []string{
			"https://github.com/desatatufuria/mcp-doc-manager/releases/latest/download/manifest.json",
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tmp := t.TempDir()
			bin := filepath.Join(tmp, "bin")
			if err := os.Mkdir(bin, 0o700); err != nil {
				t.Fatal(err)
			}
			fakeCurl := "#!/bin/sh\noutput=\nurl=\nwhile [ \"$#\" -gt 0 ]; do\n  case \"$1\" in --output) shift; output=$1 ;; esac\n  url=$1\n  shift\ndone\nprintf x > \"$output\"\nprintf '%s\\n' \"$url\" >> \"$DOCMANAGER_TEST_URL_LOG\"\nprintf '%s/%s' \"$DOCMANAGER_TEST_EFFECTIVE_BASE\" \"${url##*/}\"\n"
			if err := os.WriteFile(filepath.Join(bin, "curl"), []byte(fakeCurl), 0o700); err != nil {
				t.Fatal(err)
			}
			installDir := filepath.Join(tmp, "install")
			urlLog := filepath.Join(tmp, "urls")
			command := exec.Command("sh", "-s")
			command.Dir = tmp
			command.Stdin = strings.NewReader(string(script))
			command.Env = append(withoutDocmanagerEnv(os.Environ()),
				"PATH="+bin+":"+os.Getenv("PATH"),
				"DOCMANAGER_INSTALL_DIR="+installDir,
				"DOCMANAGER_OS=linux",
				"DOCMANAGER_ARCH=amd64",
				"DOCMANAGER_TEST_URL_LOG="+urlLog,
				"DOCMANAGER_TEST_EFFECTIVE_BASE="+tc.effectiveBase,
			)
			if tc.version != "" {
				command.Env = append(command.Env, "DOCMANAGER_VERSION="+tc.version)
			}
			output, err := command.CombinedOutput()
			if err == nil || !strings.Contains(string(output), tc.wantFailure) {
				t.Fatalf("bootstrap failure = %v, output=%q", err, output)
			}
			requested, err := os.ReadFile(urlLog)
			gotURLs := strings.Split(strings.TrimSpace(string(requested)), "\n")
			if err != nil || strings.Join(gotURLs, "\n") != strings.Join(tc.wantURLs, "\n") {
				t.Fatalf("requested URLs = %q, err=%v, want %q", gotURLs, err, tc.wantURLs)
			}
			if _, err := os.Lstat(filepath.Join(installDir, "docmanager")); !os.IsNotExist(err) {
				t.Fatalf("bootstrap installed after unverified manifest: %v", err)
			}
		})
	}
}

func TestBootstrapInstallsExecutableFixture(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	script, err := os.ReadFile(filepath.Join(root, "scripts", "install.sh"))
	if err != nil {
		t.Fatal(err)
	}
	tmp := t.TempDir()
	archiveRaw := archive(t, tarEntry{name: "docmanager", body: "#!/bin/sh\nprintf 'fixture executed\\n'\n"})
	digest := sha256.Sum256(archiveRaw)
	artifactURL := "https://release-assets.githubusercontent.com/github-production-release-asset/docmanager_v1.2.3_linux_amd64.tar.gz"
	manifest, err := json.Marshal(map[string]any{
		"schema":  1,
		"key_id":  "docmanager-2026-01",
		"expires": time.Now().UTC().Add(time.Hour).Format(time.RFC3339),
		"artifacts": []map[string]any{{
			"version": "v1.2.3",
			"os":      "linux",
			"arch":    "amd64",
			"url":     artifactURL,
			"size":    len(archiveRaw),
			"sha256":  hex.EncodeToString(digest[:]),
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(tmp, "manifest.json")
	signaturePath := filepath.Join(tmp, "manifest.sig")
	archivePath := filepath.Join(tmp, "archive.tar.gz")
	for path, contents := range map[string][]byte{
		manifestPath:  manifest,
		signaturePath: []byte("fixture signature"),
		archivePath:   archiveRaw,
	} {
		if err := os.WriteFile(path, contents, 0o600); err != nil {
			t.Fatal(err)
		}
	}

	bin := filepath.Join(tmp, "bin")
	if err := os.Mkdir(bin, 0o700); err != nil {
		t.Fatal(err)
	}
	fakeCurl := `#!/bin/sh
output=
url=
while [ "$#" -gt 0 ]; do
  case "$1" in --output) shift; output=$1 ;; esac
  url=$1
  shift
done
case "$url" in
  */manifest.json) source=$DOCMANAGER_TEST_MANIFEST ;;
  */manifest.sig) source=$DOCMANAGER_TEST_SIGNATURE ;;
  */docmanager_v1.2.3_linux_amd64.tar.gz) source=$DOCMANAGER_TEST_ARCHIVE ;;
  *) exit 1 ;;
esac
cp "$source" "$output"
printf '%s' "$url"
`
	if err := os.WriteFile(filepath.Join(bin, "curl"), []byte(fakeCurl), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bin, "openssl"), []byte("#!/bin/sh\nexit 0\n"), 0o700); err != nil {
		t.Fatal(err)
	}

	installDir := filepath.Join(tmp, "install")
	command := exec.Command("sh", "-s")
	command.Dir = tmp
	command.Stdin = strings.NewReader(string(script))
	command.Env = append(withoutDocmanagerEnv(os.Environ()),
		"PATH="+bin+":"+os.Getenv("PATH"),
		"DOCMANAGER_VERSION=v1.2.3",
		"DOCMANAGER_INSTALL_DIR="+installDir,
		"DOCMANAGER_OS=linux",
		"DOCMANAGER_ARCH=amd64",
		"DOCMANAGER_TEST_MANIFEST="+manifestPath,
		"DOCMANAGER_TEST_SIGNATURE="+signaturePath,
		"DOCMANAGER_TEST_ARCHIVE="+archivePath,
	)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("bootstrap fixture failed: %v, output=%q", err, output)
	}
	installed := filepath.Join(installDir, "docmanager")
	output, err := exec.Command(installed).CombinedOutput()
	if err != nil || string(output) != "fixture executed\n" {
		t.Fatalf("installed fixture execution = %q, %v", output, err)
	}
}

func heredoc(t *testing.T, script, marker string) string {
	t.Helper()
	header := "<<'" + marker + "'\n"
	start := strings.Index(script, header)
	if start < 0 {
		t.Fatalf("missing %s heredoc", marker)
	}
	start += len(header)
	end := strings.Index(script[start:], marker+"\n")
	if end < 0 {
		t.Fatalf("unterminated %s heredoc", marker)
	}
	return script[start : start+end]
}

func withoutDocmanagerEnv(env []string) []string {
	clean := make([]string, 0, len(env))
	for _, entry := range env {
		if !strings.HasPrefix(entry, "DOCMANAGER_") {
			clean = append(clean, entry)
		}
	}
	return clean
}
