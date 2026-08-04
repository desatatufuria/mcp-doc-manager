package sqlite

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
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
	}{{1, `CREATE TABLE IF NOT EXISTS lifecycle_records(id INTEGER PRIMARY KEY, state TEXT NOT NULL)`}} {
		if version < m.version {
			if err := l.migrateOne(ctx, m.version, m.sql); err != nil {
				return err
			}
		}
	}
	if version < 2 {
		if err := l.migrateV2(ctx); err != nil {
			return err
		}
		version = 2
	}
	if version < 3 {
		return l.migrateOne(ctx, 3, v3Schema)
	}
	return nil
}

const v2Provenance = `CREATE TABLE provenance_v2(record_id INTEGER PRIMARY KEY, authority TEXT NOT NULL, actor TEXT NOT NULL, interaction TEXT NOT NULL, context TEXT NOT NULL, operation TEXT NOT NULL, request TEXT NOT NULL, idempotency_key TEXT NOT NULL, time TEXT NOT NULL, approval TEXT NOT NULL, evidence TEXT NOT NULL, input TEXT NOT NULL, result TEXT NOT NULL, versions TEXT NOT NULL)`
const v2Idempotency = `CREATE TABLE idempotency_v2(key TEXT PRIMARY KEY, input TEXT NOT NULL, identity TEXT NOT NULL, result TEXT NOT NULL, record_id INTEGER, replay_state TEXT NOT NULL CHECK(replay_state IN ('available','legacy_unavailable')))`
const v3Schema = `CREATE TABLE catalog_entries(path TEXT PRIMARY KEY,purpose TEXT NOT NULL,audience TEXT NOT NULL,owner TEXT NOT NULL,related_areas TEXT NOT NULL,state TEXT NOT NULL,evidence TEXT NOT NULL,last_verification TEXT NOT NULL,pending_actions TEXT NOT NULL,record_id INTEGER NOT NULL);
CREATE TABLE catalog_imports(record_id INTEGER NOT NULL,path TEXT NOT NULL,digest TEXT NOT NULL,authority TEXT NOT NULL,actor TEXT NOT NULL,interaction TEXT NOT NULL,context TEXT NOT NULL,operation TEXT NOT NULL,request TEXT NOT NULL,idempotency_key TEXT NOT NULL,time TEXT NOT NULL,approval TEXT NOT NULL,evidence TEXT NOT NULL,input TEXT NOT NULL,result TEXT NOT NULL,versions TEXT NOT NULL,PRIMARY KEY(record_id,path));
CREATE TABLE initial_policy(revision TEXT PRIMARY KEY,mode TEXT NOT NULL,approved INTEGER NOT NULL CHECK(approved=1),record_id INTEGER NOT NULL UNIQUE);
CREATE TABLE catalog_policy_current(singleton INTEGER PRIMARY KEY CHECK(singleton=1),revision TEXT NOT NULL UNIQUE REFERENCES initial_policy(revision));
CREATE TRIGGER initial_policy_no_update BEFORE UPDATE ON initial_policy BEGIN SELECT RAISE(ABORT,'initial policy is immutable'); END;
CREATE TRIGGER initial_policy_no_delete BEFORE DELETE ON initial_policy BEGIN SELECT RAISE(ABORT,'initial policy is immutable'); END`

