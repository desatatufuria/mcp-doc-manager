package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"

	gitadapter "github.com/gentleman-programming/repository-documentation-manager/internal/adapters/git"
	mcpadapter "github.com/gentleman-programming/repository-documentation-manager/internal/adapters/mcp"
	sqliteadapter "github.com/gentleman-programming/repository-documentation-manager/internal/adapters/sqlite"
	"github.com/gentleman-programming/repository-documentation-manager/internal/app"
	"github.com/gentleman-programming/repository-documentation-manager/internal/domain"
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
	case "install":
		fs := flag.NewFlagSet("install", flag.ContinueOnError)
		target := fs.String("target", "", "repository target")
		if err := fs.Parse(args[1:]); err != nil {
			return domain.ErrUnsupportedRequest
		}
		return app.Install(*target)
	case "doctor":
		fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
		target := fs.String("target", "", "repository target")
		if err := fs.Parse(args[1:]); err != nil {
			return domain.ErrUnsupportedRequest
		}
		if err := app.Doctor(*target); err != nil {
			return err
		}
		return json.NewEncoder(os.Stdout).Encode(map[string]bool{"git": true})
	case "uninstall":
		fs := flag.NewFlagSet("uninstall", flag.ContinueOnError)
		target := fs.String("target", "", "repository target")
		if err := fs.Parse(args[1:]); err != nil {
			return domain.ErrUnsupportedRequest
		}
		return app.Uninstall(*target)
	default:
		return domain.ErrUnsupportedRequest
	}
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
	for _, typed := range []error{domain.ErrInvalidScope, domain.ErrNotRepository, domain.ErrOutsideRepository, domain.ErrGitUnavailable, domain.ErrUnsupportedRequest, domain.ErrLedgerFailure, domain.ErrReceiptMismatch, domain.ErrInvalidTarget} {
		if errors.Is(err, typed) {
			return typed.Error()
		}
	}
	return domain.ErrUnsupportedRequest.Error()
}
