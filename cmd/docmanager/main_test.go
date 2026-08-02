package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/desatatufuria/mcp-doc-manager/internal/domain"
)

var docmanagerTestBinary string

func TestMain(m *testing.M) {
	buildDir, err := os.MkdirTemp("", "docmanager-test-")
	if err != nil {
		fmt.Fprintf(os.Stderr, "create docmanager test build directory: %v\n", err)
		os.Exit(1)
	}
	binaryName := "docmanager"
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}
	docmanagerTestBinary = filepath.Join(buildDir, binaryName)
	command := exec.Command("go", "build", "-o", docmanagerTestBinary, ".")
	if output, err := command.CombinedOutput(); err != nil {
		_ = os.RemoveAll(buildDir)
		fmt.Fprintf(os.Stderr, "build docmanager test binary: %v: %s\n", err, output)
		os.Exit(1)
	}

	code := m.Run()
	if err := os.RemoveAll(buildDir); err != nil {
		fmt.Fprintf(os.Stderr, "remove docmanager test build directory: %v\n", err)
		if code == 0 {
			code = 1
		}
	}
	os.Exit(code)
}

func TestCLIHelp(t *testing.T) {
	want := "Usage: docmanager <command> [options]\n\nCommands:\n" +
		"  help                 Show this help\n" +
		"  version              Show the version\n" +
		"  mcp                  Start the stdio MCP server\n" +
		"  document-change      Analyze a selected Git scope\n" +
		"  verify               Verify an analysis receipt\n" +
		"  workspace            Manage repository-local integration\n" +
		"  release              Manage releases\n" +
		"  agent                Manage agent integration\n"
	for _, tc := range []struct {
		name string
		args []string
	}{
		{name: "bare command"},
		{name: "help command", args: []string{"help"}},
		{name: "long help flag", args: []string{"--help"}},
		{name: "short help flag", args: []string{"-h"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			output := runCLI(t, tc.args...)
			if string(output) != want {
				t.Fatalf("help output = %q, want %q", output, want)
			}
		})
	}
}

func TestCLIVersion(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
	}{
		{name: "flag", args: []string{"--version"}},
		{name: "command", args: []string{"version"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			output := runCLI(t, tc.args...)
			if string(output) != "docmanager dev\n" {
				t.Fatalf("docmanager %v output = %q", tc.args, output)
			}
		})
	}
}

func TestCLIMCPVersionIsBounded(t *testing.T) {
	command := docmanagerCommand("mcp", "--version")
	type result struct {
		output []byte
		err    error
	}
	done := make(chan result, 1)
	go func() {
		output, err := command.CombinedOutput()
		done <- result{output: output, err: err}
	}()

	select {
	case got := <-done:
		if got.err != nil || string(got.output) != "docmanager dev\n" {
			t.Fatalf("mcp --version = %v, %q", got.err, got.output)
		}
	case <-time.After(2 * time.Second):
		if command.Process != nil {
			_ = command.Process.Kill()
		}
		<-done
		t.Fatal("mcp --version started the MCP server or hung")
	}
}

func TestCLIUnknownCommandRemainsUnsupported(t *testing.T) {
	output, err := runCLIError("not-a-command")
	if err == nil || string(output) != "unsupported_request\n" {
		t.Fatalf("unknown command = %v, %q", err, output)
	}
}

