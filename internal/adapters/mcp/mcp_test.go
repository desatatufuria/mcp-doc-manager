package mcpadapter

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/gentleman-programming/repository-documentation-manager/internal/app"
	"github.com/gentleman-programming/repository-documentation-manager/internal/domain"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestMCPStdioDocumentChangeAndReceipt(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping MCP stdio integration test in short mode")
	}
	repo := mcpRepository(t)
	if err := app.Install(repo); err != nil {
		t.Fatal(err)
	}
	defer app.Uninstall(repo)
	writeMCP(t, filepath.Join(repo, "README.md"), "before\n")
	mcpGit(t, repo, "add", "README.md")
	mcpGit(t, repo, "commit", "-m", "initial")
	writeMCP(t, filepath.Join(repo, "README.md"), "after\n")
	mcpGit(t, repo, "add", "README.md")
	before, err := os.ReadFile(filepath.Join(repo, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	statusBefore := mcpGitOutput(t, repo, "status", "--porcelain=v1", "-z")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	client := mcp.NewClient(&mcp.Implementation{Name: "mcp-test", Version: "1.0.0"}, nil)
	session, err := client.Connect(ctx, &mcp.CommandTransport{Command: exec.Command("go", "run", "../../../cmd/docmanager", "mcp")}, nil)
	if err != nil {
		t.Fatalf("connect stdio server: %v", err)
	}
	defer session.Close()

	tools, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("list MCP tools: %v", err)
	}
	names := make([]string, 0, len(tools.Tools))
	for _, tool := range tools.Tools {
		names = append(names, tool.Name)
	}
	sort.Strings(names)
	if !reflect.DeepEqual(names, []string{"document_change", "verify_receipt"}) {
		t.Fatalf("MCP tools = %v; mutation-capable lifecycle tools must not be exposed", names)
	}
	result := callMCP(t, ctx, session, "document_change", map[string]any{"repository": repo, "scope": map[string]any{"kind": "staged", "range": ""}})
	raw, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatalf("encode document_change output: %v", err)
	}
	var output toolOutput
	if err := json.Unmarshal(raw, &output); err != nil {
		t.Fatalf("decode document_change output: %v", err)
	}
	if output.Report == nil || output.Report.Outcome != domain.OutcomeUpdate || output.Report.Receipt.Digest == "" {
		t.Fatalf("report = %#v", output.Report)
	}
	cliOutput, err := exec.Command("go", "run", "../../../cmd/docmanager", "document-change", "--repo", repo, "--scope", "staged").Output()
	if err != nil {
		t.Fatalf("CLI document-change: %v", err)
	}
	var cliReport domain.Report
	if err := json.Unmarshal(cliOutput, &cliReport); err != nil || !reflect.DeepEqual(*output.Report, cliReport) {
		t.Fatalf("CLI/MCP report parity = %#v, %v", cliReport, err)
	}
	callMCP(t, ctx, session, "verify_receipt", map[string]any{"repository": repo, "scope": map[string]any{"kind": "staged", "range": ""}, "receipt": output.Report.Receipt})
	failed, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "document_change", Arguments: map[string]any{"repository": repo, "scope": map[string]any{"kind": "invalid", "range": ""}}})
	if err != nil || !failed.IsError {
		t.Fatalf("invalid scope result = %#v, %v", failed, err)
	}
	if text, ok := failed.Content[0].(*mcp.TextContent); !ok || text.Text != `{"error":"invalid_scope"}` {
		t.Fatalf("invalid scope failure = %#v", failed.Content)
	}
	after, err := os.ReadFile(filepath.Join(repo, "README.md"))
	if err != nil || string(after) != string(before) {
		t.Fatalf("documentation mutated: %q, %v", after, err)
	}
	if statusAfter := mcpGitOutput(t, repo, "status", "--porcelain=v1", "-z"); statusAfter != statusBefore {
		t.Fatalf("Git status mutated: before %q after %q", statusBefore, statusAfter)
	}
}

func callMCP(t *testing.T, ctx context.Context, session *mcp.ClientSession, name string, arguments map[string]any) *mcp.CallToolResult {
	t.Helper()
	result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: arguments})
	if err != nil || result.IsError {
		var content string
		if len(result.Content) > 0 {
			if text, ok := result.Content[0].(*mcp.TextContent); ok {
				content = text.Text
			}
		}
		t.Fatalf("%s = %#v, %v, %s", name, result, err, content)
	}
	return result
}

func mcpRepository(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	mcpGit(t, repo, "init")
	mcpGit(t, repo, "config", "user.email", "test@example.com")
	mcpGit(t, repo, "config", "user.name", "Test")
	return repo
}

func mcpGit(t *testing.T, repo string, args ...string) {
	t.Helper()
	if output, err := exec.Command("git", append([]string{"-C", repo}, args...)...).CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v: %s", args, err, output)
	}
}

func mcpGitOutput(t *testing.T, repo string, args ...string) string {
	t.Helper()
	output, err := exec.Command("git", append([]string{"-C", repo}, args...)...).Output()
	if err != nil {
		t.Fatal(err)
	}
	return string(output)
}

func writeMCP(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
