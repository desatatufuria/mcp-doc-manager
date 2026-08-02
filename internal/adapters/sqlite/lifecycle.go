package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"

	"github.com/desatatufuria/mcp-doc-manager/internal/domain"
	_ "modernc.org/sqlite"
)

type Lifecycle struct{ db *sql.DB }

func OpenLifecycle(root string) (*Lifecycle, error) {
	if !safeDir(root) {
		return nil, domain.ErrLifecycle
	}
	dir := filepath.Join(root, ".docmanager")
	if err := os.MkdirAll(dir, 0o700); err != nil || !safeDir(dir) {
		return nil, domain.ErrLifecycle
	}
	path := filepath.Join(dir, "lifecycle.db")
	if info, err := os.Lstat(path); err == nil && (!info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0) {
		return nil, domain.ErrLifecycle
	} else if err != nil && !os.IsNotExist(err) {
		return nil, domain.ErrLifecycle
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, domain.ErrLifecycle
	}
	l := &Lifecycle{db: db}
	if err := l.migrate(context.Background()); err != nil {
		db.Close()
		return nil, err
	}
	_ = os.Chmod(path, 0o600)
	return l, nil
}

func safeDir(path string) bool {
	info, err := os.Lstat(path)
	return err == nil && info.IsDir() && info.Mode()&os.ModeSymlink == 0 && info.Mode().Perm()&0o022 == 0
}
func (l *Lifecycle) migrate(ctx context.Context) error {
	_, err := l.db.ExecContext(ctx, `BEGIN IMMEDIATE; CREATE TABLE IF NOT EXISTS schema_migrations(version INTEGER PRIMARY KEY); CREATE TABLE IF NOT EXISTS lifecycle_records(id INTEGER PRIMARY KEY, state TEXT NOT NULL); CREATE TABLE IF NOT EXISTS provenance(record_id INTEGER NOT NULL, actor TEXT NOT NULL CHECK(actor<>''), interaction TEXT NOT NULL, context TEXT NOT NULL); CREATE TABLE IF NOT EXISTS idempotency(key TEXT PRIMARY KEY, input TEXT NOT NULL, result TEXT NOT NULL); INSERT OR IGNORE INTO schema_migrations VALUES(1); COMMIT`)
	if err != nil {
		_, _ = l.db.ExecContext(ctx, "ROLLBACK")
		return domain.ErrLifecycle
	}
	return nil
}
func (l *Lifecycle) Version(ctx context.Context) (int, error) {
	var v int
	err := l.db.QueryRowContext(ctx, "SELECT MAX(version) FROM schema_migrations").Scan(&v)
	if err != nil {
		return 0, domain.ErrLifecycle
	}
	return v, nil
}
func (l *Lifecycle) Count(ctx context.Context) (int, error) {
	var n int
	err := l.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM lifecycle_records").Scan(&n)
	if err != nil {
		return 0, domain.ErrLifecycle
	}
	return n, nil
}

func (l *Lifecycle) Save(ctx context.Context, key, state string, p domain.Provenance) (string, error) {
	if l == nil || l.db == nil || key == "" || state == "" || p.Validate() != nil {
		return "", domain.ErrLifecycle
	}
	c, err := l.db.Conn(ctx)
	if err != nil {
		return "", domain.ErrLifecycle
	}
	defer c.Close()
	if _, err = c.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		return "", domain.ErrLifecycle
	}
	committed := false
	defer func() {
		if !committed {
			_, _ = c.ExecContext(context.Background(), "ROLLBACK")
		}
	}()
	var input, result string
	err = c.QueryRowContext(ctx, "SELECT input,result FROM idempotency WHERE key=?", key).Scan(&input, &result)
	if err == nil {
		if input != state {
			return "", domain.ErrIdempotencyConflict
		}
		committed = true
		_, _ = c.ExecContext(ctx, "COMMIT")
		return result, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return "", domain.ErrLifecycle
	}
	r, err := c.ExecContext(ctx, "INSERT INTO lifecycle_records(state) VALUES(?)", state)
	if err != nil {
		return "", domain.ErrLifecycle
	}
	id, _ := r.LastInsertId()
	if _, err = c.ExecContext(ctx, "INSERT INTO provenance VALUES(?,?,?,?)", id, p.DeclaredActor, p.Interaction, p.Context); err != nil {
		return "", domain.ErrLifecycle
	}
	if _, err = c.ExecContext(ctx, "INSERT INTO idempotency VALUES(?,?,?)", key, state, state); err != nil {
		return "", domain.ErrLifecycle
	}
	if _, err = c.ExecContext(ctx, "COMMIT"); err != nil {
		return "", domain.ErrLifecycle
	}
	committed = true
	return state, nil
}
func (l *Lifecycle) Close() error {
	if l == nil || l.db == nil {
		return nil
	}
	return l.db.Close()
}
