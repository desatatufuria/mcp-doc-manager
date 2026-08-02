package main

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	agentadapter "github.com/desatatufuria/mcp-doc-manager/internal/adapters/agent"
	"github.com/desatatufuria/mcp-doc-manager/internal/app"
	"github.com/desatatufuria/mcp-doc-manager/internal/domain"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestAcceptanceHeadlessWorkspaceJSONDryRunStatusAndDoctor(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	repo := t.TempDir()
	cliGit(t, repo, "init")

	dryRun := acceptanceJSON(t, "workspace", "install", "--target", repo, "--dry-run", "--json")
	if dryRun.Operation != "workspace.install" || dryRun.Outcome != "planned" {
		t.Fatalf("dry-run result = %#v", dryRun)
	}
	if _, err := os.Lstat(filepath.Join(repo, ".docmanager")); !os.IsNotExist(err) {
		t.Fatalf("dry-run created workspace state: %v", err)
	}

	status := acceptanceJSON(t, "workspace", "status", "--target", repo, "--json")
	if status.Operation != "workspace.status" || status.Outcome != "status" || status.Status.Workspace == nil || status.Status.Workspace.Root != repo || status.Status.Workspace.State != "absent" {
		t.Fatalf("status result = %#v", status)
	}
	doctor := acceptanceJSON(t, "workspace", "doctor", "--target", repo, "--json")
	if doctor.Operation != "workspace.doctor" || doctor.Outcome != "status" || doctor.Status.Workspace == nil || doctor.Status.Workspace.Root != repo {
		t.Fatalf("doctor result = %#v", doctor)
	}
}

func TestAcceptanceGuidedInstallConfiguresOnlyOpenCode(t *testing.T) {
	repo := t.TempDir()
	cliGit(t, repo, "init")
	xdgConfig := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdgConfig)
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	bin := t.TempDir()
	opencode := filepath.Join(bin, "opencode")
	writeCLI(t, opencode, "#!/bin/sh\nexit 0\n")
	if err := os.Chmod(opencode, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))

	output := runCLI(t, "install", "--target", repo, "--agent", "opencode", "--yes")
	if !strings.Contains(string(output), "Repository initialized: "+repo) || !strings.Contains(string(output), "Verify with: opencode mcp list") {
		t.Fatalf("guided success = %q", output)
	}
	configPath := filepath.Join(xdgConfig, "opencode", "opencode.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	var config map[string]any
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatal(err)
	}
	entry := config["mcp"].(map[string]any)["docmanager"].(map[string]any)
	command := entry["command"].([]any)
	if len(entry) != 3 || entry["type"] != "local" || entry["enabled"] != true || len(command) != 2 || command[1] != "mcp" {
		t.Fatalf("OpenCode entry = %#v", entry)
	}
	if _, err := os.Stat(filepath.Join(repo, ".docmanager", "ledger.db")); err != nil {
		t.Fatalf("ledger = %v", err)
	}
	if _, err := os.Lstat(filepath.Join(repo, ".git", "hooks", "pre-push")); !os.IsNotExist(err) {
		t.Fatalf("default hook = %v", err)
	}
	if output := runCLI(t, "install", "--target", repo, "--yes"); !strings.Contains(string(output), "OpenCode configured") {
		t.Fatalf("idempotent guided install = %q", output)
	}
}

