package mcpadapter

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

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
	Repository       string `json:"repository" jsonschema:"absolute repository root"`
	Section          string `json:"section,omitempty" jsonschema:"summary (default), documents, or exclusions"`
	ExpectedIdentity string `json:"expected_identity,omitempty" jsonschema:"required exact summary identity for pages"`
	Cursor           int    `json:"cursor,omitempty" jsonschema:"zero-based page cursor"`
	Limit            int    `json:"limit,omitempty" jsonschema:"page size up to 20; defaults to 20"`
}

const maxRadiographyPage = 20

var (
	errInvalidRadiographyPage = errors.New("invalid_radiography_page")
	errStaleRadiography       = errors.New("stale_radiography_identity")
)

type radiographyOutput struct {
	Root                   string                             `json:"root,omitempty"`
	Identity               string                             `json:"identity"`
	Section                string                             `json:"section"`
	DocumentationCount     *int                               `json:"documentation_count,omitempty"`
	ExclusionCount         *int                               `json:"exclusion_count,omitempty"`
	FindingCount           *int                               `json:"finding_count,omitempty"`
	ContextCount           *int                               `json:"context_count,omitempty"`
	UncertainDocumentCount *int                               `json:"uncertain_document_count,omitempty"`
	MissingDigestCount     *int                               `json:"missing_digest_count,omitempty"`
	PageableSections       []string                           `json:"pageable_sections,omitempty"`
	Cursor                 *int                               `json:"cursor,omitempty"`
	Limit                  *int                               `json:"limit,omitempty"`
	Total                  *int                               `json:"total,omitempty"`
	NextCursor             *int                               `json:"next_cursor,omitempty"`
	Documents              []gitadapter.DocumentationEvidence `json:"documents,omitempty"`
	Exclusions             []gitadapter.Exclusion             `json:"exclusions,omitempty"`
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

type verificationInput struct {
	Repository     string               `json:"repository" jsonschema:"absolute repository root"`
	Evidence       string               `json:"evidence"`
	Actor          string               `json:"actor"`
	Interaction    string               `json:"interaction"`
	Request        string               `json:"request"`
	IdempotencyKey string               `json:"idempotency_key"`
	Authorization  domain.Authorization `json:"authorization"`
	Current        domain.Authorization `json:"current"`
	Entry          domain.CatalogEntry  `json:"entry"`
	Action         domain.PlannedAction `json:"action"`
}

type catalogImportInput struct {
	Repository     string                 `json:"repository" jsonschema:"absolute repository root"`
	Plan           app.Plan               `json:"plan" jsonschema:"approved plan returned by propose_plan"`
	Policy         app.Policy             `json:"policy" jsonschema:"approved initial policy returned by propose_plan"`
	Documents      []catalogDocumentInput `json:"documents" jsonschema:"current radiography evidence for every planned document"`
	Evidence       string                 `json:"evidence"`
	Actor          string                 `json:"actor"`
	Interaction    string                 `json:"interaction"`
	Request        string                 `json:"request"`
	IdempotencyKey string                 `json:"idempotency_key"`
}

type catalogDocumentInput struct {
	Path     string `json:"path"`
	Evidence string `json:"evidence"`
	Digest   string `json:"digest"`
}

type catalogAuditInput struct {
	Repository     string        `json:"repository" jsonschema:"absolute repository root"`
	Scope          *domain.Scope `json:"scope,omitempty" jsonschema:"optional explicit Git scope; omit for on-demand radiography"`
	Actor          string        `json:"actor"`
	Interaction    string        `json:"interaction"`
	Request        string        `json:"request"`
	IdempotencyKey string        `json:"idempotency_key"`
}

type toolOutput struct {
	Report       *domain.Report                 `json:"report,omitempty"`
	Radiography  *radiographyOutput             `json:"radiography,omitempty"`
	Planning     *app.PlanningResult            `json:"planning,omitempty"`
	Import       *app.CatalogImportResult       `json:"import,omitempty"`
	Audit        *app.CatalogAuditResult        `json:"audit,omitempty"`
	Verification *app.CatalogVerificationResult `json:"verification,omitempty"`
	Verified     bool                           `json:"verified,omitempty"`
	Error        string                         `json:"error,omitempty"`
}

func Serve(ctx context.Context) error {
	return NewServer().Run(ctx, &mcp.StdioTransport{})
}

func NewServer() *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{Name: "repository-documentation-manager", Version: "1.0.0"}, nil)
	mcp.AddTool(server, &mcp.Tool{Name: "document_change", Description: "Analyze one explicit Git scope without changing documentation."}, documentChange)
	mcp.AddTool(server, &mcp.Tool{Name: "verify_receipt", Description: "Verify one content-bound analysis receipt."}, verifyReceipt)
	mcp.AddTool(server, &mcp.Tool{Name: "radiograph", Description: "Read a bounded repository radiography summary or an identity-bound documents/exclusions page without changing files."}, radiograph)
	mcp.AddTool(server, &mcp.Tool{Name: "propose_plan", Description: "Propose a bounded documentation plan without authoring or persistence."}, proposePlan)
	mcp.AddTool(server, &mcp.Tool{Name: "import_catalog", Description: "Validate and atomically persist an approved catalog plan."}, importCatalog)
	mcp.AddTool(server, &mcp.Tool{Name: "audit_catalog", Description: "Audit the durable catalog from an explicit Git scope or current radiography."}, auditCatalog)
	mcp.AddTool(server, &mcp.Tool{Name: "verify_outcome", Description: "Verify and persist one approved caller-authored outcome."}, verifyOutcome)
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
	if input.Section == "" || input.Section == "summary" {
		summary := radiographySummary(report)
		return nil, toolOutput{Radiography: &summary}, nil
	}
	page, err := radiographyPage(report, input)
	if err != nil {
		return failure(err)
	}
	return nil, toolOutput{Radiography: &page}, nil
}

