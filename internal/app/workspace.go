package app

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	fsadapter "github.com/desatatufuria/mcp-doc-manager/internal/adapters/filesystem"
)

var (
	ErrWorkspaceTarget = errors.New("invalid workspace target")
	ErrWorkspaceDrift  = errors.New("workspace hook drifted")
)

type workspaceRecord struct {
	Root string              `json:"root"`
	Hook fsadapter.Ownership `json:"hook"`
}

func WorkspaceInstall(target string, enableHook bool) (WorkspaceStatus, error) {
	root, err := workspaceRoot(target)
	if err != nil {
		return WorkspaceStatus{}, err
	}
	if err := Install(root); err != nil {
		return WorkspaceStatus{}, err
	}
	if !enableHook {
		return WorkspaceStatus{Root: root, State: "installed", Hook: "absent"}, nil
	}
	state, release, err := workspaceStateTransaction()
	if err != nil {
		return WorkspaceStatus{}, err
	}
	defer release()
	if existing, ok, err := loadWorkspaceRecord(root); err != nil {
		return WorkspaceStatus{}, err
	} else if ok {
		if err := verifyWorkspaceHook(root, existing.Hook); err != nil {
			return WorkspaceStatus{}, err
		}
		return WorkspaceStatus{Root: root, State: "installed", Hook: "opted-in"}, nil
	}
	hooks, err := workspaceHookTransaction(root)
	if err != nil {
		return WorkspaceStatus{}, err
	}
	hookRelease, err := hooks.Lock()
	if err != nil {
		return WorkspaceStatus{}, err
	}
	defer hookRelease()
	owned, err := hooks.WriteOwned("pre-push", []byte(workspaceHook(root)), 0o700, nil)
	if err != nil {
		return WorkspaceStatus{}, mapWorkspaceFileError(err)
	}
	record := workspaceRecord{Root: root, Hook: owned}
	if err := saveWorkspaceRecord(state, root, record); err != nil {
		return WorkspaceStatus{}, err
	}
	return WorkspaceStatus{Root: root, State: "installed", Hook: "opted-in"}, nil
}

func WorkspaceUninstall(target string) (WorkspaceStatus, error) {
	root, err := workspaceRoot(target)
	if err != nil {
		return WorkspaceStatus{}, err
	}
	_, release, err := workspaceStateTransaction()
	if err != nil {
		return WorkspaceStatus{}, err
	}
	defer release()
	record, ok, err := loadWorkspaceRecord(root)
	if err != nil {
		return WorkspaceStatus{}, err
	}
	if ok {
		if err := verifyWorkspaceHook(root, record.Hook); err != nil {
			return WorkspaceStatus{}, err
		}
		if err := os.Remove(filepath.Join(root, ".git", "hooks", "pre-push")); err != nil {
			return WorkspaceStatus{}, err
		}
		if err := os.Remove(workspaceStatePath(root)); err != nil && !os.IsNotExist(err) {
			return WorkspaceStatus{}, err
		}
	}
	if err := Uninstall(root); err != nil {
		return WorkspaceStatus{}, err
	}
	return WorkspaceStatus{Root: root, State: "absent", Hook: "absent"}, nil
}

func WorkspaceStatusFor(target string) (WorkspaceStatus, error) {
	root, err := workspaceRoot(target)
	if err != nil {
		return WorkspaceStatus{}, err
	}
	status := WorkspaceStatus{Root: root, State: "absent", Hook: "absent"}
	if _, err := os.Lstat(filepath.Join(root, ".docmanager")); err == nil {
		status.State = "installed"
	}
	record, ok, err := loadWorkspaceRecord(root)
	if err != nil {
		return WorkspaceStatus{}, err
	}
	if ok {
		if err := verifyWorkspaceHook(root, record.Hook); err != nil {
			if errors.Is(err, ErrWorkspaceDrift) {
				status.Hook = "drifted"
				return status, nil
			}
			return WorkspaceStatus{}, err
		}
		status.Hook = "opted-in"
	}
	return status, nil
}

func WorkspaceDoctor(target string) (WorkspaceStatus, error) {
	status, err := WorkspaceStatusFor(target)
	if err != nil || status.Hook == "drifted" {
		if status.Hook == "drifted" {
			return status, ErrWorkspaceDrift
		}
		return status, err
	}
	if err := Doctor(status.Root); err != nil {
		return WorkspaceStatus{}, err
	}
	return status, nil
}

func workspaceRoot(target string) (string, error) {
	root, err := lifecycleRoot(target)
	if err != nil {
		return "", ErrWorkspaceTarget
	}
	return root, nil
}

func workspaceHook(root string) string {
	return "#!/bin/sh\nexec docmanager hook-verify --mode warn --repo " + strconv.Quote(root) + "\n"
}

func workspaceHookTransaction(root string) (*fsadapter.Transaction, error) {
	hooks := filepath.Join(root, ".git", "hooks")
	info, err := os.Lstat(hooks)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, ErrWorkspaceTarget
	}
	return fsadapter.Open(hooks)
}

func verifyWorkspaceHook(root string, expected fsadapter.Ownership) error {
	hooks, err := workspaceHookTransaction(root)
	if err != nil {
		return err
	}
	actual, err := hooks.Ownership("pre-push")
	if err != nil || actual != expected {
		return ErrWorkspaceDrift
	}
	return nil
}

func workspaceStateTransaction() (*fsadapter.Transaction, func(), error) {
	root := filepath.Join(workspaceStateRoot(), "docmanager", "workspaces")
	if err := os.MkdirAll(root, 0o700); err != nil {
		return nil, nil, err
	}
	tx, err := fsadapter.Open(root)
	if err != nil {
		return nil, nil, err
	}
	release, err := tx.Lock()
	if err != nil {
		return nil, nil, err
	}
	return tx, release, nil
}

func workspaceStateRoot() string {
	if root := os.Getenv("XDG_STATE_HOME"); root != "" {
		return root
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return os.TempDir()
	}
	return filepath.Join(home, ".local", "state")
}

func workspaceStatePath(root string) string {
	digest := sha256.Sum256([]byte(root))
	return filepath.Join(workspaceStateRoot(), "docmanager", "workspaces", fmtHex(digest[:])+".json")
}

func fmtHex(value []byte) string { return fmt.Sprintf("%x", value) }

func loadWorkspaceRecord(root string) (workspaceRecord, bool, error) {
	payload, err := os.ReadFile(workspaceStatePath(root))
	if os.IsNotExist(err) {
		return workspaceRecord{}, false, nil
	}
	if err != nil {
		return workspaceRecord{}, false, err
	}
	var record workspaceRecord
	if err := json.Unmarshal(payload, &record); err != nil || record.Root != root {
		return workspaceRecord{}, false, ErrWorkspaceDrift
	}
	return record, true, nil
}

func saveWorkspaceRecord(tx *fsadapter.Transaction, root string, record workspaceRecord) error {
	payload, err := json.Marshal(record)
	if err != nil {
		return err
	}
	_, err = tx.WriteOwned(filepath.Base(workspaceStatePath(root)), payload, 0o600, nil)
	return err
}

func mapWorkspaceFileError(err error) error {
	if errors.Is(err, fsadapter.ErrDrift) || errors.Is(err, fsadapter.ErrOwnership) || errors.Is(err, fsadapter.ErrUnsafePath) {
		return ErrWorkspaceDrift
	}
	return err
}
