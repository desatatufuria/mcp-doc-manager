package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"reflect"
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
	if got, err := l.Version(context.Background()); err != nil || got != 4 {
		t.Fatalf("Version() = %d, %v", got, err)
	}
	for _, table := range []string{"catalog_entries", "catalog_imports", "initial_policy", "catalog_policy_current"} {
		var n int
		if err := l.db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&n); err != nil || n != 1 {
			t.Fatalf("v3 table %q = %d, %v", table, n, err)
		}
	}
	var finalColumns int
	if err := l.db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('idempotency') WHERE name IN ('input','identity','record_id','replay_state')`).Scan(&finalColumns); err != nil || finalColumns != 4 {
		t.Fatalf("fresh v2 idempotency schema columns = %d, %v", finalColumns, err)
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
	if got, err := l.Version(context.Background()); err != nil || got != 4 {
		t.Fatalf("forward migration = %d, %v", got, err)
	}
	if err := l.migrateOne(context.Background(), 5, `CREATE TABLE rejected(`); !errors.Is(err, domain.ErrLifecycle) {
		t.Fatalf("failed migration = %v", err)
	}
	var n int
	if err := l.db.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE version=5`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("rollback version = %d, %v", n, err)
	}
	if err := l.db.QueryRow(`SELECT COUNT(*) FROM lifecycle_records`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("rollback schema = %d, %v", n, err)
	}
}

