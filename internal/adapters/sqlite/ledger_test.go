package sqlite

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/desatatufuria/mcp-doc-manager/internal/domain"
)

func TestLedgerSavesRebuildsAndVerifiesReceipts(t *testing.T) {
	root := t.TempDir()
	receipt := domain.Receipt{
		Digest: "sha256:receipt", Scope: domain.Scope{Kind: domain.ScopeRange, Range: "a..b"}, Evidence: "sha256:evidence",
		ChangedPaths: []string{"README.md", "docs/guide.md"}, Documentation: []domain.DocumentationBlob{{Path: "README.md", Digest: "sha256:readme"}},
		Outcome: domain.OutcomeUpdate, Versions: domain.ReceiptVersions{Schema: "receipt/v1", Tool: "docmanager/1"},
	}
	report := domain.Report{Outcome: domain.OutcomeUpdate, Receipt: receipt}
	ledger, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := ledger.Save(context.Background(), report); err != nil {
		t.Fatal(err)
	}
	if err := ledger.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, ".docmanager", "ledger.db")); err != nil {
		t.Fatal(err)
	}
	rebuilt, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	defer rebuilt.Close()
	if err := rebuilt.Verify(context.Background(), receipt); err != nil {
		t.Fatal(err)
	}
	stored, err := rebuilt.Receipt(context.Background(), receipt.Digest)
	if err != nil || !reflect.DeepEqual(stored, receipt) {
		t.Fatalf("Receipt() = %#v, %v", stored, err)
	}
	if err := rebuilt.Verify(context.Background(), domain.Receipt{Digest: "sha256:other"}); !errors.Is(err, domain.ErrReceiptMismatch) {
		t.Fatalf("Verify() error = %v", err)
	}
}

func TestLedgerDeleteAndRebuildsFromAuthoritativeReceipt(t *testing.T) {
	root := t.TempDir()
	report := domain.Report{Outcome: domain.OutcomeUpdate, Receipt: domain.Receipt{Digest: "sha256:receipt", Outcome: domain.OutcomeUpdate, Scope: domain.Scope{Kind: domain.ScopeStaged}}}
	ledger, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := ledger.Save(context.Background(), report); err != nil {
		t.Fatal(err)
	}
	if err := ledger.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(root, ".docmanager")); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenExisting(root); !errors.Is(err, domain.ErrLedgerFailure) {
		t.Fatalf("missing ledger = %v", err)
	}
	ledger, err = Open(root)
	if err != nil {
		t.Fatal(err)
	}
	defer ledger.Close()
	if err := ledger.Save(context.Background(), report); err != nil {
		t.Fatal(err)
	}
	if err := ledger.Verify(context.Background(), report.Receipt); err != nil {
		t.Fatal(err)
	}
}

func TestLedgerClassifiesInvalidRoot(t *testing.T) {
	if _, err := Open(filepath.Join(t.TempDir(), "missing")); !errors.Is(err, domain.ErrLedgerFailure) {
		t.Fatalf("Open() error = %v", err)
	}
}

func TestLedgerRejectsUnsafeStateAndOperationalFailures(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, ".docmanager")); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(root); !errors.Is(err, domain.ErrLedgerFailure) {
		t.Fatalf("symlink Open() = %v", err)
	}
	if _, err := os.Stat(filepath.Join(outside, "ledger.db")); !os.IsNotExist(err) {
		t.Fatalf("outside ledger created: %v", err)
	}
	if err := os.Remove(filepath.Join(root, ".docmanager")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".docmanager"), []byte("not a directory"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(root); !errors.Is(err, domain.ErrLedgerFailure) {
		t.Fatalf("file Open() = %v", err)
	}

	ledger, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := ledger.Close(); err != nil {
		t.Fatal(err)
	}
	receipt := domain.Receipt{Digest: "sha256:closed"}
	if err := ledger.Save(context.Background(), domain.Report{Receipt: receipt}); !errors.Is(err, domain.ErrLedgerFailure) {
		t.Fatalf("closed save = %v", err)
	}
	if err := ledger.Verify(context.Background(), receipt); !errors.Is(err, domain.ErrLedgerFailure) {
		t.Fatalf("closed verify = %v", err)
	}

	brokenRoot := t.TempDir()
	brokenDir := filepath.Join(brokenRoot, ".docmanager")
	if err := os.Mkdir(brokenDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(brokenDir, "ledger.db"), []byte("corrupt"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenExisting(brokenRoot); !errors.Is(err, domain.ErrLedgerFailure) {
		t.Fatalf("corrupt open = %v", err)
	}

	queryLedger, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer queryLedger.Close()
	if _, err := queryLedger.db.Exec(`DROP TABLE receipts`); err != nil {
		t.Fatal(err)
	}
	if err := queryLedger.Verify(context.Background(), receipt); !errors.Is(err, domain.ErrLedgerFailure) {
		t.Fatalf("query verify = %v", err)
	}
	if err := queryLedger.Save(context.Background(), domain.Report{Receipt: receipt}); !errors.Is(err, domain.ErrLedgerFailure) {
		t.Fatalf("query save = %v", err)
	}
}
