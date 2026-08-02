package sqlite

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/desatatufuria/mcp-doc-manager/internal/domain"
)

func TestLifecycleOwnsWorkspaceAndMigratesForward(t *testing.T) {
	root := t.TempDir()
	l, err := OpenLifecycle(root)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	if got, err := l.Version(context.Background()); err != nil || got != 1 {
		t.Fatalf("Version() = %d, %v", got, err)
	}
	if info, err := os.Lstat(filepath.Join(root, ".docmanager", "lifecycle.db")); err != nil || info.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("database safety = %v, %v", info, err)
	}
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, ".docmanager", "unsafe.db")); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenLifecycle(filepath.Join(root, ".docmanager", "unsafe.db")); !errors.Is(err, domain.ErrLifecycle) {
		t.Fatalf("unsafe workspace = %v", err)
	}
}

func TestLifecycleAtomicProvenanceAndIdempotency(t *testing.T) {
	l, err := OpenLifecycle(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	ctx := context.Background()
	good := domain.Provenance{DeclaredActor: "local-user", Interaction: "mcp", Context: "request"}
	if _, err := l.db.Exec(`CREATE TRIGGER fail_provenance BEFORE INSERT ON provenance BEGIN SELECT RAISE(ABORT, 'stop'); END`); err != nil {
		t.Fatal(err)
	}
	if _, err := l.Save(ctx, "rollback", "approved", good); !errors.Is(err, domain.ErrLifecycle) {
		t.Fatalf("rollback = %v", err)
	}
	if _, err := l.db.Exec(`DROP TRIGGER fail_provenance`); err != nil {
		t.Fatal(err)
	}
	if got, err := l.Count(ctx); err != nil || got != 0 {
		t.Fatalf("rollback count = %d, %v", got, err)
	}
	first, err := l.Save(ctx, "key-1", "approved", good)
	if err != nil {
		t.Fatal(err)
	}
	replay, err := l.Save(ctx, "key-1", "approved", good)
	if err != nil || replay != first {
		t.Fatalf("replay = %q, %v", replay, err)
	}
	if _, err := l.Save(ctx, "key-1", "rejected", good); !errors.Is(err, domain.ErrIdempotencyConflict) {
		t.Fatalf("divergent key = %v", err)
	}
}
