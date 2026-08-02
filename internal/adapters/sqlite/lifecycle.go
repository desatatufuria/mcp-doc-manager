package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/desatatufuria/mcp-doc-manager/internal/domain"
	_ "modernc.org/sqlite"
)

type Lifecycle struct{ db *sql.DB }

func OpenLifecycle(root string) (*Lifecycle, error) {
	if !safeDir(root) {
		return nil, fmt.Errorf("%w: unsafe root", domain.ErrLifecycle)
	}
	dir := filepath.Join(root, ".docmanager")
	if err := os.MkdirAll(dir, 0o700); err != nil || os.Chmod(dir, 0o700) != nil || !safeDir(dir) {
		return nil, fmt.Errorf("%w: unsafe state directory", domain.ErrLifecycle)
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
	if err := os.Chmod(path, 0o600); err != nil {
		db.Close()
		return nil, domain.ErrLifecycle
	}
	return l, nil
}

// safeDir is the strongest portable boundary: a non-symlink private directory; UID ownership is not portable.
func safeDir(path string) bool {
	info, err := os.Lstat(path)
	return err == nil && info.IsDir() && info.Mode()&os.ModeSymlink == 0 && info.Mode().Perm()&0o022 == 0
}
func (l *Lifecycle) migrate(ctx context.Context) error {
	if _, err := l.db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations(version INTEGER PRIMARY KEY)`); err != nil {
		return domain.ErrLifecycle
	}
	var version int
	if err := l.db.QueryRowContext(ctx, `SELECT COALESCE(MAX(version),0) FROM schema_migrations`).Scan(&version); err != nil {
		return domain.ErrLifecycle
	}
	for _, m := range []struct {
		version int
		sql     string
	}{{1, `CREATE TABLE IF NOT EXISTS lifecycle_records(id INTEGER PRIMARY KEY, state TEXT NOT NULL)`}, {2, `CREATE TABLE IF NOT EXISTS provenance(record_id INTEGER PRIMARY KEY, authority TEXT NOT NULL, actor TEXT NOT NULL, interaction TEXT NOT NULL, context TEXT NOT NULL, operation TEXT NOT NULL, request TEXT NOT NULL, idempotency_key TEXT NOT NULL, time TEXT NOT NULL, approval TEXT NOT NULL, evidence TEXT NOT NULL, input TEXT NOT NULL, result TEXT NOT NULL, versions TEXT NOT NULL); CREATE TABLE IF NOT EXISTS idempotency(key TEXT PRIMARY KEY, identity TEXT NOT NULL, result TEXT NOT NULL, record_id INTEGER NOT NULL)`}} {
		if version < m.version {
			if err := l.migrateOne(ctx, m.version, m.sql); err != nil {
				return err
			}
		}
	}
	return nil
}
func (l *Lifecycle) migrateOne(ctx context.Context, version int, statement string) error {
	c, err := l.db.Conn(ctx)
	if err != nil {
		return domain.ErrLifecycle
	}
	defer c.Close()
	if _, err = c.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		return domain.ErrLifecycle
	}
	committed := false
	defer func() {
		if !committed {
			_, _ = c.ExecContext(context.Background(), "ROLLBACK")
		}
	}()
	if _, err = c.ExecContext(ctx, statement); err != nil {
		return fmt.Errorf("%w: migration statement: %v", domain.ErrLifecycle, err)
	}
	if _, err = c.ExecContext(ctx, `INSERT INTO schema_migrations VALUES(?)`, version); err != nil {
		return fmt.Errorf("%w: migration version: %v", domain.ErrLifecycle, err)
	}
	if _, err = c.ExecContext(ctx, "COMMIT"); err != nil {
		return fmt.Errorf("%w: migration commit: %v", domain.ErrLifecycle, err)
	}
	committed = true
	return nil
}
func (l *Lifecycle) Version(ctx context.Context) (int, error) {
	var v int
	if err := l.db.QueryRowContext(ctx, "SELECT MAX(version) FROM schema_migrations").Scan(&v); err != nil {
		return 0, domain.ErrLifecycle
	}
	return v, nil
}
func (l *Lifecycle) Count(ctx context.Context) (int, error) {
	var n int
	if err := l.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM lifecycle_records").Scan(&n); err != nil {
		return 0, domain.ErrLifecycle
	}
	return n, nil
}

func identity(p domain.Provenance) string {
	return fmt.Sprintf("%d:%s%d:%s%d:%s%d:%s%d:%s%d:%s%d:%s", len(p.Operation), p.Operation, len(p.Request), p.Request, len(p.Input), p.Input, len(p.Approval), p.Approval, len(p.Evidence), p.Evidence, len(p.Versions), p.Versions, len(p.Context), p.Context)
}
func (l *Lifecycle) Save(ctx context.Context, key, result string, p domain.Provenance) (string, error) {
	if l == nil || l.db == nil || key == "" || key != p.IdempotencyKey || result == "" || p.Validate() != nil {
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
	id := identity(p)
	var storedID, storedResult string
	var recordID int64
	err = c.QueryRowContext(ctx, "SELECT identity,result,record_id FROM idempotency WHERE key=?", key).Scan(&storedID, &storedResult, &recordID)
	if err == nil {
		if storedID != id {
			return "", domain.ErrIdempotencyConflict
		}
		if _, err = c.ExecContext(ctx, "COMMIT"); err != nil {
			return "", domain.ErrLifecycle
		}
		committed = true
		return storedResult, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return "", domain.ErrLifecycle
	}
	r, err := c.ExecContext(ctx, "INSERT INTO lifecycle_records(state) VALUES(?)", result)
	if err != nil {
		return "", domain.ErrLifecycle
	}
	recordID, err = r.LastInsertId()
	if err != nil {
		return "", domain.ErrLifecycle
	}
	if _, err = c.ExecContext(ctx, `INSERT INTO provenance VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, recordID, p.Authority, p.DeclaredActor, p.Interaction, p.Context, p.Operation, p.Request, p.IdempotencyKey, p.Time, p.Approval, p.Evidence, p.Input, p.Result, p.Versions); err != nil {
		return "", domain.ErrLifecycle
	}
	if _, err = c.ExecContext(ctx, "INSERT INTO idempotency VALUES(?,?,?,?)", key, id, result, recordID); err != nil {
		return "", domain.ErrLifecycle
	}
	if _, err = c.ExecContext(ctx, "COMMIT"); err != nil {
		return "", domain.ErrLifecycle
	}
	committed = true
	return result, nil
}
func (l *Lifecycle) Provenance(ctx context.Context, key string) (p domain.Provenance, err error) {
	err = l.db.QueryRowContext(ctx, `SELECT provenance.authority,provenance.actor,provenance.interaction,provenance.context,provenance.operation,provenance.request,provenance.idempotency_key,provenance.time,provenance.approval,provenance.evidence,provenance.input,provenance.result,provenance.versions FROM provenance JOIN idempotency ON provenance.record_id=idempotency.record_id WHERE key=?`, key).Scan(&p.Authority, &p.DeclaredActor, &p.Interaction, &p.Context, &p.Operation, &p.Request, &p.IdempotencyKey, &p.Time, &p.Approval, &p.Evidence, &p.Input, &p.Result, &p.Versions)
	if err != nil {
		return p, fmt.Errorf("%w: read provenance: %v", domain.ErrLifecycle, err)
	}
	return p, nil
}
func (l *Lifecycle) Close() error {
	if l == nil || l.db == nil {
		return nil
	}
	return l.db.Close()
}