// createHistoricalV1 recreates the schema shipped by PR #3 (0fbd204), not a
// reduced approximation: lifecycle_records plus the original provenance and
// idempotency tables were already present before v2.
func createHistoricalV1(t *testing.T, root string) *sql.DB {
	t.Helper()
	dir := filepath.Join(root, ".docmanager")
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", filepath.Join(dir, "lifecycle.db"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`CREATE TABLE schema_migrations(version INTEGER PRIMARY KEY);
		CREATE TABLE lifecycle_records(id INTEGER PRIMARY KEY, state TEXT NOT NULL);
		CREATE TABLE provenance(record_id INTEGER NOT NULL, actor TEXT NOT NULL CHECK(actor<>''), interaction TEXT NOT NULL, context TEXT NOT NULL);
		CREATE TABLE idempotency(key TEXT PRIMARY KEY, input TEXT NOT NULL, result TEXT NOT NULL);
		INSERT INTO schema_migrations VALUES(1);
		INSERT INTO lifecycle_records VALUES(7, 'approved'), (8, 'rejected');
		INSERT INTO provenance VALUES(7, 'local-user', 'mcp', 'request'), (8, 'reviewer', 'cli', 'batch');
		INSERT INTO idempotency VALUES('legacy-key-a', 'input-a', 'result-a'), ('legacy-key-b', 'input-b', 'result-b')`)
	if err != nil {
		db.Close()
		t.Fatal(err)
	}
	return db
}
func TestLifecycleMigratesCompleteHistoricalV1(t *testing.T) {
	root := t.TempDir()
	db := createHistoricalV1(t, root)
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	l, err := OpenLifecycle(root)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	if got, err := l.Version(context.Background()); err != nil || got != 4 {
		t.Fatalf("version = %d, %v", got, err)
	}
	if got, err := l.Count(context.Background()); err != nil || got != 2 {
		t.Fatalf("legacy records = %d, %v", got, err)
	}
	for _, want := range []struct {
		recordID                    int
		actor, interaction, context string
	}{{7, "local-user", "mcp", "request"}, {8, "reviewer", "cli", "batch"}} {
		var got struct{ actor, interaction, context string }
		if err := l.db.QueryRow(`SELECT actor,interaction,context FROM provenance WHERE record_id=?`, want.recordID).Scan(&got.actor, &got.interaction, &got.context); err != nil || got.actor != want.actor || got.interaction != want.interaction || got.context != want.context {
			t.Fatalf("legacy provenance %d = %#v, %v", want.recordID, got, err)
		}
	}
	for _, want := range []struct{ key, input, result string }{{"legacy-key-a", "input-a", "result-a"}, {"legacy-key-b", "input-b", "result-b"}} {
		var recordID sql.NullInt64
		var state, input, result string
		if err := l.db.QueryRow(`SELECT input,result,record_id,replay_state FROM idempotency WHERE key=?`, want.key).Scan(&input, &result, &recordID, &state); err != nil || input != want.input || result != want.result || recordID.Valid || state != "legacy_unavailable" {
			t.Fatalf("legacy idempotency %q = %q/%q record:%v state:%q err:%v", want.key, input, result, recordID, state, err)
		}
	}
}

func TestLifecycleSaveCatalogIsAtomicReplayableAndConflictSafe(t *testing.T) {
	l, err := OpenLifecycle(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	p := provenance("canonical-request")
	p.IdempotencyKey, p.Operation = "catalog-1", "catalog_import"
	entry := domain.CatalogEntry{Path: "docs/a.md", Purpose: "guide", Audience: "contributors", Owner: "docs-team", RelatedAreas: []string{"internal/app"}, State: domain.CatalogImported, Evidence: "tracked"}
	write := domain.CatalogWrite{Request: `{"plan":"one"}`, Result: `{"stored":"exact"}`, Provenance: p, Entries: []domain.CatalogEntry{entry}, Imports: []domain.CatalogImport{{Path: entry.Path, Digest: "sha256:a", Provenance: p}}, Policy: domain.InitialPolicy{Revision: "policy:request-1", Mode: "approval-required", Approved: true}}
	first, err := l.SaveCatalog(context.Background(), "catalog-1", write)
	if err != nil || first != write.Result {
		t.Fatalf("SaveCatalog() = %q, %v", first, err)
	}
	before := snapshotV3(t, l.db)
	if before != "1|1|1|1|1|1|1" {
		t.Fatalf("atomic table counts = %q", before)
	}
	var purpose, audience, owner, related, state, evidence, digest, importEvidence, revision, mode string
	if err := l.db.QueryRow(`SELECT e.purpose,e.audience,e.owner,e.related_areas,e.state,e.evidence,i.digest,i.evidence,p.revision,p.mode FROM catalog_entries e JOIN catalog_imports i USING(path) JOIN initial_policy p ON p.record_id=e.record_id JOIN catalog_policy_current c ON c.revision=p.revision`).Scan(&purpose, &audience, &owner, &related, &state, &evidence, &digest, &importEvidence, &revision, &mode); err != nil {
		t.Fatal(err)
	}
	if purpose != "guide" || audience != "contributors" || owner != "docs-team" || related != `["internal/app"]` || state != "imported" || evidence != "tracked" || digest != "sha256:a" || importEvidence != p.Evidence || revision != "policy:request-1" || mode != "approval-required" {
		t.Fatalf("durable catalog fields = %q/%q/%q/%q/%q/%q/%q/%q/%q/%q", purpose, audience, owner, related, state, evidence, digest, importEvidence, revision, mode)
	}
	if _, err := l.db.Exec(`UPDATE initial_policy SET mode='changed'`); err == nil {
		t.Fatal("initial policy update succeeded")
	}
	replay, err := l.SaveCatalog(context.Background(), "catalog-1", write)
	if err != nil || replay != first || snapshotV3(t, l.db) != before {
		t.Fatalf("replay = %q, %v", replay, err)
	}
	write.Request = `{"plan":"divergent"}`
	if _, err := l.SaveCatalog(context.Background(), "catalog-1", write); !errors.Is(err, domain.ErrIdempotencyConflict) || snapshotV3(t, l.db) != before {
		t.Fatalf("conflict = %v", err)
	}

	p.IdempotencyKey = "catalog-rollback"
	write.Request, write.Provenance = `{"plan":"rollback"}`, p
	write.Imports[0].Provenance = p
	if _, err := l.db.Exec(`CREATE TRIGGER fail_catalog_import BEFORE INSERT ON catalog_imports WHEN NEW.idempotency_key='catalog-rollback' BEGIN SELECT RAISE(ABORT, 'stop'); END`); err != nil {
		t.Fatal(err)
	}
	if _, err := l.SaveCatalog(context.Background(), "catalog-rollback", write); !errors.Is(err, domain.ErrLifecycle) || snapshotV3(t, l.db) != before {
		t.Fatalf("rollback = %v", err)
	}
	entries, err := l.LoadCatalog(context.Background())
	if err != nil || len(entries) != 1 {
		t.Fatalf("LoadCatalog() = %#v, %v", entries, err)
	}
	ap := provenance("audit-input")
	ap.IdempotencyKey, ap.Operation = "audit-1", "catalog_audit"
	audit := domain.Audit{ID: "audit-id-1", Trigger: "change", Path: entry.Path, Scope: `{"kind":"worktree"}`, State: domain.AuditReported, Result: domain.AuditOrphan, Evidence: "evidence-1", Rationale: "missing"}
	action := domain.PendingAction{ID: "action-id", Path: entry.Path, AuditID: audit.ID, State: domain.PendingOpen, Evidence: audit.Evidence}
	aw := domain.CatalogAuditWrite{Request: `{"audit":"one"}`, Result: `{"stored":"audit"}`, Provenance: ap, Entries: entries, Audits: []domain.Audit{audit}, Actions: []domain.PendingAction{action}}
	first, err = l.SaveAudit(context.Background(), "audit-1", aw)
	replay, replayErr := l.SaveAudit(context.Background(), "audit-1", aw)
	if err != nil || replayErr != nil || first != aw.Result || replay != first {
		t.Fatalf("SaveAudit/replay = %q/%q, %v/%v", first, replay, err, replayErr)
	}
	aw.Request = `{"audit":"divergent"}`
	if _, err := l.SaveAudit(context.Background(), "audit-1", aw); !errors.Is(err, domain.ErrIdempotencyConflict) {
		t.Fatalf("audit conflict = %v", err)
	}
	if err := l.db.QueryRow(`SELECT (SELECT COUNT(*) FROM catalog_audits)||'|'||(SELECT COUNT(*) FROM orphan_actions)`).Scan(&first); err != nil || first != "1|1" {
		t.Fatalf("audit/action counts = %q, %v", first, err)
	}
	if _, err := l.db.Exec(`CREATE TRIGGER fail_orphan BEFORE INSERT ON orphan_actions BEGIN SELECT RAISE(ABORT,'stop'); END`); err != nil {
		t.Fatal(err)
	}
	aw.Request, aw.Provenance.IdempotencyKey, aw.Audits[0].ID, aw.Actions[0].ID = `{"audit":"fail"}`, "audit-fail", "audit-id-fail", "action-id-fail"
	if _, err := l.SaveAudit(context.Background(), "audit-fail", aw); !errors.Is(err, domain.ErrLifecycle) || snapshotV3(t, l.db) != "2|2|2|1|1|1|1" {
		t.Fatalf("audit rollback = %v", err)
	}
}

func snapshotV3(t *testing.T, db *sql.DB) string {
	t.Helper()
	var got string
	if err := db.QueryRow(`SELECT (SELECT COUNT(*) FROM lifecycle_records)||'|'||(SELECT COUNT(*) FROM provenance)||'|'||(SELECT COUNT(*) FROM idempotency)||'|'||(SELECT COUNT(*) FROM catalog_entries)||'|'||(SELECT COUNT(*) FROM catalog_imports)||'|'||(SELECT COUNT(*) FROM initial_policy)||'|'||(SELECT COUNT(*) FROM catalog_policy_current)`).Scan(&got); err != nil {
		t.Fatal(err)
	}
	return got
}

type v1Snapshot struct {
	schema map[string]string
	rows   map[string][]string
	counts map[string]int
}

func snapshotV1(t *testing.T, db *sql.DB) v1Snapshot {
	t.Helper()
	s := v1Snapshot{schema: map[string]string{}, rows: map[string][]string{}}
	for table, query := range map[string]string{
		"schema_migrations": `SELECT quote(version) FROM schema_migrations ORDER BY version`,
		"lifecycle_records": `SELECT quote(id)||'|'||quote(state) FROM lifecycle_records ORDER BY id`,
		"provenance":        `SELECT quote(record_id)||'|'||quote(actor)||'|'||quote(interaction)||'|'||quote(context) FROM provenance ORDER BY record_id`,
		"idempotency":       `SELECT quote(key)||'|'||quote(input)||'|'||quote(result) FROM idempotency ORDER BY key`,
	} {
		var schema string
		if err := db.QueryRow(`SELECT sql FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&schema); err != nil {
			t.Fatal(err)
		}
		s.schema[table] = schema
		rows, err := db.Query(query)
		if err != nil {
			t.Fatal(err)
		}
		for rows.Next() {
			var row string
			if err := rows.Scan(&row); err != nil {
				rows.Close()
				t.Fatal(err)
			}
			s.rows[table] = append(s.rows[table], row)
		}
		if err := rows.Close(); err != nil {
			t.Fatal(err)
		}
	}
	return s
}