func radiographySummary(report gitadapter.Radiography) radiographyOutput {
	documentationCount, exclusionCount := len(report.Documentation), len(report.Exclusions)
	findingCount, contextCount := len(report.Findings), len(report.Context)
	uncertainCount, missingDigestCount := 0, 0
	summary := radiographyOutput{
		Root: report.Root, Identity: report.Identity, Section: "summary",
		DocumentationCount: &documentationCount, ExclusionCount: &exclusionCount,
		FindingCount: &findingCount, ContextCount: &contextCount,
		UncertainDocumentCount: &uncertainCount, MissingDigestCount: &missingDigestCount,
		PageableSections: []string{"documents", "exclusions"},
	}
	for _, document := range report.Documentation {
		if document.Classification == "uncertain" {
			uncertainCount++
		}
		if document.Digest == "" {
			missingDigestCount++
		}
	}
	return summary
}

func radiographyPage(report gitadapter.Radiography, input radiographyInput) (radiographyOutput, error) {
	if input.Section != "documents" && input.Section != "exclusions" {
		return radiographyOutput{}, domain.ErrUnsupportedRequest
	}
	if input.ExpectedIdentity == "" || input.Cursor < 0 || input.Limit < 0 || input.Limit > maxRadiographyPage {
		return radiographyOutput{}, errInvalidRadiographyPage
	}
	if input.ExpectedIdentity != report.Identity {
		return radiographyOutput{}, errStaleRadiography
	}
	limit := input.Limit
	if limit == 0 {
		limit = maxRadiographyPage
	}
	total := len(report.Documentation)
	if input.Section == "exclusions" {
		total = len(report.Exclusions)
	}
	if input.Cursor > total || (total > 0 && input.Cursor == total) {
		return radiographyOutput{}, errInvalidRadiographyPage
	}
	start := input.Cursor
	end := min(start+limit, total)
	cursor := input.Cursor
	page := radiographyOutput{Identity: report.Identity, Section: input.Section, Cursor: &cursor, Limit: &limit, Total: &total}
	if end < total {
		page.NextCursor = &end
	}
	if input.Section == "documents" {
		page.Documents = report.Documentation[start:end]
	} else {
		page.Exclusions = report.Exclusions[start:end]
	}
	return page, nil
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

func importCatalog(ctx context.Context, _ *mcp.CallToolRequest, input catalogImportInput) (*mcp.CallToolResult, toolOutput, error) {
	if input.Repository == "" || input.Actor == "" || input.Interaction == "" || input.Request == "" || input.IdempotencyKey == "" {
		return failure(app.ErrCatalogEvidenceRequired)
	}
	documents := make([]app.CatalogDocumentImport, len(input.Documents))
	for i, document := range input.Documents {
		documents[i] = app.CatalogDocumentImport{Path: document.Path, Evidence: document.Evidence, Digest: document.Digest}
	}
	request := app.CatalogImportRequest{Plan: input.Plan, Policy: input.Policy, Documents: documents, Evidence: input.Evidence,
		Actor: input.Actor, Interaction: input.Interaction, Request: input.Request, IdempotencyKey: input.IdempotencyKey}
	if !input.Policy.Approved || (input.Policy.Mode != app.ApprovalRequired && input.Policy.Mode != app.AutomaticAfterApprovedPlan) || (app.CatalogService{}).Import(request).Err != nil {
		return failure(app.ErrCatalogEvidenceRequired)
	}
	resolver := gitadapter.Resolver{GitPath: "git"}
	report, err := resolver.Radiograph(ctx, input.Repository)
	if err != nil {
		return failure(err)
	}
	if input.Evidence != report.Identity || input.Plan.Revision != report.Identity || input.Plan.Radiography != report.Identity || !currentDocuments(input.Documents, report.Documentation) {
		return failure(app.ErrCatalogEvidenceRequired)
	}
	lifecycle, err := sqliteadapter.OpenLifecycle(input.Repository)
	if err != nil {
		return failure(err)
	}
	defer lifecycle.Close()
	result, err := (app.CatalogService{}).ImportAndPersist(ctx, lifecycle, request)
	if err != nil {
		return failure(err)
	}
	return nil, toolOutput{Import: &result}, nil
}

func currentDocuments(input []catalogDocumentInput, current []gitadapter.DocumentationEvidence) bool {
	if len(input) != len(current) {
		return false
	}
	byPath := make(map[string]gitadapter.DocumentationEvidence, len(current))
	for _, document := range current {
		byPath[document.Path] = document
	}
	for _, document := range input {
		actual, ok := byPath[document.Path]
		if !ok || document.Digest != actual.Digest || document.Evidence != actual.Evidence {
			return false
		}
	}
	return true
}

func auditCatalog(ctx context.Context, _ *mcp.CallToolRequest, input catalogAuditInput) (*mcp.CallToolResult, toolOutput, error) {
	if input.Repository == "" || input.Actor == "" || input.Interaction == "" || input.Request == "" || input.IdempotencyKey == "" {
		return failure(app.ErrCatalogEvidenceRequired)
	}
	resolver := gitadapter.Resolver{GitPath: "git"}
	if err := resolver.ValidateRoot(ctx, input.Repository); err != nil {
		return failure(err)
	}
	request := app.CatalogAuditRequest{Repository: input.Repository, Actor: input.Actor, Interaction: input.Interaction, Request: input.Request, IdempotencyKey: input.IdempotencyKey}
	if input.Scope != nil {
		if input.Scope.Validate() != nil {
			return failure(domain.ErrInvalidScope)
		}
		request.Scope = *input.Scope
	}
	if info, err := os.Lstat(filepath.Join(input.Repository, ".docmanager", "lifecycle.db")); err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return failure(app.ErrCatalogEvidenceRequired)
	}
	lifecycle, err := sqliteadapter.OpenLifecycle(input.Repository)
	if err != nil {
		return failure(err)
	}
	defer lifecycle.Close()
	entries, err := lifecycle.LoadCatalog(ctx)
	if err != nil {
		return failure(err)
	}
	if len(entries) == 0 {
		return failure(app.ErrCatalogEvidenceRequired)
	}
	service := app.CatalogService{Resolver: resolver, Radiographer: resolver, AuditStore: lifecycle}
	var result app.CatalogAuditResult
	if input.Scope == nil {
		result, err = service.AuditOnDemand(ctx, request)
	} else {
		result, err = service.AuditChange(ctx, request)
	}
	if err != nil {
		return failure(err)
	}
	return nil, toolOutput{Audit: &result}, nil
}

