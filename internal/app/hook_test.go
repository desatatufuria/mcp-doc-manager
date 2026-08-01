package app

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	gitadapter "github.com/gentleman-programming/repository-documentation-manager/internal/adapters/git"
	sqliteadapter "github.com/gentleman-programming/repository-documentation-manager/internal/adapters/sqlite"
	"github.com/gentleman-programming/repository-documentation-manager/internal/domain"
)

func TestPrePushScopeRejectsArbitraryTokens(t *testing.T) {
	if _, err := prePushScope(context.Background(), t.TempDir(), "a b c d"); !errors.Is(err, domain.ErrUnsupportedRequest) {
		t.Fatalf("arbitrary fields error = %v", err)
	}
}

func TestPrePushScopeRejectsGitInvalidRefs(t *testing.T) {
	repo, old, next := hookRepository(t)
	for _, ref := range []string{"refs/heads/../main", "refs/heads/main..next", "refs/heads/.hidden", "refs/heads/main.lock"} {
		for _, update := range []string{"refs/heads/main " + next + " " + ref + " " + old, ref + " " + next + " refs/heads/main " + old} {
			if _, err := prePushScope(context.Background(), repo, update); !errors.Is(err, domain.ErrUnsupportedRequest) {
				t.Fatalf("invalid ref update %q error = %v", update, err)
			}
		}
	}
	deletion := "(delete) " + strings.Repeat("0", len(next)) + " refs/heads/main " + next
	if scope, err := prePushScope(context.Background(), repo, deletion); err != nil || scope != nil {
		t.Fatalf("deletion scope = %#v, %v", scope, err)
	}
}

func TestPrePushVerifierRequiresMatchingStoredReceipt(t *testing.T) {
	repo, old, next := hookRepository(t)
	ctx := context.Background()
	ledger, err := sqliteadapter.Open(repo)
	if err != nil {
		t.Fatal(err)
	}
	defer ledger.Close()
	resolver := gitadapter.Resolver{GitPath: "git"}
	for _, scope := range []domain.Scope{{Kind: domain.ScopeRange, Range: old + ".." + next}, {Kind: domain.ScopeRange, Range: next + "^.." + next}} {
		if result := (DocumentChange{Resolver: resolver, Ledger: ledger}).Execute(ctx, DocumentChangeRequest{Repository: repo, Scope: scope}); result.Err != nil {
			t.Fatalf("store receipt for %s: %v", scope.Range, result.Err)
		}
	}
	verifier := PrePushVerifier{Resolver: resolver, Ledger: ledger, Receipts: ledger}
	updates := strings.Join([]string{
		"refs/heads/main " + next + " refs/heads/main " + old,
		"refs/heads/topic " + next + " refs/for/review " + old,
	}, "\n")
	for _, update := range strings.Split(updates, "\n") {
		if _, err := prePushScope(ctx, repo, update); err != nil {
			t.Fatalf("parse update %q: %v", update, err)
		}
	}
	for _, update := range strings.Split(updates, "\n") {
		if result, err := verifier.Validate(ctx, strings.NewReader(update), "fail", repo); err != nil || result.Status != "valid" {
			t.Fatalf("valid update %q = %#v, %v", update, result, err)
		}
	}
	if result, err := verifier.Validate(ctx, strings.NewReader(updates), "fail", repo); err != nil || result.Status != "valid" {
		t.Fatalf("valid update batch = %#v, %v", result, err)
	}
	if result, err := verifier.Validate(ctx, strings.NewReader("a b c d\n"), "warn", repo); err != nil || result.Status != "warning" {
		t.Fatalf("warn ambiguity = %#v, %v", result, err)
	}
	if _, err := verifier.Validate(ctx, strings.NewReader("a b c d\n"), "fail", repo); !errors.Is(err, domain.ErrUnsupportedRequest) {
		t.Fatalf("fail ambiguity = %v", err)
	}
	if _, err := (PrePushVerifier{Resolver: resolver, Ledger: ledger, Receipts: receiptCatalog{}}).Validate(ctx, strings.NewReader("refs/heads/main "+next+" refs/heads/main "+old), "fail", repo); !errors.Is(err, domain.ErrReceiptMismatch) {
		t.Fatalf("missing receipt = %v", err)
	}
	receipts, err := ledger.Receipts(ctx)
	if err != nil {
		t.Fatal(err)
	}
	tampered := append([]domain.Receipt(nil), receipts...)
	tampered[0].Evidence = "sha256:tampered"
	if _, err := (PrePushVerifier{Resolver: resolver, Ledger: ledger, Receipts: receiptCatalog{tampered}}).Validate(ctx, strings.NewReader("refs/heads/main "+next+" refs/heads/main "+old), "fail", repo); !errors.Is(err, domain.ErrReceiptMismatch) {
		t.Fatalf("tampered receipt = %v", err)
	}
	staleScope := domain.Scope{Kind: domain.ScopeRange, Range: next + ".." + old}
	if err := (VerifyReceipt{Resolver: resolver, Ledger: ledger}).Execute(ctx, DocumentChangeRequest{Repository: repo, Scope: staleScope}, receipts[0]); !errors.Is(err, domain.ErrReceiptMismatch) {
		t.Fatalf("stale scope receipt = %v", err)
	}
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("tampered\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if result, err := verifier.Validate(ctx, strings.NewReader("refs/heads/main "+next+" refs/heads/main "+old), "fail", repo); err != nil || result.Status != "valid" {
		t.Fatalf("dirty worktree changed range receipt = %#v, %v", result, err)
	}
}

