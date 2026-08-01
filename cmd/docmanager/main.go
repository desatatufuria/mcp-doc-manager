package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	gitadapter "github.com/desatatufuria/mcp-doc-manager/internal/adapters/git"
	mcpadapter "github.com/desatatufuria/mcp-doc-manager/internal/adapters/mcp"
	sqliteadapter "github.com/desatatufuria/mcp-doc-manager/internal/adapters/sqlite"
	"github.com/desatatufuria/mcp-doc-manager/internal/app"
	"github.com/desatatufuria/mcp-doc-manager/internal/domain"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, classify(err))
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return domain.ErrUnsupportedRequest
	}
	switch args[0] {
	case "mcp":
		return mcpadapter.Serve(context.Background())
	case "hook-verify":
		fs := flag.NewFlagSet("hook-verify", flag.ContinueOnError)
		mode := fs.String("mode", "warn", "warn|fail")
		repo := fs.String("repo", "", "repository")
		if err := fs.Parse(args[1:]); err != nil {
			return domain.ErrUnsupportedRequest
		}
		if *repo == "" {
			var err error
			*repo, err = os.Getwd()
			if err != nil {
				return domain.ErrNotRepository
			}
		}
		resolver := gitadapter.Resolver{GitPath: "git"}
		if err := resolver.ValidateRoot(context.Background(), *repo); err != nil {
			return err
		}
		ledger, err := sqliteadapter.OpenExisting(*repo)
		if err != nil {
			return err
		}
		defer ledger.Close()
		result, err := (app.PrePushVerifier{Resolver: resolver, Ledger: ledger, Receipts: ledger}).Validate(context.Background(), os.Stdin, *mode, *repo)
		if err != nil {
			return err
		}
		return json.NewEncoder(os.Stdout).Encode(result)
	case "document-change":
		request, err := requestFlags(args[1:])
		if err != nil {
			return err
		}
		resolver := gitadapter.Resolver{GitPath: "git"}
		if err := resolver.ValidateRoot(context.Background(), request.Repository); err != nil {
			return err
		}
		ledger, err := sqliteadapter.Open(request.Repository)
		if err != nil {
			return err
		}
		defer ledger.Close()
		result := (app.DocumentChange{Resolver: resolver, Ledger: ledger}).Execute(context.Background(), request)
		if result.Err != nil {
			return result.Err
		}
		return json.NewEncoder(os.Stdout).Encode(result.Report)
	case "verify":
		request, receipt, err := verifyFlags(args[1:])
		if err != nil {
			return err
		}
		ledger, err := sqliteadapter.OpenExisting(request.Repository)
		if err != nil {
			return err
		}
		defer ledger.Close()
		if err := (app.VerifyReceipt{Resolver: gitadapter.Resolver{GitPath: "git"}, Ledger: ledger}).Execute(context.Background(), request, receipt); err != nil {
			return err
		}
		return json.NewEncoder(os.Stdout).Encode(map[string]bool{"verified": true})
	case "release", "agent":
		return runResource(args[0], args[1:])
	case "workspace":
		return runWorkspace(args[1:], false)
	case "install", "doctor", "uninstall":
		return runWorkspace(append([]string{args[0]}, args[1:]...), true)
	default:
		return domain.ErrUnsupportedRequest
	}
}

func runResource(resource string, args []string) error {
	if len(args) == 0 {
		return &app.StableError{Code: "invalid_input", Classification: "invalid_input", Detail: "missing operation"}
	}
	request, asJSON, err := orchestrationFlags(resource+"."+args[0], args[1:])
	if err != nil {
		return err
	}
	return renderResult(app.Execute(context.Background(), request), asJSON)
}

func runWorkspace(args []string, alias bool) error {
	if len(args) == 0 {
		return &app.StableError{Code: "invalid_input", Classification: "invalid_input", Detail: "missing workspace operation"}
	}
	request, asJSON, err := orchestrationFlags("workspace."+args[0], args[1:])
	if err != nil {
		return err
	}
	if request.Target == "" {
		request.Target, err = os.Getwd()
		if err != nil {
			return err
		}
	}
	if stable := app.ValidateRequest(request); stable != nil {
		return renderResult(app.Execute(context.Background(), request), asJSON)
	}
	if !request.DryRun {
		switch request.Operation {
		case app.OperationWorkspaceInstall:
			err = app.Install(request.Target)
		case app.OperationWorkspaceUninstall:
			err = app.Uninstall(request.Target)
		case app.OperationWorkspaceDoctor:
			err = app.Doctor(request.Target)
		}
		if err != nil {
			return err
		}
	}
	result := app.Execute(context.Background(), request)
	if request.Operation == app.OperationWorkspaceInstall || request.Operation == app.OperationWorkspaceUninstall {
		result.Outcome = app.OutcomeSuccess
		result.Error = nil
	}
	if err := renderResult(result, asJSON); err != nil {
		return err
	}
	if alias && !asJSON {
		fmt.Fprintf(os.Stdout, "deprecated: use workspace %s\n", args[0])
	}
	return nil
}

