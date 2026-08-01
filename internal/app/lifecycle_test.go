package app

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/gentleman-programming/repository-documentation-manager/assets"
	"github.com/gentleman-programming/repository-documentation-manager/internal/domain"
)

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
