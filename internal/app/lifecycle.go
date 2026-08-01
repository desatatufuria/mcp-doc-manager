package app

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/desatatufuria/mcp-doc-manager/assets"
	"github.com/desatatufuria/mcp-doc-manager/internal/domain"
)

var writeOwnership = os.WriteFile

const ownershipMarker = "docmanager/1\n"

func Install(target string) error {
	root, err := lifecycleRoot(target)
	if err != nil {
		return err
	}
	dir := filepath.Join(root, ".docmanager")
	if _, err := os.Lstat(dir); err == nil {
		if err := ownedState(dir); err != nil {
			return err
		}
		return writeAssets(dir)
	} else if !os.IsNotExist(err) {
		return domain.ErrInvalidTarget
	}
	if err := os.Mkdir(dir, 0o700); err != nil {
		return domain.ErrLedgerFailure
	}
	if err := writeOwnership(filepath.Join(dir, ".owned"), []byte(ownershipMarker), 0o600); err != nil {
		_ = os.RemoveAll(dir)
		return domain.ErrLedgerFailure
	}
	if err := writeAssets(dir); err != nil {
		_ = os.RemoveAll(dir)
		return err
	}
	return nil
}

func writeAssets(dir string) error {
	for path, payload := range assets.Files {
		target := filepath.Join(dir, path)
		if filepath.Clean(target) == dir || !strings.HasPrefix(filepath.Clean(target), dir+string(filepath.Separator)) {
			return domain.ErrInvalidTarget
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
			return domain.ErrLedgerFailure
		}
		if info, err := os.Lstat(target); err == nil {
			if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
				return domain.ErrInvalidTarget
			}
			content, err := os.ReadFile(target)
			if err != nil || string(content) != payload {
				return domain.ErrInvalidTarget
			}
			continue
		} else if !os.IsNotExist(err) {
			return domain.ErrLedgerFailure
		}
		if err := os.WriteFile(target, []byte(payload), 0o600); err != nil {
			return domain.ErrLedgerFailure
		}
	}
	return nil
}

func validateAssets(dir string) error {
	for path, payload := range assets.Files {
		info, err := os.Lstat(filepath.Join(dir, path))
		if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			return domain.ErrInvalidTarget
		}
		content, err := os.ReadFile(filepath.Join(dir, path))
		if err != nil || string(content) != payload {
			return domain.ErrInvalidTarget
		}
	}
	return nil
}

func Doctor(target string) error {
	if target != "" {
		root, err := lifecycleRoot(target)
		if err != nil {
			return err
		}
		if _, err := os.Lstat(filepath.Join(root, ".docmanager")); err == nil {
			if err := ownedState(filepath.Join(root, ".docmanager")); err != nil {
				return err
			}
			if err := validateAssets(filepath.Join(root, ".docmanager")); err != nil {
				return err
			}
		} else if !os.IsNotExist(err) {
			return domain.ErrInvalidTarget
		}
	}
	if _, err := exec.LookPath("git"); err != nil {
		return domain.ErrGitUnavailable
	}
	return nil
}

func Uninstall(target string) error {
	root, err := lifecycleRoot(target)
	if err != nil {
		return err
	}
	dir := filepath.Join(root, ".docmanager")
	_, err = os.Lstat(dir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return domain.ErrInvalidTarget
	}
	if err := ownedState(dir); err != nil {
		return err
	}
	if err := os.RemoveAll(dir); err != nil {
		return domain.ErrLedgerFailure
	}
	return nil
}

func ownedState(dir string) error {
	info, err := os.Lstat(dir)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return domain.ErrInvalidTarget
	}
	marker := filepath.Join(dir, ".owned")
	info, err = os.Lstat(marker)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return domain.ErrInvalidTarget
	}
	content, err := os.ReadFile(marker)
	if err != nil || string(content) != ownershipMarker {
		return domain.ErrInvalidTarget
	}
	return nil
}

func lifecycleRoot(target string) (string, error) {
	if target == "" || hasTraversalComponent(target) {
		return "", domain.ErrInvalidTarget
	}
	root, err := filepath.Abs(target)
	if err != nil {
		return "", domain.ErrInvalidTarget
	}
	info, err := os.Lstat(root)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", domain.ErrInvalidTarget
	}
	output, err := exec.Command("git", "-C", root, "rev-parse", "--show-toplevel").Output()
	if err != nil || filepath.Clean(strings.TrimSpace(string(output))) != filepath.Clean(root) {
		return "", domain.ErrInvalidTarget
	}
	return root, nil
}

func hasTraversalComponent(target string) bool {
	for _, component := range strings.FieldsFunc(target, func(r rune) bool {
		return r == '/' || r == '\\'
	}) {
		if component == ".." {
			return true
		}
	}
	return false
}
