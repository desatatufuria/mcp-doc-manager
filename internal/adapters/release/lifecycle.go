package release

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

var (
	ErrProbeFailed = errors.New("probe_failed")
	ErrOwnership   = errors.New("ownership")
	ErrRollback    = errors.New("rollback_failed")
	ErrLocked      = errors.New("locked")
)

type ProbeResult struct{ Path string }

// Runner is deliberately argv-only; callers never provide a shell command.
type Runner interface {
	Run(context.Context, string, ...string) (ProbeResult, error)
}

type Lifecycle struct {
	Runner   Runner
	Timeout  time.Duration
	StateDir string
	GOOS     string
	GOARCH   string
}

type LifecycleStatus struct {
	Installed bool
	Healthy   bool
	Version   string
	Platform  string
}

type lifecycleRecord struct {
	Version string        `json:"version"`
	Target  string        `json:"target"`
	Digest  string        `json:"digest"`
	Mode    os.FileMode   `json:"mode"`
	Backup  *backupRecord `json:"backup,omitempty"`
}
type backupRecord struct {
	Version string      `json:"version"`
	Data    []byte      `json:"data"`
	Mode    os.FileMode `json:"mode"`
}

func (l Lifecycle) Install(ctx context.Context, source, installDir, version string) (LifecycleStatus, error) {
	release, err := l.lock()
	if err != nil {
		return LifecycleStatus{}, err
	}
	defer release()
	if _, err := l.Status(installDir); err != nil {
		return LifecycleStatus{}, err
	}
	target, err := l.target(installDir)
	if err != nil {
		return LifecycleStatus{}, err
	}
	if _, statErr := os.Lstat(target); statErr == nil {
		return LifecycleStatus{}, ErrOwnership
	} else if !os.IsNotExist(statErr) {
		return LifecycleStatus{}, ErrOwnership
	}
	return l.replace(ctx, source, target, version, nil)
}

func (l Lifecycle) Upgrade(ctx context.Context, source, installDir, version string) (LifecycleStatus, error) {
	release, err := l.lock()
	if err != nil {
		return LifecycleStatus{}, err
	}
	defer release()
	record, err := l.record(installDir)
	if err != nil {
		return LifecycleStatus{}, err
	}
	target, err := l.target(installDir)
	if err != nil || record.Target != target {
		return LifecycleStatus{}, ErrOwnership
	}
	data, mode, err := ownedFile(target, record)
	if err != nil {
		return LifecycleStatus{}, err
	}
	return l.replace(ctx, source, target, version, &backupRecord{Version: record.Version, Data: data, Mode: mode})
}

func (l Lifecycle) Rollback(ctx context.Context, installDir string) (LifecycleStatus, error) {
	release, err := l.lock()
	if err != nil {
		return LifecycleStatus{}, err
	}
	defer release()
	record, err := l.record(installDir)
	if err != nil || record.Backup == nil {
		return LifecycleStatus{}, ErrRollback
	}
	target, err := l.target(installDir)
	if err != nil {
		return LifecycleStatus{}, err
	}
	if _, _, err := ownedFile(target, record); err != nil {
		return LifecycleStatus{}, err
	}
	if err := atomicWrite(target, record.Backup.Data, record.Backup.Mode); err != nil {
		return LifecycleStatus{}, ErrRollback
	}
	next := lifecycleRecord{Version: record.Backup.Version, Target: target, Mode: record.Backup.Mode}
	next.Digest = digest(record.Backup.Data)
	if err := l.save(installDir, next); err != nil {
		return LifecycleStatus{}, ErrRollback
	}
	return l.status(next, true), nil
}

func (l Lifecycle) Status(installDir string) (LifecycleStatus, error) {
	record, err := l.record(installDir)
	if os.IsNotExist(err) {
		return LifecycleStatus{Platform: l.platform()}, nil
	}
	if err != nil {
		return LifecycleStatus{}, err
	}
	target, err := l.target(installDir)
	if err != nil || record.Target != target {
		return LifecycleStatus{}, ErrOwnership
	}
	if _, _, err := ownedFile(target, record); err != nil {
		return LifecycleStatus{}, err
	}
	return l.status(record, false), nil
}

func (l Lifecycle) Doctor(ctx context.Context, installDir string) (LifecycleStatus, error) {
	status, err := l.Status(installDir)
	if err != nil || !status.Installed {
		return status, err
	}
	target, _ := l.target(installDir)
	if err := l.probe(ctx, target); err != nil {
		return status, err
	}
	status.Healthy = true
	return status, nil
}

func (l Lifecycle) replace(ctx context.Context, source, target, version string, backup *backupRecord) (LifecycleStatus, error) {
	data, mode, err := sourceFile(source)
	if err != nil {
		return LifecycleStatus{}, err
	}
	if err := atomicWrite(target, data, mode); err != nil {
		return LifecycleStatus{}, err
	}
	if err := l.probe(ctx, target); err != nil {
		if backup != nil && atomicWrite(target, backup.Data, backup.Mode) != nil {
			return LifecycleStatus{}, ErrRollback
		}
		if backup == nil {
			_ = os.Remove(target)
		}
		return LifecycleStatus{}, err
	}
	record := lifecycleRecord{Version: version, Target: target, Digest: digest(data), Mode: mode, Backup: backup}
	if err := l.save(filepath.Dir(target), record); err != nil {
		return LifecycleStatus{}, err
	}
	return l.status(record, true), nil
}