func (l *Lifecycle) migrateV2(ctx context.Context) error {
	c, err := l.db.Conn(ctx)
	if err != nil {
		return domain.ErrLifecycle
	}
	defer c.Close()
	if _, err = c.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		return domain.ErrLifecycle
	}
	fail := func(cause error) error {
		if _, rollbackErr := c.ExecContext(ctx, "ROLLBACK"); rollbackErr != nil {
			return fmt.Errorf("%w: migration rollback: %v", domain.ErrLifecycle, rollbackErr)
		}
		return fmt.Errorf("%w: migration v2: %v", domain.ErrLifecycle, cause)
	}
	var oldProvenance int
	if err = c.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='provenance'`).Scan(&oldProvenance); err != nil {
		return fail(err)
	}
	if oldProvenance == 0 {
		if _, err = c.ExecContext(ctx, v2Provenance+`; ALTER TABLE provenance_v2 RENAME TO provenance; `+v2Idempotency+`; ALTER TABLE idempotency_v2 RENAME TO idempotency`); err != nil {
			return fail(err)
		}
	} else if _, err = c.ExecContext(ctx, `ALTER TABLE provenance RENAME TO provenance_v1; ALTER TABLE idempotency RENAME TO idempotency_v1; `+v2Provenance+`; `+v2Idempotency+`;
		INSERT INTO provenance_v2 SELECT record_id, 'declared_local_provenance', actor, interaction, context, 'legacy', 'legacy', 'legacy-record-' || record_id, 'legacy', 'legacy', 'legacy', 'legacy', 'legacy', 'v1' FROM provenance_v1;
		INSERT INTO idempotency_v2(key,input,identity,result,record_id,replay_state) SELECT key, input, 'legacy:' || input, result, NULL, 'legacy_unavailable' FROM idempotency_v1;
		DROP TABLE provenance_v1; DROP TABLE idempotency_v1; ALTER TABLE provenance_v2 RENAME TO provenance; ALTER TABLE idempotency_v2 RENAME TO idempotency`); err != nil {
		return fail(err)
	}
	if _, err = c.ExecContext(ctx, `INSERT INTO schema_migrations VALUES(2)`); err != nil {
		return fail(err)
	}
	if _, err = c.ExecContext(ctx, "COMMIT"); err != nil {
		return fail(err)
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
	fail := func(cause error) error {
		if _, rollbackErr := c.ExecContext(ctx, "ROLLBACK"); rollbackErr != nil {
			return fmt.Errorf("%w: migration rollback: %v", domain.ErrLifecycle, rollbackErr)
		}
		return fmt.Errorf("%w: migration statement: %v", domain.ErrLifecycle, cause)
	}
	if _, err = c.ExecContext(ctx, statement); err != nil {
		return fail(err)
	}
	if _, err = c.ExecContext(ctx, `INSERT INTO schema_migrations VALUES(?)`, version); err != nil {
		return fail(err)
	}
	if _, err = c.ExecContext(ctx, "COMMIT"); err != nil {
		return fail(err)
	}
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
	if l == nil || l.db == nil || key == "" {
		return "", domain.ErrLifecycle
	}
	var replayState domain.IdempotencyReplayState
	err := l.db.QueryRowContext(ctx, "SELECT replay_state FROM idempotency WHERE key=?", key).Scan(&replayState)
	if err == nil && replayState == domain.IdempotencyLegacyUnavailable {
		return "", domain.ErrLegacyIdempotencyReplayUnavailable
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return "", domain.ErrLifecycle
	}
	if key != p.IdempotencyKey || result == "" || p.Validate() != nil {
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
	var recordID sql.NullInt64
	err = c.QueryRowContext(ctx, "SELECT identity,result,record_id FROM idempotency WHERE key=?", key).Scan(&storedID, &storedResult, &recordID)
	if err == nil {
		if !recordID.Valid || storedID != id {
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
	newRecordID, err := r.LastInsertId()
	if err != nil {
		return "", domain.ErrLifecycle
	}
	if _, err = c.ExecContext(ctx, `INSERT INTO provenance VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, newRecordID, p.Authority, p.DeclaredActor, p.Interaction, p.Context, p.Operation, p.Request, p.IdempotencyKey, p.Time, p.Approval, p.Evidence, p.Input, p.Result, p.Versions); err != nil {
		return "", domain.ErrLifecycle
	}
	if _, err = c.ExecContext(ctx, "INSERT INTO idempotency VALUES(?,?,?,?,?,?)", key, p.Input, id, result, newRecordID, domain.IdempotencyAvailable); err != nil {
		return "", domain.ErrLifecycle
	}
	if _, err = c.ExecContext(ctx, "COMMIT"); err != nil {
		return "", domain.ErrLifecycle
	}
	committed = true
	return result, nil
}

