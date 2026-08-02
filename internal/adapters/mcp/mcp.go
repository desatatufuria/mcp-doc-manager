package mcpadapter

import (
	"context"
	"encoding/json"
	"errors"

	gitadapter "github.com/desatatufuria/mcp-doc-manager/internal/adapters/git"
	sqliteadapter "github.com/desatatufuria/mcp-doc-manager/internal/adapters/sqlite"
	"github.com/desatatufuria/mcp-doc-manager/internal/app"
	"github.com/desatatufuria/mcp-doc-manager/internal/domain"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type scopeInput struct {
	Repository string       `json:"repository" jsonschema:"absolute repository root"`
	Scope      domain.Scope `json:"scope" jsonschema:"explicit selected Git scope"`
}

type receiptInput struct {
	Repository string         `json:"repository" jsonschema:"absolute repository root"`
	Scope      domain.Scope   `json:"scope" jsonschema:"explicit selected Git scope"`
	Receipt    domain.Receipt `json:"receipt" jsonschema:"receipt returned by document_change"`
}

type toolOutput struct {
	Report   *domain.Report `json:"report,omitempty"`
	Verified bool           `json:"verified,omitempty"`
	Error    string         `json:"error,omitempty"`
}

func Serve(ctx context.Context) error {
	return NewServer().Run(ctx, &mcp.StdioTransport{})
}

func NewServer() *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{Name: "repository-documentation-manager", Version: "1.0.0"}, nil)
	mcp.AddTool(server, &mcp.Tool{Name: "document_change", Description: "Analyze one explicit Git scope without changing documentation."}, documentChange)
	mcp.AddTool(server, &mcp.Tool{Name: "verify_receipt", Description: "Verify one content-bound analysis receipt."}, verifyReceipt)
	return server
}

func documentChange(ctx context.Context, _ *mcp.CallToolRequest, input scopeInput) (*mcp.CallToolResult, toolOutput, error) {
	if err := validateScope(input.Scope); err != nil {
		return failure(err)
	}
	resolver := gitadapter.Resolver{GitPath: "git"}
	if err := resolver.ValidateRoot(ctx, input.Repository); err != nil {
		return failure(err)
	}
	ledger, err := sqliteadapter.OpenExisting(input.Repository)
	if err != nil {
		return failure(err)
	}
	defer ledger.Close()
	result := (app.DocumentChange{Resolver: resolver, Ledger: ledger}).Execute(ctx, app.DocumentChangeRequest{Repository: input.Repository, Scope: input.Scope})
	if result.Err != nil {
		return failure(result.Err)
	}
	return nil, toolOutput{Report: result.Report}, nil
}

func verifyReceipt(ctx context.Context, _ *mcp.CallToolRequest, input receiptInput) (*mcp.CallToolResult, toolOutput, error) {
	if err := validateScope(input.Scope); err != nil {
		return failure(err)
	}
	ledger, err := sqliteadapter.OpenExisting(input.Repository)
	if err != nil {
		return failure(err)
	}
	defer ledger.Close()
	err = (app.VerifyReceipt{Resolver: gitadapter.Resolver{GitPath: "git"}, Ledger: ledger}).Execute(ctx, app.DocumentChangeRequest{Repository: input.Repository, Scope: input.Scope}, input.Receipt)
	if err != nil {
		return failure(err)
	}
	return nil, toolOutput{Verified: true}, nil
}

func validateScope(scope domain.Scope) error {
	if scope.Kind != domain.ScopeRange && scope.Kind != domain.ScopeStaged && scope.Kind != domain.ScopeWorktree {
		return domain.ErrInvalidScope
	}
	return scope.Validate()
}

func failure(err error) (*mcp.CallToolResult, toolOutput, error) {
	payload, _ := json.Marshal(toolOutput{Error: classify(err)})
	return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: string(payload)}}}, toolOutput{Error: classify(err)}, nil
}

func classify(err error) string {
	for _, typed := range []error{domain.ErrInvalidScope, domain.ErrNotRepository, domain.ErrOutsideRepository, domain.ErrGitUnavailable, domain.ErrUnsupportedRequest, domain.ErrLedgerFailure, domain.ErrReceiptMismatch, domain.ErrInvalidTarget} {
		if errors.Is(err, typed) {
			return typed.Error()
		}
	}
	return domain.ErrUnsupportedRequest.Error()
}
