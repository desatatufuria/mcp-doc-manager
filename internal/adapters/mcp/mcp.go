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

type radiographyInput struct {
	Repository string `json:"repository" jsonschema:"absolute repository root"`
}

type planningInput struct {
	Repository     string               `json:"repository" jsonschema:"absolute repository root"`
	VisibleStorage string               `json:"visible_storage"`
	Audience       string               `json:"audience"`
	Language       string               `json:"language"`
	Owner          string               `json:"owner"`
	Actor          string               `json:"actor"`
	Interaction    string               `json:"interaction"`
	Request        string               `json:"request"`
	IdempotencyKey string               `json:"idempotency_key"`
	Confirmed      bool                 `json:"confirmed"`
	PlanApproved   bool                 `json:"plan_approved"`
	BatchApproved  bool                 `json:"batch_approved"`
	PolicyApproved bool                 `json:"policy_approved"`
	Policy         app.LifecyclePolicy  `json:"policy"`
	Actions        []plannedActionInput `json:"actions"`
}

type plannedActionInput struct {
	Path       string            `json:"path"`
	Kind       domain.ActionKind `json:"kind"`
	Structural bool              `json:"structural"`
}

type toolOutput struct {
	Report      *domain.Report          `json:"report,omitempty"`
	Radiography *gitadapter.Radiography `json:"radiography,omitempty"`
	Planning    *app.PlanningResult     `json:"planning,omitempty"`
	Verified    bool                    `json:"verified,omitempty"`
	Error       string                  `json:"error,omitempty"`
}

func Serve(ctx context.Context) error {
	return NewServer().Run(ctx, &mcp.StdioTransport{})
}

func NewServer() *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{Name: "repository-documentation-manager", Version: "1.0.0"}, nil)
	mcp.AddTool(server, &mcp.Tool{Name: "document_change", Description: "Analyze one explicit Git scope without changing documentation."}, documentChange)
	mcp.AddTool(server, &mcp.Tool{Name: "verify_receipt", Description: "Verify one content-bound analysis receipt."}, verifyReceipt)
	mcp.AddTool(server, &mcp.Tool{Name: "radiograph", Description: "Read repository documentation evidence without changing files."}, radiograph)
	mcp.AddTool(server, &mcp.Tool{Name: "propose_plan", Description: "Propose a bounded documentation plan without authoring or persistence."}, proposePlan)
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

func radiograph(ctx context.Context, _ *mcp.CallToolRequest, input radiographyInput) (*mcp.CallToolResult, toolOutput, error) {
	resolver := gitadapter.Resolver{GitPath: "git"}
	report, err := resolver.Radiograph(ctx, input.Repository)
	if err != nil {
		return failure(err)
	}
	return nil, toolOutput{Radiography: &report}, nil
}

func proposePlan(ctx context.Context, _ *mcp.CallToolRequest, input planningInput) (*mcp.CallToolResult, toolOutput, error) {
	request := app.PlanningRequest{
		Repository: input.Repository, VisibleStorage: input.VisibleStorage, Audience: input.Audience, Language: input.Language, Owner: input.Owner,
		Actor: input.Actor, Interaction: input.Interaction, Request: input.Request, IdempotencyKey: input.IdempotencyKey,
		Confirmed: input.Confirmed, PlanApproved: input.PlanApproved, BatchApproved: input.BatchApproved, PolicyApproved: input.PolicyApproved,
		Policy: input.Policy,
	}
	for _, action := range input.Actions {
		request.Actions = append(request.Actions, domain.PlannedAction{Path: action.Path, Kind: action.Kind, Structural: action.Structural})
	}
	result := (app.PlanningService{Radiography: gitadapter.Resolver{GitPath: "git"}}).Plan(ctx, request)
	if result.Err != nil {
		return failure(result.Err)
	}
	return nil, toolOutput{Planning: &result}, nil
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
	for _, typed := range []error{app.ErrPlanningDenied, domain.ErrIdempotencyConflict, domain.ErrInvalidScope, domain.ErrNotRepository, domain.ErrOutsideRepository, domain.ErrGitUnavailable, domain.ErrUnsupportedRequest, domain.ErrLedgerFailure, domain.ErrReceiptMismatch, domain.ErrInvalidTarget} {
		if errors.Is(err, typed) {
			return typed.Error()
		}
	}
	return domain.ErrUnsupportedRequest.Error()
}
