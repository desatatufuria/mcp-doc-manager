package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/gentleman-programming/repository-documentation-manager/internal/domain"
	_ "modernc.org/sqlite"
)

type Ledger struct{ db *sql.DB }

func Open(root string) (*Ledger, error) {
	info, err := os.Lstat(root)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, domain.ErrLedgerFailure
	}
	dir := filepath.Join(root, ".docmanager")
	if info, err := os.Lstat(dir); err == nil {
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return nil, domain.ErrLedgerFailure
		}
	} else if os.IsNotExist(err) {
		if err := os.Mkdir(dir, 0o700); err != nil {
			return nil, domain.ErrLedgerFailure
		}
	} else {
		return nil, domain.ErrLedgerFailure
	}
	db, err := sql.Open("sqlite", filepath.Join(dir, "ledger.db"))
	if err != nil {
		return nil, domain.ErrLedgerFailure
	}
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS receipts (digest TEXT PRIMARY KEY, receipt TEXT NOT NULL)`); err != nil {
		db.Close()
		return nil, domain.ErrLedgerFailure
	}
	return &Ledger{db: db}, nil
}

func OpenExisting(root string) (*Ledger, error) {
	info, err := os.Lstat(root)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, domain.ErrLedgerFailure
	}
	dir := filepath.Join(root, ".docmanager")
	if info, err := os.Lstat(dir); err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, domain.ErrLedgerFailure
	}
	path := filepath.Join(dir, "ledger.db")
	if info, err := os.Lstat(path); err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return nil, domain.ErrLedgerFailure
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, domain.ErrLedgerFailure
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, domain.ErrLedgerFailure
	}
	return &Ledger{db: db}, nil
}

func (l *Ledger) Save(ctx context.Context, report domain.Report) error {
	if l == nil || l.db == nil || report.Receipt.Digest == "" {
		return domain.ErrLedgerFailure
	}
	raw, err := json.Marshal(report.Receipt)
	if err != nil {
		return domain.ErrLedgerFailure
	}
	_, err = l.db.ExecContext(ctx, `INSERT OR REPLACE INTO receipts(digest, receipt) VALUES (?, ?)`, report.Receipt.Digest, raw)
	if err != nil {
		return domain.ErrLedgerFailure
	}
	return nil
}

func (l *Ledger) Verify(ctx context.Context, receipt domain.Receipt) error {
	if l == nil || l.db == nil {
		return domain.ErrLedgerFailure
	}
	stored, err := l.Receipt(ctx, receipt.Digest)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ErrReceiptMismatch
		}
		return domain.ErrLedgerFailure
	}
	storedRaw, _ := json.Marshal(stored)
	receiptRaw, _ := json.Marshal(receipt)
	if string(storedRaw) != string(receiptRaw) {
		return domain.ErrReceiptMismatch
	}
	return nil
}

func (l *Ledger) Receipt(ctx context.Context, digest string) (domain.Receipt, error) {
	if l == nil || l.db == nil {
		return domain.Receipt{}, domain.ErrLedgerFailure
	}
	var raw []byte
	if err := l.db.QueryRowContext(ctx, `SELECT receipt FROM receipts WHERE digest = ?`, digest).Scan(&raw); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Receipt{}, err
		}
		return domain.Receipt{}, domain.ErrLedgerFailure
	}
	var receipt domain.Receipt
	if err := json.Unmarshal(raw, &receipt); err != nil {
		return domain.Receipt{}, domain.ErrLedgerFailure
	}
	return receipt, nil
}

// Receipts returns a bounded snapshot of stored receipts for hook verification.
func (l *Ledger) Receipts(ctx context.Context) ([]domain.Receipt, error) {
	if l == nil || l.db == nil {
		return nil, domain.ErrLedgerFailure
	}
	rows, err := l.db.QueryContext(ctx, `SELECT receipt FROM receipts LIMIT 65`)
	if err != nil {
		return nil, domain.ErrLedgerFailure
	}
	defer rows.Close()
	var receipts []domain.Receipt
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			return nil, domain.ErrLedgerFailure
		}
		var receipt domain.Receipt
		if err := json.Unmarshal(raw, &receipt); err != nil {
			return nil, domain.ErrLedgerFailure
		}
		receipts = append(receipts, receipt)
		if len(receipts) > 64 {
			return nil, domain.ErrLedgerFailure
		}
	}
	if err := rows.Err(); err != nil {
		return nil, domain.ErrLedgerFailure
	}
	return receipts, nil
}

func (l *Ledger) Close() error {
	if l == nil || l.db == nil {
		return nil
	}
	return l.db.Close()
}