func TestPrePushVerifierInitialPushBindsCompleteReachableEvidence(t *testing.T) {
	repo, _, next := hookRepository(t)
	ctx := context.Background()
	ledger, err := sqliteadapter.Open(repo)
	if err != nil {
		t.Fatal(err)
	}
	defer ledger.Close()
	resolver := gitadapter.Resolver{GitPath: "git"}
	result := (DocumentChange{Resolver: resolver, Ledger: ledger}).Execute(ctx, DocumentChangeRequest{Repository: repo, Scope: domain.Scope{Kind: domain.ScopeInitial, Range: next}})
	if result.Err != nil {
		t.Fatalf("store complete initial receipt: %v", result.Err)
	}
	if got := strings.Join(result.Report.Evidence.ChangedPaths, ","); got != "README.md,cmd/server.go" {
		t.Fatalf("initial paths = %q", got)
	}
	verifier := PrePushVerifier{Resolver: resolver, Ledger: ledger, Receipts: ledger}
	update := "refs/heads/main " + next + " refs/heads/main " + strings.Repeat("0", len(next))
	if got, err := verifier.Validate(ctx, strings.NewReader(update), "fail", repo); err != nil || got.Status != "valid" {
		t.Fatalf("initial update = %#v, %v", got, err)
	}
	if _, err := (PrePushVerifier{Resolver: resolver, Ledger: ledger, Receipts: receiptCatalog{}}).Validate(ctx, strings.NewReader("refs/heads/main "+next+" refs/heads/main "+strings.Repeat("0", len(next))), "fail", repo); !errors.Is(err, domain.ErrReceiptMismatch) {
		t.Fatalf("initial update without exact receipt = %v", err)
	}
}

func TestPrePushVerifierRootInitialPushRequiresExactReceipt(t *testing.T) {
	repo := t.TempDir()
	hookGit(t, repo, "init")
	hookGit(t, repo, "config", "user.email", "test@example.com")
	hookGit(t, repo, "config", "user.name", "Test")
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("root\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	hookGit(t, repo, "add", "README.md")
	hookGit(t, repo, "commit", "-m", "root")
	next := strings.TrimSpace(hookGitOutput(t, repo, "rev-parse", "HEAD"))
	ctx := context.Background()
	ledger, err := sqliteadapter.Open(repo)
	if err != nil {
		t.Fatal(err)
	}
	defer ledger.Close()
	resolver := gitadapter.Resolver{GitPath: "git"}
	if result := (DocumentChange{Resolver: resolver, Ledger: ledger}).Execute(ctx, DocumentChangeRequest{Repository: repo, Scope: domain.Scope{Kind: domain.ScopeInitial, Range: next}}); result.Err != nil {
		t.Fatalf("store root receipt: %v", result.Err)
	}
	update := "refs/heads/main " + next + " refs/heads/main " + strings.Repeat("0", len(next))
	if result, err := (PrePushVerifier{Resolver: resolver, Ledger: ledger, Receipts: ledger}).Validate(ctx, strings.NewReader(update), "fail", repo); err != nil || result.Status != "valid" {
		t.Fatalf("root initial update = %#v, %v", result, err)
	}
	if _, err := (PrePushVerifier{Resolver: resolver, Ledger: ledger, Receipts: receiptCatalog{}}).Validate(ctx, strings.NewReader(update), "fail", repo); !errors.Is(err, domain.ErrReceiptMismatch) {
		t.Fatalf("root initial update without receipt = %v", err)
	}
}

type receiptCatalog struct{ receipts []domain.Receipt }

func (c receiptCatalog) Receipts(context.Context) ([]domain.Receipt, error) { return c.receipts, nil }

func hookRepository(t *testing.T) (string, string, string) {
	t.Helper()
	repo := t.TempDir()
	hookGit(t, repo, "init")
	hookGit(t, repo, "config", "user.email", "test@example.com")
	hookGit(t, repo, "config", "user.name", "Test")
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("before\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	hookGit(t, repo, "add", "README.md")
	hookGit(t, repo, "commit", "-m", "first")
	old := strings.TrimSpace(hookGitOutput(t, repo, "rev-parse", "HEAD"))
	if err := os.Mkdir(filepath.Join(repo, "cmd"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "cmd", "server.go"), []byte("package cmd\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	hookGit(t, repo, "add", "cmd/server.go")
	hookGit(t, repo, "commit", "-m", "second")
	return repo, old, strings.TrimSpace(hookGitOutput(t, repo, "rev-parse", "HEAD"))
}

func hookGit(t *testing.T, repo string, args ...string) {
	t.Helper()
	if output, err := exec.Command("git", append([]string{"-C", repo}, args...)...).CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v: %s", args, err, output)
	}
}

func hookGitOutput(t *testing.T, repo string, args ...string) string {
	t.Helper()
	output, err := exec.Command("git", append([]string{"-C", repo}, args...)...).Output()
	if err != nil {
		t.Fatal(err)
	}
	return string(output)
}