func TestAcceptanceGuidedInstallConflictDoesNotInitializeRepository(t *testing.T) {
	repo := t.TempDir()
	cliGit(t, repo, "init")
	xdgConfig := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdgConfig)
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	bin := t.TempDir()
	opencode := filepath.Join(bin, "opencode")
	writeCLI(t, opencode, "#!/bin/sh\nexit 0\n")
	if err := os.Chmod(opencode, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	config := filepath.Join(xdgConfig, "opencode", "opencode.json")
	writeCLI(t, config, `{"mcp":{"docmanager":{"type":"local","command":["other","mcp"],"enabled":true}}}`)
	before, _ := os.ReadFile(config)

	output, err := runCLIError("install", "--target", repo, "--yes")
	if err == nil || string(output) != "ownership\n" {
		t.Fatalf("conflict = %v, %q", err, output)
	}
	if _, err := os.Lstat(filepath.Join(repo, ".docmanager")); !os.IsNotExist(err) {
		t.Fatalf("conflict initialized repository: %v", err)
	}
	after, _ := os.ReadFile(config)
	if string(after) != string(before) {
		t.Fatal("conflict changed OpenCode config")
	}
}

func TestAcceptanceWorkspaceRejectsSymlinkTargetAndTraversal(t *testing.T) {
	repo := t.TempDir()
	cliGit(t, repo, "init")
	link := filepath.Join(t.TempDir(), "repo-link")
	if err := os.Symlink(repo, link); err != nil {
		t.Fatal(err)
	}
	for _, target := range []string{link, repo + string(filepath.Separator) + "nested" + string(filepath.Separator) + ".."} {
		output, commandErr := runCLIError("workspace", "status", "--target", target, "--json")
		var result acceptanceResult
		jsonOutput := strings.SplitN(string(output), "\n", 2)[0]
		if err := json.Unmarshal([]byte(jsonOutput), &result); commandErr == nil || err != nil || result.Outcome != "failure" || !strings.Contains(string(output), "invalid_input") {
			t.Fatalf("target %q accepted: %v: %s", target, commandErr, output)
		}
	}
}

func TestAcceptanceMCPReceiptAndSQLiteRemainReadOnly(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping MCP stdio acceptance integration test in short mode")
	}
	repo := t.TempDir()
	cliGit(t, repo, "init")
	cliGit(t, repo, "config", "user.email", "test@example.com")
	cliGit(t, repo, "config", "user.name", "Test")
	runCLI(t, "workspace", "install", "--target", repo)
	t.Cleanup(func() { runCLI(t, "uninstall", "--target", repo) })
	writeCLI(t, filepath.Join(repo, "README.md"), "before\n")
	cliGit(t, repo, "add", "README.md")
	cliGit(t, repo, "commit", "-m", "initial")
	writeCLI(t, filepath.Join(repo, "README.md"), "after\n")
	cliGit(t, repo, "add", "README.md")
	before := cliGitOutput(t, repo, "status", "--porcelain=v1", "-z")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	client := mcp.NewClient(&mcp.Implementation{Name: "acceptance-test", Version: "1.0.0"}, nil)
	session, err := client.Connect(ctx, &mcp.CommandTransport{Command: docmanagerCommand("mcp")}, nil)
	if err != nil {
		t.Fatalf("connect MCP server: %v", err)
	}
	defer session.Close()
	result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "document_change", Arguments: map[string]any{"repository": repo, "scope": map[string]any{"kind": "staged", "range": ""}}})
	if err != nil || result.IsError {
		t.Fatalf("document_change = %#v, %v", result, err)
	}
	raw, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	var output struct {
		Report *domain.Report `json:"report"`
	}
	if err := json.Unmarshal(raw, &output); err != nil || output.Report == nil || output.Report.Receipt.Digest == "" {
		t.Fatalf("MCP report = %#v, %v", output.Report, err)
	}
	verified, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "verify_receipt", Arguments: map[string]any{"repository": repo, "scope": map[string]any{"kind": "staged", "range": ""}, "receipt": output.Report.Receipt}})
	if err != nil || verified.IsError {
		t.Fatalf("verify_receipt = %#v, %v", verified, err)
	}
	if info, err := os.Stat(filepath.Join(repo, ".docmanager", "ledger.db")); err != nil || !info.Mode().IsRegular() {
		t.Fatalf("SQLite receipt ledger = %v, %v", info, err)
	}
	if after := cliGitOutput(t, repo, "status", "--porcelain=v1", "-z"); after != before {
		t.Fatalf("MCP changed Git status: before %q after %q", before, after)
	}
}