func TestLifecycleV2MigrationRollsBackHistoricalSchemaAndData(t *testing.T) {
	root := t.TempDir()
	db := createHistoricalV1(t, root)
	before := snapshotV1(t, db)
	if _, err := db.Exec(`CREATE TRIGGER fail_v2_version BEFORE INSERT ON schema_migrations
		WHEN NEW.version = 2 BEGIN SELECT RAISE(ABORT, 'stop v2'); END`); err != nil {
		db.Close()
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenLifecycle(root); !errors.Is(err, domain.ErrLifecycle) {
		t.Fatalf("migration failure = %v", err)
	}
	db, err := sql.Open("sqlite", filepath.Join(root, ".docmanager", "lifecycle.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if after := snapshotV1(t, db); !reflect.DeepEqual(after, before) {
		t.Fatalf("migration rollback changed v1 snapshot: got %#v want %#v", after, before)
	}
}

func TestLifecycleV3MigrationFailurePreservesV2Exactly(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, ".docmanager")
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", filepath.Join(dir, "lifecycle.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	l := &Lifecycle{db: db}
	if _, err := db.Exec(`CREATE TABLE schema_migrations(version INTEGER PRIMARY KEY)`); err != nil {
		t.Fatal(err)
	}
	if err := l.migrateOne(context.Background(), 1, `CREATE TABLE lifecycle_records(id INTEGER PRIMARY KEY, state TEXT NOT NULL)`); err != nil {
		t.Fatal(err)
	}
	if err := l.migrateV2(context.Background()); err != nil {
		t.Fatal(err)
	}
	p := provenance("v2-data")
	if _, err := l.Save(context.Background(), "key-1", "v2-result", p); err != nil {
		t.Fatal(err)
	}
	before := snapshotV2(t, db)
	if _, err := db.Exec(`CREATE TRIGGER fail_v3_version BEFORE INSERT ON schema_migrations WHEN NEW.version=3 BEGIN SELECT RAISE(ABORT, 'stop v3'); END`); err != nil {
		t.Fatal(err)
	}
	if err := l.migrate(context.Background()); !errors.Is(err, domain.ErrLifecycle) {
		t.Fatalf("migration failure = %v", err)
	}
	if after := snapshotV2(t, db); !reflect.DeepEqual(after, before) {
		t.Fatalf("v3 failure changed v2 snapshot: got %#v want %#v", after, before)
	}
	if _, err := db.Exec(`DROP TRIGGER fail_v3_version; ` + v3Schema + `; INSERT INTO schema_migrations VALUES(3); INSERT INTO catalog_entries VALUES('docs/v3.md','guide','contributors','docs','["docs"]','active','tracked','','[]',1); INSERT INTO catalog_imports VALUES(1,'docs/v3.md','sha256:v3','','','','','','','','','','','','','')`); err != nil {
		t.Fatal(err)
	}
	if err := l.migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	entries, err := l.LoadCatalog(context.Background())
	if err != nil || len(entries) != 1 || entries[0].Digest != "sha256:v3" {
		t.Fatalf("migrated v3 catalog = %#v, %v", entries, err)
	}
}

func TestLifecycleRefusesEveryMigratedLegacyKeyWithoutMutation(t *testing.T) {
	root := t.TempDir()
	db := createHistoricalV1(t, root)
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	l, err := OpenLifecycle(root)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	before := snapshotV2(t, l.db)
	matchingA := provenance("input-a")
	matchingA.IdempotencyKey = "legacy-key-a"
	matchingB := provenance("input-b")
	matchingB.IdempotencyKey = "legacy-key-b"
	for _, tt := range []struct {
		name string
		key  string
		p    domain.Provenance
	}{
		{"matching provenance", "legacy-key-a", matchingA},
		{"repeated matching provenance", "legacy-key-a", matchingA},
		{"every migrated key", "legacy-key-b", matchingB},
		{"malformed provenance", "legacy-key-b", domain.Provenance{}},
		{"mismatched provenance key", "legacy-key-b", provenance("input-b")},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got, err := l.Save(context.Background(), tt.key, "new-result", tt.p); got != "" || !errors.Is(err, domain.ErrLegacyIdempotencyReplayUnavailable) {
				t.Fatalf("Save() = %q, %v", got, err)
			}
			if after := snapshotV2(t, l.db); !reflect.DeepEqual(after, before) {
				t.Fatalf("legacy refusal mutated database: got %#v want %#v", after, before)
			}
		})
	}
}

