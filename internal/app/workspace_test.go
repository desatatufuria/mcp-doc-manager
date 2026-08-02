package app

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestWorkspaceInstallLeavesHookAbsentWithoutExplicitConsent(t *testing.T) {
	repo := testWorkspaceRepository(t)
	status, err := WorkspaceInstall(repo, false)
	if err != nil {
		t.Fatal(err)
	}
	if status.Hook != "absent" {
		t.Fatalf("hook = %q, want absent", status.Hook)
	}
	if _, err := os.Lstat(filepath.Join(repo, ".git", "hooks", "pre-push")); !os.IsNotExist(err) {
		t.Fatalf("default install created hook: %v", err)
	}
	if info, err := os.Stat(filepath.Join(repo, ".docmanager", "ledger.db")); err != nil || !info.Mode().IsRegular() {
		t.Fatalf("ledger = %v, %v", info, err)
	}
	if _, err := WorkspaceInstall(repo, false); err != nil {
		t.Fatalf("idempotent install: %v", err)
	}
}

func TestWorkspaceInstallRequiresExplicitHookOptIn(t *testing.T) {
	repo := testWorkspaceRepository(t)
	status, err := WorkspaceInstall(repo, true)
	if err != nil {
		t.Fatal(err)
	}
	if status.Hook != "opted-in" {
		t.Fatalf("hook = %q, want opted-in", status.Hook)
	}
	content, err := os.ReadFile(filepath.Join(repo, ".git", "hooks", "pre-push"))
	if err != nil || string(content) != workspaceHook(repo) {
		t.Fatalf("hook = %q, err = %v", content, err)
	}
}

func TestWorkspaceRefusesDriftedHookRemoval(t *testing.T) {
	repo := testWorkspaceRepository(t)
	if _, err := WorkspaceInstall(repo, true); err != nil {
		t.Fatal(err)
	}
	hook := filepath.Join(repo, ".git", "hooks", "pre-push")
	if err := os.WriteFile(hook, []byte("externally changed"), 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := WorkspaceUninstall(repo); !errors.Is(err, ErrWorkspaceDrift) {
		t.Fatalf("WorkspaceUninstall() error = %v, want ErrWorkspaceDrift", err)
	}
	content, _ := os.ReadFile(hook)
	if string(content) != "externally changed" {
		t.Fatalf("drifted hook was mutated: %q", content)
	}
	status, err := WorkspaceStatusFor(repo)
	if err != nil || status.Hook != "drifted" {
		t.Fatalf("status = %#v, err = %v", status, err)
	}
}

func TestWorkspaceRejectsNonRootGitSelectors(t *testing.T) {
	repo := testWorkspaceRepository(t)
	child := filepath.Join(repo, "child")
	if err := os.Mkdir(child, 0o700); err != nil {
		t.Fatal(err)
	}
	for _, target := range []string{child, repo + "/../" + filepath.Base(repo)} {
		if _, err := WorkspaceInstall(target, false); !errors.Is(err, ErrWorkspaceTarget) {
			t.Fatalf("WorkspaceInstall(%q) error = %v, want ErrWorkspaceTarget", target, err)
		}
	}
}

func testWorkspaceRepository(t *testing.T) string {
	t.Helper()
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	repo := t.TempDir()
	cmd := exec.Command("git", "init", repo)
	cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, output)
	}
	return repo
}