func TestAcceptanceAgentOrchestrationComposesManagedAdapter(t *testing.T) {
	root := t.TempDir()
	binary := filepath.Join(root, "docmanager")
	service := app.AgentService{Adapters: map[string]app.AgentPort{
		"opencode": agentadapter.NewOpenCode(agentadapter.OpenCodeOptions{
			Root: root, Binary: binary, Installed: func() bool { return true },
		}),
	}}
	result := service.Execute(context.Background(), app.Request{Operation: app.OperationAgentStatus, Agent: "opencode"})
	if result.Error != nil || result.Outcome != app.OutcomeStatus || result.Status == nil || result.Status.Agent == nil || result.Status.Agent.Agent != "opencode" || !result.Status.Agent.Installed || !result.Status.Agent.Supported {
		t.Fatalf("agent composition result = %#v", result)
	}
	if _, err := os.Lstat(filepath.Join(root, "opencode.json")); !os.IsNotExist(err) {
		t.Fatalf("agent status mutated isolated config: %v", err)
	}
}

func TestAcceptanceMacOSLexicalVarTargetMatchesPhysicalGitRoot(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("requires macOS /var-to-/private/var lexical path behavior")
	}
	lexicalRoot, err := os.MkdirTemp("/var/tmp", "docmanager-acceptance-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(lexicalRoot) })
	physicalRoot, err := filepath.EvalSymlinks(lexicalRoot)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Clean(lexicalRoot) == filepath.Clean(physicalRoot) {
		t.Skip("macOS temporary directory did not expose distinct lexical and physical roots")
	}
	cliGit(t, lexicalRoot, "init")
	result := acceptanceJSON(t, "workspace", "status", "--target", lexicalRoot, "--json")
	if result.Outcome != "status" || result.Status.Workspace == nil || result.Status.Workspace.Root != lexicalRoot {
		t.Fatalf("lexical root status = %#v", result)
	}
}

func TestAcceptanceWindowsSmokeBuildAndInvocationAgree(t *testing.T) {
	workflow, err := os.ReadFile(filepath.Join("..", "..", ".github", "workflows", "ci.yml"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(workflow)
	if !strings.Contains(text, "output: docmanager.exe") || !strings.Contains(text, "go build -o ${{ matrix.output }} ./cmd/docmanager") {
		t.Fatal("Windows native package build must produce its matrix-selected docmanager.exe output")
	}
	if !strings.Contains(text, `.\docmanager.exe doctor --target $repo`) || !strings.Contains(text, `.\docmanager.exe uninstall --target $repo`) {
		t.Fatal("Windows native package smoke must invoke the executable it builds")
	}
	output := filepath.Join(t.TempDir(), "docmanager.exe")
	command := exec.Command("go", "build", "-o", output, "./cmd/docmanager")
	command.Dir = filepath.Join("..", "..")
	command.Env = append(os.Environ(), "CGO_ENABLED=0", "GOOS=windows", "GOARCH=amd64")
	if buildOutput, err := command.CombinedOutput(); err != nil {
		t.Fatalf("cross-build Windows package smoke: %v: %s", err, buildOutput)
	}
	if info, err := os.Stat(output); err != nil || info.Size() == 0 {
		t.Fatalf("Windows package output = %v, %v", info, err)
	}
}

type acceptanceResult struct {
	Operation string `json:"operation"`
	Outcome   string `json:"outcome"`
	Status    struct {
		Workspace *struct {
			Root  string `json:"root"`
			State string `json:"state"`
		} `json:"workspace"`
	} `json:"status"`
}

func acceptanceJSON(t *testing.T, args ...string) acceptanceResult {
	t.Helper()
	output := runCLI(t, args...)
	var result acceptanceResult
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode %v: %v: %s", args, err, output)
	}
	return result
}