func TestLifecycleReplaysAvailableV2KeyExactlyOnce(t *testing.T) {
	l, err := OpenLifecycle(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	p := provenance("v2-input")
	p.IdempotencyKey = "v2-key"
	first, err := l.Save(context.Background(), "v2-key", "v2-result", p)
	if err != nil || first != "v2-result" {
		t.Fatalf("first Save() = %q, %v", first, err)
	}
	before := snapshotV2(t, l.db)
	replay, err := l.Save(context.Background(), "v2-key", "ignored-result", p)
	if err != nil || replay != first {
		t.Fatalf("replay Save() = %q, %v", replay, err)
	}
	if after := snapshotV2(t, l.db); !reflect.DeepEqual(after, before) {
		t.Fatalf("available replay changed database: got %#v want %#v", after, before)
	}
	var recordID sql.NullInt64
	var state string
	if err := l.db.QueryRow(`SELECT record_id,replay_state FROM idempotency WHERE key='v2-key'`).Scan(&recordID, &state); err != nil || !recordID.Valid || state != string(domain.IdempotencyAvailable) {
		t.Fatalf("v2 idempotency link/state = %v/%q, %v", recordID, state, err)
	}
}

func snapshotV2(t *testing.T, db *sql.DB) v1Snapshot {
	t.Helper()
	s := v1Snapshot{schema: map[string]string{}, rows: map[string][]string{}, counts: map[string]int{}}
	for table, query := range map[string]string{
		"schema_migrations": `SELECT quote(version) FROM schema_migrations ORDER BY version`,
		"lifecycle_records": `SELECT quote(id)||'|'||quote(state) FROM lifecycle_records ORDER BY id`,
		"provenance":        `SELECT quote(record_id)||'|'||quote(authority)||'|'||quote(actor)||'|'||quote(interaction)||'|'||quote(context)||'|'||quote(operation)||'|'||quote(request)||'|'||quote(idempotency_key)||'|'||quote(time)||'|'||quote(approval)||'|'||quote(evidence)||'|'||quote(input)||'|'||quote(result)||'|'||quote(versions) FROM provenance ORDER BY record_id`,
		"idempotency":       `SELECT quote(key)||'|'||quote(input)||'|'||quote(identity)||'|'||quote(result)||'|'||quote(record_id)||'|'||quote(replay_state) FROM idempotency ORDER BY key`,
	} {
		var schema string
		if err := db.QueryRow(`SELECT sql FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&schema); err != nil {
			t.Fatal(err)
		}
		s.schema[table] = schema
		var count int
		if err := db.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&count); err != nil {
			t.Fatal(err)
		}
		s.counts[table] = count
		rows, err := db.Query(query)
		if err != nil {
			t.Fatal(err)
		}
		for rows.Next() {
			var row string
			if err := rows.Scan(&row); err != nil {
				rows.Close()
				t.Fatal(err)
			}
			s.rows[table] = append(s.rows[table], row)
		}
		if err := rows.Close(); err != nil {
			t.Fatal(err)
		}
	}
	return s
}
