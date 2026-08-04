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

	sqliteadapter "github.com/desatatufuria/mcp-doc-manager/internal/adapters/sqlite"
	"github.com/desatatufuria/mcp-doc-manager/internal/app"
	"github.com/desatatufuria/mcp-doc-manager/internal/domain"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestMCPStdioDocumentChangeAndReceipt(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping MCP stdio integration test in short mode")
	}
	repo := mcpRepository(t)
	if _, err := app.WorkspaceInstall(repo, false); err != nil {
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
	if !reflect.DeepEqual(names, []string{"document_change", "propose_plan", "radiograph", "verify_outcome", "verify_receipt"}) {
		t.Fatalf("MCP tools = %v; only bounded lifecycle tools may be exposed", names)
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

func TestMCPDocumentChangeRequiresWorkspaceInitialization(t *testing.T) {
	repo := mcpRepository(t)
	writeMCP(t, filepath.Join(repo, "README.md"), "before\n")
	mcpGit(t, repo, "add", "README.md")
	mcpGit(t, repo, "commit", "-m", "initial")
	writeMCP(t, filepath.Join(repo, "README.md"), "after\n")
	mcpGit(t, repo, "add", "README.md")
	_, output, err := documentChange(context.Background(), nil, scopeInput{Repository: repo, Scope: domain.Scope{Kind: domain.ScopeStaged}})
	if err != nil || output.Error != domain.ErrLedgerFailure.Error() {
		t.Fatalf("before install = %#v, %v", output, err)
	}
	if _, err := os.Lstat(filepath.Join(repo, ".docmanager")); !os.IsNotExist(err) {
		t.Fatalf("MCP analysis created state: %v", err)
	}
	if _, err := app.WorkspaceInstall(repo, false); err != nil {
		t.Fatal(err)
	}
	_, output, err = documentChange(context.Background(), nil, scopeInput{Repository: repo, Scope: domain.Scope{Kind: domain.ScopeStaged}})
	if err != nil || output.Report == nil {
		t.Fatalf("after install = %#v, %v", output, err)
	}
}

func TestMCPStdioRadiographyToPlanWithoutVisibleWrites(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping MCP stdio integration test in short mode")
	}
	repo := mcpRepository(t)
	writeMCP(t, filepath.Join(repo, "README.md"), "before\n")
	mcpGit(t, repo, "add", "README.md")
	mcpGit(t, repo, "commit", "-m", "initial")
	before, err := os.ReadFile(filepath.Join(repo, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	statusBefore := mcpGitOutput(t, repo, "status", "--porcelain=v1", "-z")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	client := mcp.NewClient(&mcp.Implementation{Name: "mcp-plan-test", Version: "1.0.0"}, nil)
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
	if !reflect.DeepEqual(names, []string{"document_change", "propose_plan", "radiograph", "verify_outcome", "verify_receipt"}) {
		t.Fatalf("MCP tools = %v", names)
	}

	radiographyResult := callMCP(t, ctx, session, "radiograph", map[string]any{"repository": repo})
	raw, err := json.Marshal(radiographyResult.StructuredContent)
	if err != nil {
		t.Fatalf("encode radiography output: %v", err)
	}
	var radiographyOutput toolOutput
	if err := json.Unmarshal(raw, &radiographyOutput); err != nil {
		t.Fatalf("decode radiography output: %v", err)
	}
	if radiographyOutput.Radiography == nil || radiographyOutput.Radiography.Identity == "" || len(radiographyOutput.Radiography.Documentation) != 1 || radiographyOutput.Radiography.Documentation[0].Path != "README.md" || radiographyOutput.Radiography.Documentation[0].Digest == "" || radiographyOutput.Radiography.Documentation[0].Classification != "uncertain" || radiographyOutput.Radiography.Documentation[0].Evidence != "maintenance_evidence_absent" {
		t.Fatalf("radiography = %#v", radiographyOutput.Radiography)
	}

	planResult := callMCP(t, ctx, session, "propose_plan", map[string]any{
		"repository": repo, "visible_storage": ".", "audience": "contributors", "language": "en", "owner": "docs-team",
		"confirmed": true, "plan_approved": true, "batch_approved": true, "policy_approved": true,
		"policy": "approval-required", "actions": []map[string]any{{"path": "README.md", "kind": "update", "structural": false}},
		"actor": "local-user", "interaction": "mcp:plan", "request": "request-1", "idempotency_key": "key-1",
	})
	raw, err = json.Marshal(planResult.StructuredContent)
	if err != nil {
		t.Fatalf("encode plan output: %v", err)
	}
	var planOutput toolOutput
	if err := json.Unmarshal(raw, &planOutput); err != nil {
		t.Fatalf("decode plan output: %v", err)
	}
	if planOutput.Planning == nil || planOutput.Planning.Err != nil || planOutput.Planning.Plan.State != domain.PlanApproved || planOutput.Planning.Authorization == nil || !reflect.DeepEqual(planOutput.Planning.Plan.ExistingDocuments, []app.ExistingDocumentTreatment{{Path: "README.md", Treatment: app.InPlaceInventory}}) {
		t.Fatalf("plan = %#v", planOutput.Planning)
	}

	failed, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "propose_plan", Arguments: map[string]any{
		"repository": repo, "visible_storage": ".", "audience": "contributors", "language": "en", "owner": "docs-team",
		"confirmed": true, "plan_approved": true, "batch_approved": false, "policy_approved": true, "policy": "automatic-after-approved-plan",
		"actions": []map[string]any{{"path": "README.md", "kind": "move", "structural": true}},
		"actor":   "local-user", "interaction": "mcp:plan", "request": "request-2", "idempotency_key": "key-2",
	}})
	if err != nil || !failed.IsError {
		t.Fatalf("unapproved structural plan = %#v, %v", failed, err)
	}
	if text, ok := failed.Content[0].(*mcp.TextContent); !ok || text.Text != `{"error":"planning_denied"}` {
		t.Fatalf("unapproved structural plan failure = %#v", failed.Content)
	}
	if after, err := os.ReadFile(filepath.Join(repo, "README.md")); err != nil || string(after) != string(before) {
		t.Fatalf("documentation mutated: %q, %v", after, err)
	}
	if statusAfter := mcpGitOutput(t, repo, "status", "--porcelain=v1", "-z"); statusAfter != statusBefore {
		t.Fatalf("Git status mutated: before %q after %q", statusBefore, statusAfter)
	}
	if _, err := os.Lstat(filepath.Join(repo, ".docmanager")); !os.IsNotExist(err) {
		t.Fatalf("MCP plan persisted lifecycle state: %v", err)
	}
}

func TestMCPVerifyOutcomePersistsIdempotently(t *testing.T) {
	repo := mcpRepository(t)
	writeMCP(t, filepath.Join(repo, "README.md"), "before\n")
	mcpGit(t, repo, "add", "README.md")
	mcpGit(t, repo, "commit", "-m", "initial")
	action := domain.PlannedAction{Path: "README.md", Kind: domain.ActionUpdate}
	authorization := domain.Authorization{BatchID: "batch-1", PlanRevision: "plan-1", PolicyRevision: "policy-1", Scope: domain.Scope{Kind: domain.ScopeWorktree}, Baseline: "baseline-1", Evidence: "evidence-1", Allowed: []domain.PlannedAction{action}, Active: true}
	input := verificationInput{Repository: repo, Authorization: authorization, Current: authorization, Entry: domain.CatalogEntry{Path: "README.md", Evidence: "evidence-1"}, Action: action, Evidence: "evidence-1", Actor: "local-user", Interaction: "mcp:verify", Request: "request-1", IdempotencyKey: "key-1"}
	t.Run("invalid first request does not persist", func(t *testing.T) {
		invalid := input
		invalid.Repository, invalid.Entry.Path = t.TempDir(), "OTHER.md"
		failed, output, err := verifyOutcome(context.Background(), nil, invalid)
		if err != nil || failed == nil || !failed.IsError || output.Error != app.ErrCatalogEvidenceRequired.Error() {
			t.Fatalf("invalid verification = %#v, %#v, %v", failed, output, err)
		}
		lifecycle, err := sqliteadapter.OpenLifecycle(invalid.Repository)
		if err != nil {
			t.Fatal(err)
		}
		defer lifecycle.Close()
		if count, err := lifecycle.Count(context.Background()); err != nil || count != 0 {
			t.Fatalf("persisted invalid outcomes = %d, %v", count, err)
		}
	})
	_, first, err := verifyOutcome(context.Background(), nil, input)
	if err != nil || first.Verification == nil || first.Verification.Err != nil || first.Verification.Verification.State != domain.VerificationVerified {
		t.Fatalf("first verification = %#v, %v", first, err)
	}
	_, replay, err := verifyOutcome(context.Background(), nil, input)
	if err != nil || !reflect.DeepEqual(replay.Verification, first.Verification) {
		t.Fatalf("replay verification = %#v, %v", replay, err)
	}
	for _, tt := range []struct {
		name   string
		mutate func(*verificationInput)
	}{
		{"entry", func(input *verificationInput) { input.Entry.Purpose = "changed" }},
		{"current authorization", func(input *verificationInput) {
			input.Authorization.Baseline, input.Current.Baseline = "baseline-2", "baseline-2"
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			divergent := input
			tt.mutate(&divergent)
			failed, output, err := verifyOutcome(context.Background(), nil, divergent)
			if err != nil || failed == nil || !failed.IsError || output.Error != domain.ErrIdempotencyConflict.Error() {
				t.Fatalf("divergent replay = %#v, %#v, %v", failed, output, err)
			}
		})
	}
	lifecycle, err := sqliteadapter.OpenLifecycle(repo)
	if err != nil {
		t.Fatal(err)
	}
	defer lifecycle.Close()
	if count, err := lifecycle.Count(context.Background()); err != nil || count != 1 {
		t.Fatalf("persisted outcomes = %d, %v", count, err)
	}
}

func TestMCPStdioVerifiesEditedApprovedOutcomeIdempotently(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping MCP stdio integration test in short mode")
	}
	repo := mcpRepository(t)
	writeMCP(t, filepath.Join(repo, "README.md"), "before\n")
	mcpGit(t, repo, "add", "README.md")
	mcpGit(t, repo, "commit", "-m", "initial")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	client := mcp.NewClient(&mcp.Implementation{Name: "mcp-verify-test", Version: "1.0.0"}, nil)
	session, err := client.Connect(ctx, &mcp.CommandTransport{Command: exec.Command("go", "run", "../../../cmd/docmanager", "mcp")}, nil)
	if err != nil {
		t.Fatalf("connect stdio server: %v", err)
	}
	defer session.Close()
	plan := callMCP(t, ctx, session, "propose_plan", map[string]any{
		"repository": repo, "visible_storage": ".", "audience": "contributors", "language": "en", "owner": "docs-team",
		"confirmed": true, "plan_approved": true, "batch_approved": true, "policy_approved": true, "policy": "approval-required",
		"actions": []map[string]any{{"path": "README.md", "kind": "update", "structural": false}},
		"actor":   "local-user", "interaction": "mcp:plan", "request": "request-1", "idempotency_key": "plan-key",
	})
	raw, err := json.Marshal(plan.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	var planned toolOutput
	if err := json.Unmarshal(raw, &planned); err != nil || planned.Planning == nil || planned.Planning.Authorization == nil {
		t.Fatalf("plan = %#v, %v", planned, err)
	}
	writeMCP(t, filepath.Join(repo, "README.md"), "after\n")
	arguments := map[string]any{"repository": repo, "authorization": planned.Planning.Authorization, "current": planned.Planning.Authorization, "entry": domain.CatalogEntry{Path: "README.md", Evidence: planned.Planning.Authorization.Evidence}, "action": domain.PlannedAction{Path: "README.md", Kind: domain.ActionUpdate}, "evidence": planned.Planning.Authorization.Evidence, "actor": "local-user", "interaction": "mcp:verify", "request": "request-2", "idempotency_key": "verify-key"}
	first := callMCP(t, ctx, session, "verify_outcome", arguments)
	replay := callMCP(t, ctx, session, "verify_outcome", arguments)
	firstRaw, err := json.Marshal(first.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	replayRaw, err := json.Marshal(replay.StructuredContent)
	if err != nil || string(replayRaw) != string(firstRaw) {
		t.Fatalf("replay = %s, %v", replayRaw, err)
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
