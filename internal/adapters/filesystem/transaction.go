// Package filesystem provides small, ownership-aware atomic file transactions.
package filesystem

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

var (
	ErrLocked     = errors.New("docmanager transaction is locked")
	ErrDrift      = errors.New("docmanager owned file drifted")
	ErrOwnership  = errors.New("docmanager file is not owned")
	ErrUnsafePath = errors.New("unsafe transaction path")
)

// Ownership is the exact pre-mutation identity required before replacement.
type Ownership struct {
	Digest string
	Mode   fs.FileMode
}

type Transaction struct {
	root string
}

func Open(root string) (*Transaction, error) {
	info, err := os.Lstat(root)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, ErrUnsafePath
	}
	return &Transaction{root: root}, nil
}

// Lock serializes mutations. Its release function must always be called.
func (t *Transaction) Lock() (func(), error) {
	path := filepath.Join(t.root, ".docmanager.lock")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if os.IsExist(err) {
		return nil, ErrLocked
	}
	if err != nil {
		return nil, err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		_ = os.Remove(path)
		return nil, err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(path)
		return nil, err
	}
	return func() { _ = os.Remove(path) }, nil
}

func (t *Transaction) Ownership(path string) (Ownership, error) {
	full, err := t.path(path)
	if err != nil {
		return Ownership{}, err
	}
	return ownership(full)
}

// WriteOwned atomically writes content only when the recorded owner still matches.
// A nil previous value permits creation only; it never claims an existing file.
func (t *Transaction) WriteOwned(path string, content []byte, mode fs.FileMode, previous *Ownership) (Ownership, error) {
	full, err := t.path(path)
	if err != nil {
		return Ownership{}, err
	}
	actual, err := ownership(full)
	if err == nil {
		if previous == nil {
			return Ownership{}, ErrOwnership
		}
		if actual != *previous {
			return Ownership{}, ErrDrift
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		return Ownership{}, err
	} else if previous != nil {
		return Ownership{}, ErrDrift
	}

	tmp, err := os.CreateTemp(t.root, "."+filepath.Base(full)+".*")
	if err != nil {
		return Ownership{}, err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(content); err != nil {
		_ = tmp.Close()
		return Ownership{}, err
	}
	if err := tmp.Chmod(mode); err != nil {
		_ = tmp.Close()
		return Ownership{}, err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return Ownership{}, err
	}
	if err := tmp.Close(); err != nil {
		return Ownership{}, err
	}
	if err := os.Rename(tmpName, full); err != nil {
		return Ownership{}, err
	}
	if err := syncDir(t.root); err != nil {
		return Ownership{}, err
	}
	return ownership(full)
}

func (t *Transaction) path(path string) (string, error) {
	if path == "" || filepath.IsAbs(path) || strings.Contains(path, "\\") {
		return "", ErrUnsafePath
	}
	clean := filepath.Clean(path)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) || filepath.Dir(clean) != "." {
		return "", ErrUnsafePath
	}
	return filepath.Join(t.root, clean), nil
}

func ownership(path string) (Ownership, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return Ownership{}, err
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return Ownership{}, ErrUnsafePath
	}
	f, err := os.Open(path)
	if err != nil {
		return Ownership{}, err
	}
	defer f.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, f); err != nil {
		return Ownership{}, err
	}
	return Ownership{Digest: fmt.Sprintf("%x", hash.Sum(nil)), Mode: info.Mode().Perm()}, nil
}

func syncDir(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}
