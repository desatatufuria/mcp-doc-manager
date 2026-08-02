package sqlite

import (
	"context"
	"database/sql"
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
	if got, err := l.Version(context.Background()); err != nil || got != 2 {
		t.Fatalf("Version() = %d, %v", got, err)
	}
	if info, err := os.Lstat(filepath.Join(root, ".docmanager", "lifecycle.db")); err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0o600 {
		t.Fatalf("database safety = %v, %v", info, err)
	}
	if info, err := os.Lstat(filepath.Join(root, ".docmanager")); err != nil || info.Mode().Perm() != 0o700 {
		t.Fatalf("state directory mode = %v, %v", info, err)
	}
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, ".docmanager", "unsafe.db")); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenLifecycle(filepath.Join(root, ".docmanager", "unsafe.db")); !errors.Is(err, domain.ErrLifecycle) {
		t.Fatalf("unsafe workspace = %v", err)
	}
	if err := os.Chmod(root, 0o777); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenLifecycle(root); !errors.Is(err, domain.ErrLifecycle) {
		t.Fatalf("unsafe mode = %v", err)
	}
}

func TestLifecycleRejectsUnsafeStatePaths(t *testing.T) {
	for _, tt := range []struct {
		name  string
		setup func(string) error
	}{
		{"state symlink", func(root string) error { return os.Symlink(t.TempDir(), filepath.Join(root, ".docmanager")) }},
		{"database symlink", func(root string) error {
			dir := filepath.Join(root, ".docmanager")
			if err := os.Mkdir(dir, 0o700); err != nil {
				return err
			}
			return os.Symlink(t.TempDir(), filepath.Join(dir, "lifecycle.db"))
		}},
		{"database non-regular", func(root string) error {
			dir := filepath.Join(root, ".docmanager", "lifecycle.db")
			return os.MkdirAll(dir, 0o700)
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			if err := tt.setup(root); err != nil {
				t.Fatal(err)
			}
			if _, err := OpenLifecycle(root); !errors.Is(err, domain.ErrLifecycle) {
				t.Fatalf("OpenLifecycle() = %v", err)
			}
		})
	}
}

func TestLifecycleAtomicProvenanceAndIdempotency(t *testing.T) {
	l, err := OpenLifecycle(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	ctx := context.Background()
	good := provenance("input")
	if _, err := l.Save(ctx, "wrong", "approved", good); !errors.Is(err, domain.ErrLifecycle) {
		t.Fatalf("key mismatch = %v", err)
	}
	if got, err := l.Count(ctx); err != nil || got != 0 {
		t.Fatalf("mismatch count = %d, %v", got, err)
	}
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
	if got, err := l.Save(ctx, "key-1", "changed-result", good); err != nil || got != first {
		t.Fatalf("result replay = %q, %v", got, err)
	}
	for _, mutate := range []func(*domain.Provenance){
		func(p *domain.Provenance) { p.Operation = "other" }, func(p *domain.Provenance) { p.Request = "other" },
		func(p *domain.Provenance) { p.Input = "other" }, func(p *domain.Provenance) { p.Approval = "other" },
		func(p *domain.Provenance) { p.Evidence = "other" }, func(p *domain.Provenance) { p.Versions = "other" }, func(p *domain.Provenance) { p.Context = "other" },
	} {
		changed := good
		mutate(&changed)
		if _, err := l.Save(ctx, "key-1", "approved", changed); !errors.Is(err, domain.ErrIdempotencyConflict) {
			t.Fatalf("divergent identity = %v", err)
		}
	}
	if got, err := l.Provenance(ctx, "key-1"); err != nil || got != good {
		t.Fatalf("provenance = %#v, %v", got, err)
	}
	collision := provenance("input")
	collision.IdempotencyKey, collision.Operation, collision.Request = "nul", "a\x00b", "c"
	if _, err := l.Save(ctx, "nul", "approved", collision); err != nil {
		t.Fatal(err)
	}
	second := collision
	second.Operation, second.Request = "a", "b\x00c"
	if _, err := l.Save(ctx, "nul", "approved", second); !errors.Is(err, domain.ErrIdempotencyConflict) {
		t.Fatalf("NUL collision = %v", err)
	}
}

func provenance(input string) domain.Provenance {
	return domain.Provenance{Authority: domain.DeclaredLocalProvenance, DeclaredActor: "local-user", Interaction: "mcp", Context: "request", Operation: "save", Request: "request", IdempotencyKey: "key-1", Time: "2026-08-02T00:00:00Z", Approval: "approved", Evidence: "evidence", Input: input, Result: "approved", Versions: "plan-1/policy-1"}
}

func TestLifecycleMigratesOlderSchemaAndRollsBackFailure(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, ".docmanager")
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", filepath.Join(dir, "lifecycle.db"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`CREATE TABLE schema_migrations(version INTEGER PRIMARY KEY); INSERT INTO schema_migrations VALUES(1); CREATE TABLE lifecycle_records(id INTEGER PRIMARY KEY, state TEXT NOT NULL)`); err != nil {
		t.Fatal(err)
	}
	db.Close()
	l, err := OpenLifecycle(root)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	if got, err := l.Version(context.Background()); err != nil || got != 2 {
		t.Fatalf("forward migration = %d, %v", got, err)
	}
	if err := l.migrateOne(context.Background(), 3, `CREATE TABLE rejected(`); !errors.Is(err, domain.ErrLifecycle) {
		t.Fatalf("failed migration = %v", err)
	}
	var n int
	if err := l.db.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE version=3`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("rollback version = %d, %v", n, err)
	}
	if err := l.db.QueryRow(`SELECT COUNT(*) FROM lifecycle_records`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("rollback schema = %d, %v", n, err)
	}
}