func TestCLIDocumentChangeVerifyAndLifecycleWithoutDocumentationMutation(t *testing.T) {
	repo := t.TempDir()
	cliGit(t, repo, "init")
	cliGit(t, repo, "config", "user.email", "test@example.com")
	cliGit(t, repo, "config", "user.name", "Test")
	writeCLI(t, filepath.Join(repo, "README.md"), "before\n")
	cliGit(t, repo, "add", "README.md")
	cliGit(t, repo, "commit", "-m", "initial")
	writeCLI(t, filepath.Join(repo, "README.md"), "after\n")
	cliGit(t, repo, "add", "README.md")
	before, err := os.ReadFile(filepath.Join(repo, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	runCLI(t, "install", "--target", repo)
	if _, err := os.Stat(filepath.Join(repo, ".docmanager")); err != nil {
		t.Fatal(err)
	}
	runCLI(t, "doctor", "--target", repo)

	output := runCLI(t, "document-change", "--repo", repo, "--scope", "staged")
	var report domain.Report
	if err := json.Unmarshal(output, &report); err != nil {
		t.Fatalf("decode report: %v: %s", err, output)
	}
	if report.Receipt.Digest == "" || report.Outcome != domain.OutcomeUpdate {
		t.Fatalf("report = %#v", report)
	}
	rawReceipt, err := json.Marshal(report.Receipt)
	if err != nil {
		t.Fatal(err)
	}
	statusBeforeVerify := cliGitOutput(t, repo, "status", "--porcelain=v1", "-z")
	verified := runCLI(t, "verify", "--repo", repo, "--scope", "staged", "--receipt", string(rawReceipt))
	if string(verified) != "{\"verified\":true}\n" {
		t.Fatalf("verify output = %s", verified)
	}
	for _, mutate := range []func(*domain.Receipt){
		func(r *domain.Receipt) { r.Scope.Kind = domain.ScopeWorktree },
		func(r *domain.Receipt) { r.Scope.Range = "other" },
		func(r *domain.Receipt) { r.Evidence = "sha256:other" },
		func(r *domain.Receipt) { r.ChangedPaths[0] = "other.md" },
		func(r *domain.Receipt) { r.Documentation[0].Path = "other.md" },
		func(r *domain.Receipt) { r.Documentation[0].Digest = "sha256:other" },
		func(r *domain.Receipt) { r.Outcome = domain.OutcomeCreate },
		func(r *domain.Receipt) { r.Versions.Schema = "other" },
		func(r *domain.Receipt) { r.Versions.Tool = "other" },
	} {
		tampered := report.Receipt
		tampered.ChangedPaths = append([]string(nil), report.Receipt.ChangedPaths...)
		tampered.Documentation = append([]domain.DocumentationBlob(nil), report.Receipt.Documentation...)
		mutate(&tampered)
		rawTampered, err := json.Marshal(tampered)
		if err != nil {
			t.Fatal(err)
		}
		output, err = runCLIError("verify", "--repo", repo, "--scope", "staged", "--receipt", string(rawTampered))
		if err == nil || string(output) != "receipt_mismatch\n" {
			t.Fatalf("tampered verify = %v, %s", err, output)
		}
	}
	status := cliGitOutput(t, repo, "status", "--porcelain=v1", "-z")
	if status != statusBeforeVerify {
		t.Fatalf("Git status mutated: before %q after %q", statusBeforeVerify, status)
	}
	after, err := os.ReadFile(filepath.Join(repo, "README.md"))
	if err != nil || string(before) != string(after) {
		t.Fatalf("documentation changed: %q, %v", after, err)
	}

	runCLI(t, "uninstall", "--target", repo)
	if _, err := os.Stat(filepath.Join(repo, ".docmanager")); !os.IsNotExist(err) {
		t.Fatalf("ledger still exists: %v", err)
	}
}

func TestCLIRejectsInvalidScope(t *testing.T) {
	output, err := runCLIError("document-change", "--repo", t.TempDir(), "--scope", "invalid")
	if err == nil || string(output) != "invalid_scope\n" {
		t.Fatalf("invalid scope = %v, %s", err, output)
	}
	repo := t.TempDir()
	cliGit(t, repo, "init")
	before := cliGitOutput(t, repo, "status", "--porcelain=v1", "-z")
	output, err = runCLIError("verify", "--repo", repo, "--scope", "invalid", "--receipt", `{}`)
	if err == nil || string(output) != "invalid_scope\n" {
		t.Fatalf("invalid verify scope = %v, %s", err, output)
	}
	if after := cliGitOutput(t, repo, "status", "--porcelain=v1", "-z"); after != before {
		t.Fatalf("Git status mutated: %q", after)
	}
}

func TestCLIHeadlessHierarchyReturnsJSONResult(t *testing.T) {
	repo := t.TempDir()
	cliGit(t, repo, "init")

	output := runCLI(t, "workspace", "status", "--target", repo, "--json")
	var result struct {
		Operation string `json:"operation"`
		Outcome   string `json:"outcome"`
		Status    struct {
			Workspace *struct {
				Root string `json:"root"`
			} `json:"workspace"`
		} `json:"status"`
	}
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode result: %v: %s", err, output)
	}
	if result.Operation != "workspace.status" || result.Outcome != "status" || result.Status.Workspace == nil || result.Status.Workspace.Root != repo {
		t.Fatalf("workspace status = %#v", result)
	}
}

func TestCLIAliasesAreDeprecatedOnlyInHumanOutput(t *testing.T) {
	repo := t.TempDir()
	cliGit(t, repo, "init")

	human := runCLI(t, "install", "--target", repo)
	if string(human) != "deprecated: use workspace install\n" {
		t.Fatalf("install alias output = %q", human)
	}
	t.Cleanup(func() { runCLI(t, "uninstall", "--target", repo) })

	machine := runCLI(t, "doctor", "--target", repo, "--json")
	if string(machine) == "" || string(machine[:1]) != "{" || string(machine) == "deprecated: use workspace doctor\n" {
		t.Fatalf("doctor JSON alias output = %q", machine)
	}
}

func TestCLIPrevalidatesRepositoryBeforeOpeningLedger(t *testing.T) {
	repo := t.TempDir()
	cliGit(t, repo, "init")
	subdir := filepath.Join(repo, "nested")
	if err := os.Mkdir(subdir, 0o700); err != nil {
		t.Fatal(err)
	}
	before := cliGitOutput(t, repo, "status", "--porcelain=v1", "-z")
	output, err := runCLIError("document-change", "--repo", subdir, "--scope", "staged")
	if err == nil || string(output) != "outside_repository\n" {
		t.Fatalf("prevalidation = %v, %s", err, output)
	}
	if _, err := os.Stat(filepath.Join(subdir, ".docmanager")); !os.IsNotExist(err) {
		t.Fatalf("ledger state created: %v", err)
	}
	if after := cliGitOutput(t, repo, "status", "--porcelain=v1", "-z"); after != before {
		t.Fatalf("Git status mutated: %q", after)
	}
}

func runCLIError(args ...string) ([]byte, error) {
	return docmanagerCommand(args...).CombinedOutput()
}

func runCLI(t *testing.T, args ...string) []byte {
	t.Helper()
	command := docmanagerCommand(args...)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("docmanager %v: %v: %s", args, err, output)
	}
	return output
}

func docmanagerCommand(args ...string) *exec.Cmd {
	return exec.Command(docmanagerTestBinary, args...)
}

func cliGit(t *testing.T, repo string, args ...string) {
	t.Helper()
	if output, err := exec.Command("git", append([]string{"-C", repo}, args...)...).CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v: %s", args, err, output)
	}
}

func cliGitOutput(t *testing.T, repo string, args ...string) string {
	t.Helper()
	output, err := exec.Command("git", append([]string{"-C", repo}, args...)...).Output()
	if err != nil {
		t.Fatal(err)
	}
	return string(output)
}

func writeCLI(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
