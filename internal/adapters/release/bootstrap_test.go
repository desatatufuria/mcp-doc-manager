package release

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestReleaseBootstrapAssetsDefineSignedSupportedArchivesWithoutPrivateMaterial(t *testing.T) {
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

	tracked, err := exec.Command("git", "-C", root, "ls-files", "-z").Output()
	if err != nil {
		t.Fatalf("list committed files: %v", err)
	}
	for _, path := range strings.Split(string(tracked), "\x00") {
		name := strings.ToLower(filepath.Base(path))
		if strings.Contains(name, "private") || strings.Contains(name, "signing-key") || strings.HasSuffix(name, ".key") {
			t.Fatalf("committed private signing material detected: %s", path)
		}
	}
	privatePath := filepath.Join("assets", "release", "private-key.pem")
	if err := exec.Command("git", "-C", root, "check-ignore", "--quiet", privatePath).Run(); err != nil {
		t.Fatalf("local private signing key path is not ignored: %v", err)
	}
	if err := exec.Command("git", "-C", root, "ls-files", "--error-unmatch", privatePath).Run(); err == nil {
		t.Fatal("local private signing key is tracked")
	}
}

func TestBootstrapFixtureFailsClosedWithoutInstalling(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	tmp := t.TempDir()
	bin := filepath.Join(tmp, "bin")
	if err := os.Mkdir(bin, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bin, "curl"), []byte("#!/bin/sh\nexit 1\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	installDir := filepath.Join(tmp, "install")
	command := exec.Command("sh", filepath.Join(root, "scripts", "install.sh"))
	command.Env = append(os.Environ(), "PATH="+bin+":"+os.Getenv("PATH"), "DOCMANAGER_RELEASE_BASE=https://github.com/example/releases", "DOCMANAGER_INSTALL_DIR="+installDir, "DOCMANAGER_OS=linux", "DOCMANAGER_ARCH=amd64")
	if output, err := command.CombinedOutput(); err == nil || !strings.Contains(string(output), "unable to fetch trusted manifest") {
		t.Fatalf("bootstrap failure = %v, output=%q", err, output)
	}
	if _, err := os.Lstat(filepath.Join(installDir, "docmanager")); !os.IsNotExist(err) {
		t.Fatalf("bootstrap installed after unverified manifest: %v", err)
	}
}
