package app

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/desatatufuria/mcp-doc-manager/assets"
	"github.com/desatatufuria/mcp-doc-manager/internal/domain"
)

func TestOpenCodeGuidanceIsVersionedAndBounded(t *testing.T) {
	guidance := assets.OpenCodeGuidance
	if guidance.Path != "guidance/opencode.md" || guidance.Version == "" || guidance.Digest == "" {
		t.Fatalf("identity = %#v", guidance)
	}
	for _, required := range []string{
		"worktree", "staged", "base..head", "one `document_change`", "human review",
		"one `verify_receipt`", "MUST NOT mutate", "user instructions", "AGENTS.md",
		"routine coding", "unrelated questions", "inferred scope", "automatic editing",
		"unavailable evidence", "MCP lifecycle", "workspace", "hook", "auto-enable hooks",
		"other agents", "generic agent", "release", "historical installer-plan evidence",
	} {
		if !strings.Contains(guidance.Content, required) {
			t.Errorf("guidance is missing %q", required)
		}
	}
}

func TestInstallUpgradesOnlyKnownOpenCodeGuidanceAndPreservesUserFiles(t *testing.T) {
	repo := t.TempDir()
	appGit(t, repo, "init")
	if err := Install(repo); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(repo, ".docmanager")
	guide := filepath.Join(dir, assets.OpenCodeGuidance.Path)
	older := assets.OpenCodeGuidanceSnapshots[0]
	if err := os.WriteFile(guide, []byte(older.Content), 0o600); err != nil {
		t.Fatal(err)
	}
	userFile := filepath.Join(dir, "user-notes.txt")
	if err := os.WriteFile(userFile, []byte("retain me\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Install(repo); err != nil {
		t.Fatalf("upgrade known guidance: %v", err)
	}
	if got, err := os.ReadFile(guide); err != nil || string(got) != assets.OpenCodeGuidance.Content {
		t.Fatalf("upgraded guidance = %q, %v", got, err)
	}
	if got, err := os.ReadFile(userFile); err != nil || string(got) != "retain me\n" {
		t.Fatalf("user file = %q, %v", got, err)
	}
	for _, content := range []string{"edited guidance\n", "substituted guidance\n"} {
		if err := os.WriteFile(guide, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := Install(repo); !errors.Is(err, domain.ErrInvalidTarget) {
			t.Fatalf("drift %q = %v", content, err)
		}
	}
	if err := os.Remove(guide); err != nil {
		t.Fatal(err)
	}
	if err := Install(repo); !errors.Is(err, domain.ErrInvalidTarget) {
		t.Fatalf("missing guidance = %v", err)
	}
}

func TestLifecycleContainsOnlyOwnedRepositoryState(t *testing.T) {
	repo := t.TempDir()
	appGit(t, repo, "init")
	if err := Install(repo); err != nil {
		t.Fatal(err)
	}
	if err := Doctor(repo); err != nil {
		t.Fatal(err)
	}
	if err := Uninstall(repo); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(repo, ".docmanager")); !os.IsNotExist(err) {
		t.Fatalf("state remains: %v", err)
	}

	if err := os.Mkdir(filepath.Join(repo, ".docmanager"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := Install(repo); !errors.Is(err, domain.ErrInvalidTarget) {
		t.Fatalf("unowned install = %v", err)
	}
	if err := Doctor(repo); !errors.Is(err, domain.ErrInvalidTarget) {
		t.Fatalf("unowned doctor = %v", err)
	}
	if err := Uninstall(repo); !errors.Is(err, domain.ErrInvalidTarget) {
		t.Fatalf("unowned uninstall = %v", err)
	}
	if err := os.RemoveAll(filepath.Join(repo, ".docmanager")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(t.TempDir(), filepath.Join(repo, ".docmanager")); err != nil {
		t.Fatal(err)
	}
	if err := Install(repo); !errors.Is(err, domain.ErrInvalidTarget) {
		t.Fatalf("symlink install = %v", err)
	}
	if err := Uninstall(repo); !errors.Is(err, domain.ErrInvalidTarget) {
		t.Fatalf("symlink uninstall = %v", err)
	}
	if err := Doctor(repo); !errors.Is(err, domain.ErrInvalidTarget) {
		t.Fatalf("symlink doctor = %v", err)
	}
}

func TestInstallWritesDeterministicContainedAssets(t *testing.T) {
	repo := t.TempDir()
	appGit(t, repo, "init")
	if err := Install(repo); err != nil {
		t.Fatal(err)
	}
	for path, want := range assets.Files {
		got, err := os.ReadFile(filepath.Join(repo, ".docmanager", path))
		if err != nil || string(got) != want {
			t.Fatalf("asset %s = %q, %v", path, got, err)
		}
	}
	if err := Install(repo); err != nil {
		t.Fatalf("idempotent install: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".docmanager", "config.json"), []byte("tampered\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Doctor(repo); !errors.Is(err, domain.ErrInvalidTarget) {
		t.Fatalf("doctor accepted tampered config: %v", err)
	}
}

func TestLifecycleRejectsUnsafeOwnershipMarkers(t *testing.T) {
	repo := t.TempDir()
	appGit(t, repo, "init")
	dir := filepath.Join(repo, ".docmanager")
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	for _, marker := range []struct {
		name string
		make func() error
	}{
		{"wrong-content", func() error { return os.WriteFile(filepath.Join(dir, ".owned"), []byte("other\n"), 0o600) }},
		{"directory", func() error { return os.Mkdir(filepath.Join(dir, ".owned"), 0o700) }},
		{"symlink", func() error { return os.Symlink(filepath.Join(repo, "outside"), filepath.Join(dir, ".owned")) }},
	} {
		t.Run(marker.name, func(t *testing.T) {
			for _, operation := range []struct {
				name string
				fn   func(string) error
			}{{"install", Install}, {"doctor", Doctor}, {"uninstall", Uninstall}} {
				if err := marker.make(); err != nil {
					t.Fatal(err)
				}
				if err := operation.fn(repo); !errors.Is(err, domain.ErrInvalidTarget) {
					t.Fatalf("%s = %v", operation.name, err)
				}
				if err := os.RemoveAll(filepath.Join(dir, ".owned")); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestLifecycleRejectsTraversalAndNonRepositoryTargets(t *testing.T) {
	if err := Install(filepath.Join(t.TempDir(), "..")); !errors.Is(err, domain.ErrInvalidTarget) {
		t.Fatalf("traversal = %v", err)
	}
	if err := Doctor(t.TempDir()); !errors.Is(err, domain.ErrInvalidTarget) {
		t.Fatalf("non-repository doctor = %v", err)
	}
	for _, operation := range []struct {
		name  string
		setup func(string) error
		fn    func(string) error
	}{
		{"install", func(string) error { return nil }, Install},
		{"doctor", Install, Doctor},
		{"uninstall", Install, Uninstall},
	} {
		t.Run(operation.name, func(t *testing.T) {
			repo := t.TempDir()
			appGit(t, repo, "init")
			if err := operation.setup(repo); err != nil {
				t.Fatal(err)
			}
			target := repo + string(filepath.Separator) + "nested" + string(filepath.Separator) + ".."
			if err := operation.fn(target); !errors.Is(err, domain.ErrInvalidTarget) {
				t.Fatalf("contained traversal = %v", err)
			}
			if operation.name != "install" {
				if _, err := os.Stat(filepath.Join(repo, ".docmanager", ".owned")); err != nil {
					t.Fatalf("owned state changed: %v", err)
				}
			}
		})
	}
}

func TestInstallRollsBackNewStateWhenOwnershipWriteFails(t *testing.T) {
	repo := t.TempDir()
	appGit(t, repo, "init")
	original := writeOwnership
	writeOwnership = func(string, []byte, os.FileMode) error { return errors.New("write failed") }
	t.Cleanup(func() { writeOwnership = original })
	if err := Install(repo); !errors.Is(err, domain.ErrLedgerFailure) {
		t.Fatalf("Install() = %v", err)
	}
	if _, err := os.Stat(filepath.Join(repo, ".docmanager")); !os.IsNotExist(err) {
		t.Fatalf("partial state remains: %v", err)
	}
}