func orchestrationFlags(operation string, args []string) (app.Request, bool, error) {
	fs := flag.NewFlagSet(operation, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	manifestURL := fs.String("manifest-url", "", "manifest URL")
	installDir := fs.String("install-dir", "", "installation directory")
	version := fs.String("version", "", "version")
	agent := fs.String("agent", "", "agent")
	binary := fs.String("binary", "", "binary")
	configRoot := fs.String("config-root", "", "configuration root")
	target := fs.String("target", "", "workspace target")
	enableHook := fs.Bool("enable-hook", false, "enable hook")
	dryRun := fs.Bool("dry-run", false, "plan without writes")
	asJSON := fs.Bool("json", false, "emit JSON")
	if err := fs.Parse(args); err != nil || fs.NArg() != 0 {
		return app.Request{}, false, &app.StableError{Code: "invalid_input", Classification: "invalid_input", Detail: "invalid command flags"}
	}
	return app.Request{Operation: app.Operation(operation), ManifestURL: *manifestURL, InstallDir: *installDir, Version: *version, Agent: *agent, Binary: *binary, ConfigRoot: *configRoot, Target: *target, EnableHook: *enableHook, DryRun: *dryRun}, *asJSON, nil
}

func renderResult(result app.Result, asJSON bool) error {
	if asJSON || result.Outcome == app.OutcomeStatus || result.Outcome == app.OutcomePlanned {
		if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
			return err
		}
	}
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func requestFlags(args []string) (app.DocumentChangeRequest, error) {
	fs := flag.NewFlagSet("scope", flag.ContinueOnError)
	repo := fs.String("repo", "", "repository")
	kind := fs.String("scope", "", "range|staged|worktree")
	value := fs.String("range", "", "revision range")
	if err := fs.Parse(args); err != nil || *repo == "" {
		return app.DocumentChangeRequest{}, domain.ErrInvalidScope
	}
	if domain.ScopeKind(*kind) != domain.ScopeRange && domain.ScopeKind(*kind) != domain.ScopeStaged && domain.ScopeKind(*kind) != domain.ScopeWorktree {
		return app.DocumentChangeRequest{}, domain.ErrInvalidScope
	}
	return app.DocumentChangeRequest{Repository: *repo, Scope: domain.Scope{Kind: domain.ScopeKind(*kind), Range: *value}}, nil
}

func verifyFlags(args []string) (app.DocumentChangeRequest, domain.Receipt, error) {
	fs := flag.NewFlagSet("verify", flag.ContinueOnError)
	repo := fs.String("repo", "", "repository")
	kind := fs.String("scope", "", "range|staged|worktree")
	value := fs.String("range", "", "revision range")
	raw := fs.String("receipt", "", "receipt JSON")
	if err := fs.Parse(args); err != nil || *repo == "" || *raw == "" {
		return app.DocumentChangeRequest{}, domain.Receipt{}, domain.ErrInvalidScope
	}
	if domain.ScopeKind(*kind) != domain.ScopeRange && domain.ScopeKind(*kind) != domain.ScopeStaged && domain.ScopeKind(*kind) != domain.ScopeWorktree {
		return app.DocumentChangeRequest{}, domain.Receipt{}, domain.ErrInvalidScope
	}
	var receipt domain.Receipt
	if err := json.Unmarshal([]byte(*raw), &receipt); err != nil {
		return app.DocumentChangeRequest{}, domain.Receipt{}, domain.ErrReceiptMismatch
	}
	return app.DocumentChangeRequest{Repository: *repo, Scope: domain.Scope{Kind: domain.ScopeKind(*kind), Range: *value}}, receipt, nil
}

func classify(err error) string {
	var stable *app.StableError
	if errors.As(err, &stable) {
		return stable.Code
	}
	for _, typed := range []error{domain.ErrInvalidScope, domain.ErrNotRepository, domain.ErrOutsideRepository, domain.ErrGitUnavailable, domain.ErrUnsupportedRequest, domain.ErrLedgerFailure, domain.ErrReceiptMismatch, domain.ErrInvalidTarget} {
		if errors.Is(err, typed) {
			return typed.Error()
		}
	}
	return domain.ErrUnsupportedRequest.Error()
}
