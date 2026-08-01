package filesystem

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestTransactionRejectsSecondLockWithoutMutation(t *testing.T) {
	root := t.TempDir()
	first, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	release, err := first.Lock()
	if err != nil {
		t.Fatal(err)
	}
	defer release()

	second, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := second.Lock(); !errors.Is(err, ErrLocked) {
		t.Fatalf("Lock() error = %v, want ErrLocked", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".docmanager.lock")); err != nil {
		t.Fatalf("lock changed unexpectedly: %v", err)
	}
}

func TestTransactionWriteOwnedRecordsDigestModeAndAtomicallyReplaces(t *testing.T) {
	root := t.TempDir()
	tx, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.WriteOwned("state.json", []byte("before"), 0o600, nil); err != nil {
		t.Fatal(err)
	}
	previous, err := tx.Ownership("state.json")
	if err != nil {
		t.Fatal(err)
	}
	if previous.Mode != 0o600 || previous.Digest == "" {
		t.Fatalf("ownership = %#v, want digest and 0600 mode", previous)
	}
	if _, err := tx.WriteOwned("state.json", []byte("after"), 0o640, &previous); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(root, "state.json"))
	if err != nil || string(content) != "after" {
		t.Fatalf("content = %q, %v", content, err)
	}
	info, err := os.Stat(filepath.Join(root, "state.json"))
	if err != nil || info.Mode().Perm() != 0o640 {
		t.Fatalf("mode = %v, err = %v", info.Mode(), err)
	}
	if matches, _ := filepath.Glob(filepath.Join(root, ".state.json.*")); len(matches) != 0 {
		t.Fatalf("temporary files remain: %v", matches)
	}
}

func TestTransactionRejectsDriftOwnershipSymlinkAndRouteEscapeWithoutMutation(t *testing.T) {
	root := t.TempDir()
	tx, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.WriteOwned("owned", []byte("original"), 0o600, nil); err != nil {
		t.Fatal(err)
	}
	ownership, err := tx.Ownership("owned")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "owned"), []byte("changed"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.WriteOwned("owned", []byte("replacement"), 0o600, &ownership); !errors.Is(err, ErrDrift) {
		t.Fatalf("drift error = %v, want ErrDrift", err)
	}
	content, _ := os.ReadFile(filepath.Join(root, "owned"))
	if string(content) != "changed" {
		t.Fatalf("drifted content mutated: %q", content)
	}

	if err := os.Symlink(filepath.Join(root, "owned"), filepath.Join(root, "link")); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.WriteOwned("link", []byte("nope"), 0o600, nil); !errors.Is(err, ErrUnsafePath) {
		t.Fatalf("symlink error = %v, want ErrUnsafePath", err)
	}
	if _, err := tx.WriteOwned("../escape", []byte("nope"), 0o600, nil); !errors.Is(err, ErrUnsafePath) {
		t.Fatalf("escape error = %v, want ErrUnsafePath", err)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(root), "escape")); !os.IsNotExist(err) {
		t.Fatalf("route escape mutated filesystem: %v", err)
	}
}
