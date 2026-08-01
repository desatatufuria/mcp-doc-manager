package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	gitadapter "github.com/gentleman-programming/repository-documentation-manager/internal/adapters/git"
	"github.com/gentleman-programming/repository-documentation-manager/internal/domain"
)

func TestDocumentChangeOutcomesAndFailuresDoNotMutate(t *testing.T) {
	tests := []struct {
		name string
		req  DocumentChangeRequest
		want domain.Outcome
		err  error
	}{
		{"update", DocumentChangeRequest{Repository: "repo", Scope: domain.Scope{Kind: domain.ScopeWorktree}}, domain.OutcomeUpdate, nil},
		{"create", DocumentChangeRequest{Repository: "repo", Scope: domain.Scope{Kind: domain.ScopeWorktree}}, domain.OutcomeCreate, nil},
		{"no impact", DocumentChangeRequest{Repository: "repo", Scope: domain.Scope{Kind: domain.ScopeWorktree}}, domain.OutcomeNoImpact, nil},
		{"ambiguous", DocumentChangeRequest{Repository: "repo", Scope: domain.Scope{Kind: domain.ScopeStaged, Range: "a..b"}}, "", domain.ErrInvalidScope},
		{"unsupported", DocumentChangeRequest{Repository: "repo", Scope: domain.Scope{Kind: "remote"}}, "", domain.ErrUnsupportedRequest},
		{"git unavailable", DocumentChangeRequest{Repository: "repo", Scope: domain.Scope{Kind: domain.ScopeWorktree}}, "", domain.ErrGitUnavailable},
	}
	for index, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := fakeResolver{evidence: domain.Evidence{ChangedPaths: [][]string{{"README.md"}, {"cmd/server.go"}, {"internal/format.go"}}[index%3], DocumentationDigests: map[string]string{"README.md": "missing", "cmd/server.go": "missing", "internal/format.go": "missing"}}}
			if tt.err != nil {
				resolver.err = tt.err
			}
			service := DocumentChange{Resolver: resolver}
			result := service.Execute(context.Background(), tt.req)
			if tt.err != nil {
				if !errors.Is(result.Err, tt.err) || result.Report != nil {
					t.Fatalf("result = %#v", result)
				}
				return
			}
			if result.Err != nil || result.Report == nil || result.Report.Outcome != tt.want {
				t.Fatalf("result = %#v", result)
			}
		})
	}
}

func TestDocumentChangeBindsDocumentationToSelectedGitScope(t *testing.T) {
	repo := t.TempDir()
	appGit(t, repo, "init")
	appGit(t, repo, "config", "user.email", "test@example.com")
	appGit(t, repo, "config", "user.name", "Test User")
	writeApp(t, filepath.Join(repo, "README.md"), "first\n")
	appGit(t, repo, "add", "README.md")
	appGit(t, repo, "commit", "-m", "first")
	first := strings.TrimSpace(appGitOutput(t, repo, "rev-parse", "HEAD"))
	writeApp(t, filepath.Join(repo, "README.md"), "range\n")
	appGit(t, repo, "commit", "-am", "range")
	second := strings.TrimSpace(appGitOutput(t, repo, "rev-parse", "HEAD"))
	writeApp(t, filepath.Join(repo, "README.md"), "staged\n")
	appGit(t, repo, "add", "README.md")
	writeApp(t, filepath.Join(repo, "README.md"), "worktree\n")

	resolver := gitadapter.Resolver{GitPath: "git"}
	tests := []struct {
		name    string
		scope   domain.Scope
		content string
	}{
		{"range", domain.Scope{Kind: domain.ScopeRange, Range: first + ".." + second}, "range\n"},
		{"staged", domain.Scope{Kind: domain.ScopeStaged}, "staged\n"},
		{"worktree", domain.Scope{Kind: domain.ScopeWorktree}, "worktree\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := (DocumentChange{Resolver: resolver}).Execute(context.Background(), DocumentChangeRequest{Repository: repo, Scope: tt.scope})
			if result.Err != nil {
				t.Fatal(result.Err)
			}
			want := sha256.Sum256([]byte(tt.content))
			if got := result.Report.Receipt.Documentation[0].Digest; got != "sha256:"+hex.EncodeToString(want[:]) {
				t.Fatalf("documentation digest = %q", got)
			}
		})
	}
}

func TestDocumentChangePreservesTemporaryRepository(t *testing.T) {
	for _, changed := range []string{"README.md", "cmd/server.go", "internal/format.go"} {
		t.Run(changed, func(t *testing.T) {
			repo := t.TempDir()
			appGit(t, repo, "init")
			appGit(t, repo, "config", "user.email", "test@example.com")
			appGit(t, repo, "config", "user.name", "Test User")
			writeApp(t, filepath.Join(repo, "initial.txt"), "initial\n")
			appGit(t, repo, "add", "initial.txt")
			appGit(t, repo, "commit", "-m", "initial")
			writeApp(t, filepath.Join(repo, changed), "change\n")
			appGit(t, repo, "add", changed)
			before := appGitOutput(t, repo, "status", "--porcelain=v1", "-z")

			result := DocumentChange{Resolver: gitadapter.Resolver{GitPath: "git"}}.Execute(context.Background(), DocumentChangeRequest{Repository: repo, Scope: domain.Scope{Kind: domain.ScopeStaged}})
			if result.Err != nil || result.Report == nil {
				t.Fatalf("result = %#v", result)
			}
			if changed == "README.md" {
				digest := sha256.Sum256([]byte("change\n"))
				if got := result.Report.Receipt.Documentation[0].Digest; got != "sha256:"+hex.EncodeToString(digest[:]) {
					t.Fatalf("documentation digest = %q", got)
				}
			}
			after := appGitOutput(t, repo, "status", "--porcelain=v1", "-z")
			if before != after {
				t.Fatalf("analysis mutated Git state: before %q after %q", before, after)
			}
			content, err := os.ReadFile(filepath.Join(repo, changed))
			if err != nil || string(content) != "change\n" {
				t.Fatalf("content after analysis = %q, %v", content, err)
			}
		})
	}
}

type fakeResolver struct {
	evidence domain.Evidence
	err      error
}

func (f fakeResolver) Resolve(context.Context, string, domain.Scope) (domain.Evidence, error) {
	return f.evidence, f.err
}

func writeApp(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func appGit(t *testing.T, repo string, args ...string) {
	t.Helper()
	if output, err := exec.Command("git", append([]string{"-C", repo}, args...)...).CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v: %s", args, err, output)
	}
}

func appGitOutput(t *testing.T, repo string, args ...string) string {
	t.Helper()
	output, err := exec.Command("git", append([]string{"-C", repo}, args...)...).Output()
	if err != nil {
		t.Fatal(err)
	}
	return string(output)
}