func verifyOutcome(ctx context.Context, _ *mcp.CallToolRequest, input verificationInput) (*mcp.CallToolResult, toolOutput, error) {
	result := (app.CatalogService{}).VerifyOutcome(app.CatalogVerificationRequest{Authorization: input.Authorization, Current: input.Current, Entry: input.Entry, Action: input.Action, Evidence: input.Evidence})
	if result.Err != nil {
		return failure(result.Err)
	}
	payload, err := json.Marshal(struct {
		Entry        domain.CatalogEntry
		Verification domain.Verification
	}{result.Entry, result.Verification})
	if err != nil {
		return failure(domain.ErrLifecycle)
	}
	lifecycle, err := sqliteadapter.OpenLifecycle(input.Repository)
	if err != nil {
		return failure(err)
	}
	defer lifecycle.Close()
	stored, err := lifecycle.Save(ctx, input.IdempotencyKey, string(payload), verificationProvenance(input, result))
	if err != nil {
		return failure(err)
	}
	var persisted struct {
		Entry        domain.CatalogEntry
		Verification domain.Verification
	}
	if err := json.Unmarshal([]byte(stored), &persisted); err != nil {
		return failure(domain.ErrLifecycle)
	}
	result.Entry, result.Verification = persisted.Entry, persisted.Verification
	return nil, toolOutput{Verification: &result}, nil
}

func verificationProvenance(input verificationInput, result app.CatalogVerificationResult) domain.Provenance {
	requestIdentity, _ := json.Marshal(input)
	return domain.Provenance{
		Authority: domain.DeclaredLocalProvenance, DeclaredActor: input.Actor, Interaction: input.Interaction,
		Context: input.Authorization.PlanRevision, Operation: "verify_outcome", Request: input.Request, IdempotencyKey: input.IdempotencyKey,
		Time: time.Now().UTC().Format(time.RFC3339Nano), Approval: "approved", Evidence: input.Evidence,
		Input: string(requestIdentity), Result: string(result.Verification.State),
		Versions: input.Authorization.PlanRevision + "/" + input.Authorization.PolicyRevision,
	}
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
	for _, typed := range []error{errInvalidRadiographyPage, errStaleRadiography, app.ErrPlanningDenied, app.ErrCatalogEvidenceRequired, domain.ErrIdempotencyConflict, domain.ErrLegacyIdempotencyReplayUnavailable, domain.ErrInvalidScope, domain.ErrNotRepository, domain.ErrOutsideRepository, domain.ErrGitUnavailable, domain.ErrUnsupportedRequest, domain.ErrLedgerFailure, domain.ErrReceiptMismatch, domain.ErrInvalidTarget} {
		if errors.Is(err, typed) {
			return typed.Error()
		}
	}
	return domain.ErrUnsupportedRequest.Error()
}
