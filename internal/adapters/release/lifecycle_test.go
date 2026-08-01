package release

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type probeCall struct {
	path string
	args []string
}

type fakeRunner struct {
	calls []probeCall
	err   error
	path  string
}

type timeoutRunner struct{}

func (timeoutRunner) Run(ctx context.Context, _ string, _ ...string) (ProbeResult, error) {
	<-ctx.Done()
	return ProbeResult{}, ctx.Err()
}

func (r *fakeRunner) Run(_ context.Context, path string, args ...string) (ProbeResult, error) {
	r.calls = append(r.calls, probeCall{path: path, args: args})
	if r.err != nil {
		return ProbeResult{}, r.err
	}
	return ProbeResult{Path: r.path}, nil
}

func TestLifecycleProbeUsesLiteralArgvAndRejectsBadHealth(t *testing.T) {
	dir := t.TempDir()
	stage := writeExecutable(t, dir, "candidate", "new")
	runner := &fakeRunner{path: filepath.Join(dir, "docmanager")}
	lifecycle := Lifecycle{Runner: runner, Timeout: time.Millisecond, StateDir: filepath.Join(dir, "state")}
	if _, err := lifecycle.Install(context.Background(), stage, dir, "v1"); err != nil {
		t.Fatal(err)
	}
	if len(runner.calls) != 1 || runner.calls[0].path != filepath.Join(dir, "docmanager") || len(runner.calls[0].args) != 1 || runner.calls[0].args[0] != "--version" {
		t.Fatalf("probe calls = %#v; want literal argv-only --version", runner.calls)
	}
	for _, tc := range []struct {
		name string
		run  *fakeRunner
		want error
	}{
		{"process failure", &fakeRunner{err: errors.New("exit 1")}, ErrProbeFailed},
		{"wrong binary", &fakeRunner{path: stage}, ErrProbeFailed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			l := lifecycle
			l.Runner = tc.run
			if _, err := l.Doctor(context.Background(), dir); !errors.Is(err, tc.want) {
				t.Fatalf("Doctor() error = %v, want %v", err, tc.want)
			}
		})
	}
	timeout := lifecycle
	timeout.Timeout = time.Millisecond
	timeout.Runner = timeoutRunner{}
	if _, err := timeout.Doctor(context.Background(), dir); !errors.Is(err, ErrProbeFailed) {
		t.Fatalf("bounded timeout error = %v, want probe failure", err)
	}
}

func TestLifecycleRefusesUnsafeOrDriftedReplacement(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(t *testing.T, target string)
	}{
		{"symlink", func(t *testing.T, target string) {
			if err := os.Remove(target); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink("elsewhere", target); err != nil {
				t.Fatal(err)
			}
		}},
		{"drift", func(t *testing.T, target string) {
			if err := os.WriteFile(target, []byte("changed"), 0o700); err != nil {
				t.Fatal(err)
			}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			l := healthyLifecycle(dir)
			if _, err := l.Install(context.Background(), writeExecutable(t, dir, "first", "old"), dir, "v1"); err != nil {
				t.Fatal(err)
			}
			target := filepath.Join(dir, "docmanager")
			tc.mutate(t, target)
			if _, err := l.Upgrade(context.Background(), writeExecutable(t, dir, "next", "new"), dir, "v2"); !errors.Is(err, ErrOwnership) {
				t.Fatalf("Upgrade() error = %v, want ownership refusal", err)
			}
		})
	}
}

func TestLifecycleRefusesConcurrentMutation(t *testing.T) {
	dir := t.TempDir()
	l := healthyLifecycle(dir)
	release, err := l.lock()
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	if _, err := l.Install(context.Background(), writeExecutable(t, dir, "candidate", "new"), dir, "v1"); !errors.Is(err, ErrLocked) {
		t.Fatalf("concurrent install error = %v, want locked", err)
	}
}

func TestLifecycleInstallUpgradeStatusDoctorAndRollback(t *testing.T) {
	dir := t.TempDir()
	l := healthyLifecycle(dir)
	if status, err := l.Status(dir); err != nil || status.Installed {
		t.Fatalf("absent Status() = %#v, %v", status, err)
	}
	if status, err := l.Install(context.Background(), writeExecutable(t, dir, "first", "one"), dir, "v1"); err != nil || !status.Installed || !status.Healthy || status.Version != "v1" {
		t.Fatalf("Install() = %#v, %v", status, err)
	}
	if status, err := l.Upgrade(context.Background(), writeExecutable(t, dir, "next", "two"), dir, "v2"); err != nil || status.Version != "v2" {
		t.Fatalf("Upgrade() = %#v, %v", status, err)
	}
	if status, err := l.Doctor(context.Background(), dir); err != nil || !status.Healthy {
		t.Fatalf("Doctor() = %#v, %v", status, err)
	}
	if status, err := l.Rollback(context.Background(), dir); err != nil || status.Version != "v1" {
		t.Fatalf("Rollback() = %#v, %v", status, err)
	}
	if got, err := os.ReadFile(filepath.Join(dir, "docmanager")); err != nil || string(got) != "one" {
		t.Fatalf("restored binary = %q, %v", got, err)
	}
}

func TestLifecycleFailedHealthRestoresPriorBinary(t *testing.T) {
	dir := t.TempDir()
	l := healthyLifecycle(dir)
	if _, err := l.Install(context.Background(), writeExecutable(t, dir, "first", "old"), dir, "v1"); err != nil {
		t.Fatal(err)
	}
	l.Runner = &fakeRunner{err: errors.New("unhealthy")}
	if _, err := l.Upgrade(context.Background(), writeExecutable(t, dir, "next", "new"), dir, "v2"); !errors.Is(err, ErrProbeFailed) {
		t.Fatalf("Upgrade() error = %v", err)
	}
	if got, err := os.ReadFile(filepath.Join(dir, "docmanager")); err != nil || string(got) != "old" {
		t.Fatalf("prior binary = %q, %v", got, err)
	}
}

func TestLifecycleOwnedBinaryHarness(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping subprocess lifecycle harness in short mode")
	}
	dir := t.TempDir()
	l := Lifecycle{Timeout: time.Second, StateDir: filepath.Join(dir, "state")}
	first := writeExecutable(t, dir, "first", "#!/bin/sh\nprintf 'v1\\n'\n")
	second := writeExecutable(t, dir, "second", "#!/bin/sh\nprintf 'v2\\n'\n")
	if _, err := l.Install(context.Background(), first, dir, "v1"); err != nil {
		t.Fatalf("install owned binary: %v", err)
	}
	if _, err := l.Upgrade(context.Background(), second, dir, "v2"); err != nil {
		t.Fatalf("upgrade owned binary: %v", err)
	}
	if status, err := l.Rollback(context.Background(), dir); err != nil || status.Version != "v1" {
		t.Fatalf("rollback owned binary = %#v, %v", status, err)
	}
}

func healthyLifecycle(dir string) Lifecycle {
	return Lifecycle{Runner: &fakeRunner{path: filepath.Join(dir, "docmanager")}, Timeout: time.Second, StateDir: filepath.Join(dir, "state")}
}

func writeExecutable(t *testing.T, dir, name, body string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(body), 0o700); err != nil {
		t.Fatal(err)
	}
	return path
}