func (l Lifecycle) probe(parent context.Context, target string) error {
	timeout := l.Timeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()
	runner := l.Runner
	if runner == nil {
		runner = commandRunner{}
	}
	result, err := runner.Run(ctx, target, "--version")
	if err != nil || result.Path != target || ctx.Err() != nil {
		return ErrProbeFailed
	}
	return nil
}

func (l Lifecycle) target(dir string) (string, error) {
	info, err := os.Lstat(dir)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", ErrOwnership
	}
	return filepath.Join(dir, "docmanager"), nil
}
func (l Lifecycle) stateDir() string {
	if l.StateDir != "" {
		return l.StateDir
	}
	return filepath.Join(os.TempDir(), "docmanager-release-state")
}
func (l Lifecycle) statePath(dir string) string {
	return filepath.Join(l.stateDir(), digest([]byte(dir))+".json")
}
func (l Lifecycle) record(dir string) (lifecycleRecord, error) {
	var r lifecycleRecord
	path := l.statePath(dir)
	info, err := os.Lstat(path)
	if err != nil {
		return r, err
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return r, ErrOwnership
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return r, err
	}
	if json.Unmarshal(b, &r) != nil {
		return r, ErrOwnership
	}
	return r, nil
}
func (l Lifecycle) save(dir string, record lifecycleRecord) error {
	if err := l.ensureStateDir(); err != nil {
		return err
	}
	path := l.statePath(dir)
	if info, err := os.Lstat(path); err == nil && (!info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0) {
		return ErrOwnership
	} else if err != nil && !os.IsNotExist(err) {
		return ErrOwnership
	}
	b, err := json.Marshal(record)
	if err != nil {
		return err
	}
	return atomicWrite(path, b, 0o600)
}
func (l Lifecycle) platform() string {
	goos, goarch := l.GOOS, l.GOARCH
	if goos == "" {
		goos = runtime.GOOS
	}
	if goarch == "" {
		goarch = runtime.GOARCH
	}
	return goos + "/" + goarch
}
func (l Lifecycle) status(record lifecycleRecord, healthy bool) LifecycleStatus {
	return LifecycleStatus{Installed: true, Healthy: healthy, Version: record.Version, Platform: l.platform()}
}

func (l Lifecycle) ensureStateDir() error {
	if info, err := os.Lstat(l.stateDir()); err == nil {
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return ErrOwnership
		}
		return nil
	} else if !os.IsNotExist(err) {
		return ErrOwnership
	}
	return os.MkdirAll(l.stateDir(), 0o700)
}

func (l Lifecycle) lock() (func(), error) {
	if err := l.ensureStateDir(); err != nil {
		return nil, err
	}
	path := filepath.Join(l.stateDir(), ".lock")
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if os.IsExist(err) {
		return nil, ErrLocked
	}
	if err != nil {
		return nil, err
	}
	return func() { _ = file.Close(); _ = os.Remove(path) }, nil
}

func sourceFile(path string) ([]byte, os.FileMode, error) {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return nil, 0, ErrOwnership
	}
	b, err := os.ReadFile(path)
	return b, info.Mode().Perm(), err
}
func ownedFile(path string, record lifecycleRecord) ([]byte, os.FileMode, error) {
	b, mode, err := sourceFile(path)
	if err != nil || digest(b) != record.Digest || mode != record.Mode {
		return nil, 0, ErrOwnership
	}
	return b, mode, nil
}
func digest(b []byte) string { sum := sha256.Sum256(b); return hex.EncodeToString(sum[:]) }
func atomicWrite(path string, data []byte, mode os.FileMode) error {
	temp, err := os.CreateTemp(filepath.Dir(path), ".docmanager-release-")
	if err != nil {
		return err
	}
	name := temp.Name()
	defer os.Remove(name)
	if _, err = temp.Write(data); err == nil {
		err = temp.Chmod(mode)
	}
	if err == nil {
		err = temp.Sync()
	}
	if closeErr := temp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	return os.Rename(name, path)
}

type commandRunner struct{}

func (commandRunner) Run(ctx context.Context, path string, args ...string) (ProbeResult, error) {
	cmd := exec.CommandContext(ctx, path, args...)
	out, err := cmd.StdoutPipe()
	if err != nil {
		return ProbeResult{}, err
	}
	if err := cmd.Start(); err != nil {
		return ProbeResult{}, err
	}
	_, readErr := io.Copy(io.Discard, io.LimitReader(out, 64<<10))
	err = errors.Join(err, readErr, cmd.Wait())
	return ProbeResult{Path: path}, err
}
