package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestCLIWorkspaceInstallHonorsExplicitHookOptIn(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	repo := t.TempDir()
	if output, err := exec.Command("git", "init", repo).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, output)
	}
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(repo); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(previous)

	if err := run([]string{"workspace", "install", "--enable-hook", "--json"}); err != nil {
		t.Fatal(err)
	}
	info, err := os.Lstat(filepath.Join(repo, ".git", "hooks", "pre-push"))
	if err != nil || !info.Mode().IsRegular() {
		t.Fatalf("explicit opt-in did not create a regular hook: %v", err)
	}
}