func (l *Lifecycle) SaveCatalog(ctx context.Context, key string, write domain.CatalogWrite) (string, error) {
	if l == nil || l.db == nil || key == "" || key != write.Provenance.IdempotencyKey || write.Request == "" || write.Result == "" || write.Provenance.Validate() != nil || len(write.Entries) == 0 || len(write.Entries) != len(write.Imports) || write.Policy.Revision == "" || write.Policy.Mode == "" || !write.Policy.Approved {
		return "", domain.ErrLifecycle
	}
	for i := range write.Entries {
		entry, imported := write.Entries[i], write.Imports[i]
		if entry.Path == "" || entry.Purpose == "" || entry.Audience == "" || entry.Owner == "" || len(entry.RelatedAreas) == 0 || entry.State == "" || entry.Evidence == "" || imported.Path != entry.Path || imported.Digest == "" || imported.Provenance.Validate() != nil || imported.Provenance.IdempotencyKey != key {
			return "", domain.ErrLifecycle
		}
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
	digest := sha256.Sum256([]byte(write.Request))
	id := "catalog-v3:" + hex.EncodeToString(digest[:])
	var storedID, storedResult, replayState string
	var recordID sql.NullInt64
	err = c.QueryRowContext(ctx, `SELECT identity,result,record_id,replay_state FROM idempotency WHERE key=?`, key).Scan(&storedID, &storedResult, &recordID, &replayState)
	if err == nil {
		if replayState == string(domain.IdempotencyLegacyUnavailable) {
			return "", domain.ErrLegacyIdempotencyReplayUnavailable
		}
		if !recordID.Valid || storedID != id {
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
	r, err := c.ExecContext(ctx, `INSERT INTO lifecycle_records(state) VALUES(?)`, write.Result)
	if err != nil {
		return "", domain.ErrLifecycle
	}
	newRecordID, err := r.LastInsertId()
	if err != nil {
		return "", domain.ErrLifecycle
	}
	p := write.Provenance
	if _, err = c.ExecContext(ctx, `INSERT INTO provenance VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, newRecordID, p.Authority, p.DeclaredActor, p.Interaction, p.Context, p.Operation, p.Request, p.IdempotencyKey, p.Time, p.Approval, p.Evidence, p.Input, p.Result, p.Versions); err != nil {
		return "", domain.ErrLifecycle
	}
	if _, err = c.ExecContext(ctx, `INSERT INTO idempotency VALUES(?,?,?,?,?,?)`, key, write.Request, id, write.Result, newRecordID, domain.IdempotencyAvailable); err != nil {
		return "", domain.ErrLifecycle
	}
	for i, entry := range write.Entries {
		related, _ := json.Marshal(entry.RelatedAreas)
		pending, _ := json.Marshal(entry.PendingActions)
		if _, err = c.ExecContext(ctx, `INSERT INTO catalog_entries VALUES(?,?,?,?,?,?,?,?,?,?)`, entry.Path, entry.Purpose, entry.Audience, entry.Owner, related, entry.State, entry.Evidence, entry.LastVerification, pending, newRecordID); err != nil {
			return "", domain.ErrLifecycle
		}
		imported, ip := write.Imports[i], write.Imports[i].Provenance
		if _, err = c.ExecContext(ctx, `INSERT INTO catalog_imports VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, newRecordID, imported.Path, imported.Digest, ip.Authority, ip.DeclaredActor, ip.Interaction, ip.Context, ip.Operation, ip.Request, ip.IdempotencyKey, ip.Time, ip.Approval, ip.Evidence, ip.Input, ip.Result, ip.Versions); err != nil {
			return "", domain.ErrLifecycle
		}
	}
	if _, err = c.ExecContext(ctx, `INSERT INTO initial_policy VALUES(?,?,?,?)`, write.Policy.Revision, write.Policy.Mode, write.Policy.Approved, newRecordID); err != nil {
		return "", domain.ErrLifecycle
	}
	if _, err = c.ExecContext(ctx, `INSERT INTO catalog_policy_current VALUES(1,?)`, write.Policy.Revision); err != nil {
		return "", domain.ErrLifecycle
	}
	if _, err = c.ExecContext(ctx, "COMMIT"); err != nil {
		return "", domain.ErrLifecycle
	}
	committed = true
	return write.Result, nil
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
