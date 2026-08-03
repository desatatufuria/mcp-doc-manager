package git

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/desatatufuria/mcp-doc-manager/internal/domain"
)

func TestResolverRadiographsDocumentationWithExplicitExclusions(t *testing.T) {
	repo := newRepository(t)
	files := map[string]string{
		"docs/guide.md":          "maintained\n",
		"README.md":              "uncertain\n",
		"docs/generated/api.md":  "generated\n",
		"docs/api.generated.mdx": "generated\n",
		"vendor/guide.md":        "vendor\n",
		"requirements.txt":       "dependency\n",
		"CMakeLists.txt":         "project()\n",
		"README.sh":              "#!/bin/sh\n",
		"docs/executable.mdx":    "#!/bin/sh\n",
	}
	for path, content := range files {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(repo, path)), 0o700); err != nil {
			t.Fatal(err)
		}
		write(t, filepath.Join(repo, path), content)
	}
	if err := os.Chmod(filepath.Join(repo, "docs/executable.mdx"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("guide.md", filepath.Join(repo, "docs", "linked.md")); err != nil {
		t.Fatal(err)
	}
	runGit(t, repo, "add", ".")
	runGit(t, repo, "commit", "-m", "fixture")
	write(t, filepath.Join(repo, "docs", "draft.mdx"), "untracked\n")
	write(t, filepath.Join(repo, "notes.md"), "untracked uncertain\n")
	if err := os.Symlink("guide.md", filepath.Join(repo, "docs", "untracked-link.md")); err != nil {
		t.Fatal(err)
	}

	resolver := Resolver{GitPath: "git"}
	first, err := resolver.Radiograph(context.Background(), repo)
	if err != nil {
		t.Fatal(err)
	}
	second, err := resolver.Radiograph(context.Background(), repo)
	if err != nil {
		t.Fatal(err)
	}
	if first.Identity == "" || first.Identity != second.Identity {
		t.Fatalf("radiography = %#v", first)
	}
	wantDocumentation := []DocumentationEvidence{
		{Path: "README.md", Digest: "sha256:2c2c27e79d6971c6e7e865045aa88361e791ba82d2b582b97aa81944903e99f6", Classification: "uncertain", Evidence: "maintenance_evidence_absent"},
		{Path: "docs/api.generated.mdx", Digest: "sha256:9f5936ff15d3a2ba7d3d8f21858338a6c1e2adc9fe34c685c7de5b4a00caa29a", Classification: "uncertain", Evidence: "generated_name_heuristic"},
		{Path: "docs/draft.mdx", Classification: "uncertain", Evidence: "untracked_content_not_read"},
		{Path: "docs/guide.md", Digest: "sha256:4038d6296bb0b2d4f46a2467edea3960d474a5027379f018dbd4cf8fb4cc37d2", Classification: "uncertain", Evidence: "maintenance_evidence_absent"},
		{Path: "docs/untracked-link.md", Classification: "uncertain", Evidence: "untracked_content_not_read"},
		{Path: "notes.md", Classification: "uncertain", Evidence: "untracked_content_not_read"},
	}
	wantExclusions := []Exclusion{
		{Path: "CMakeLists.txt", Reason: "non_documentation_path", Evidence: "path_extension_or_known_non_documentation_name"},
		{Path: "README.sh", Reason: "non_documentation_path", Evidence: "path_extension_or_known_non_documentation_name"},
		{Path: "docs/executable.mdx", Reason: "executable_documentation", Evidence: "executable_mode"},
		{Path: "docs/generated/api.md", Reason: "generated_or_vendor", Evidence: "generated_or_vendor_directory"},
		{Path: "docs/linked.md", Reason: "symlink_documentation", Evidence: "symlink_mode"},
		{Path: "requirements.txt", Reason: "non_documentation_path", Evidence: "path_extension_or_known_non_documentation_name"},
		{Path: "vendor/guide.md", Reason: "generated_or_vendor", Evidence: "generated_or_vendor_directory"},
	}
	if !reflect.DeepEqual(first.Documentation, wantDocumentation) || !reflect.DeepEqual(first.Exclusions, wantExclusions) || first.Ownership != (OwnershipUncertainty{Uncertain: true, Reason: "ownership_not_inferred_from_repository_evidence"}) {
		t.Fatalf("radiography = %#v", first)
	}
}

func TestResolverRadiographSkipsIgnoredUntrackedDocumentation(t *testing.T) {
	repo := newRepository(t)
	write(t, filepath.Join(repo, ".gitignore"), "ignored.md\n")
	runGit(t, repo, "add", ".gitignore")
	runGit(t, repo, "commit", "-m", "ignore fixture")
	write(t, filepath.Join(repo, "ignored.md"), "ignored\n")
	write(t, filepath.Join(repo, "visible.md"), "visible\n")

	report, err := (Resolver{GitPath: "git"}).Radiograph(context.Background(), repo)
	if err != nil {
		t.Fatal(err)
	}
	want := []DocumentationEvidence{{Path: "visible.md", Classification: "uncertain", Evidence: "untracked_content_not_read"}}
	if !reflect.DeepEqual(report.Documentation, want) {
		t.Fatalf("documentation = %#v, want %#v", report.Documentation, want)
	}
}

func TestRadiographyIdentityBindsSemanticFields(t *testing.T) {
	entries := map[string]inventoryEntry{"README.sh": {"100755", "script", true}, "guide.md": {"100644", "object", true}}
	documentation := []DocumentationEvidence{{"guide.md", "sha256:digest", "uncertain", "maintenance_evidence_absent"}, {"other.md", "sha256:other", "maintained", "owner_signal"}}
	exclusions := []Exclusion{{"README.sh", "non_documentation_path", "path_extension_or_known_non_documentation_name"}, {"vendor.md", "generated_or_vendor", "generated_or_vendor_directory"}}
	ownership := OwnershipUncertainty{Uncertain: true, Reason: "ownership_not_inferred_from_repository_evidence"}
	root, otherRoot := "/repo", "/other"
	identity := func() string { return radiographyIdentity(root, ownership, entries, documentation, exclusions) }
	swap := func(a, b *string) { *a, *b = *b, *a }
	cases := []struct {
		name   string
		mutate func()
	}{
		{"root", func() { swap(&root, &otherRoot) }},
		{"documentation order", func() { documentation[0], documentation[1] = documentation[1], documentation[0] }},
		{"documentation path", func() { swap(&documentation[0].Path, &documentation[1].Path) }},
		{"documentation digest", func() { swap(&documentation[0].Digest, &documentation[1].Digest) }},
		{"documentation classification", func() { swap(&documentation[0].Classification, &documentation[1].Classification) }},
		{"documentation evidence", func() { swap(&documentation[0].Evidence, &documentation[1].Evidence) }},
		{"exclusion order", func() { exclusions[0], exclusions[1] = exclusions[1], exclusions[0] }},
		{"exclusion path", func() { swap(&exclusions[0].Path, &exclusions[1].Path) }},
		{"exclusion reason", func() { swap(&exclusions[0].Reason, &exclusions[1].Reason) }},
		{"exclusion evidence", func() { swap(&exclusions[0].Evidence, &exclusions[1].Evidence) }},
		{"mode", func() { entries["guide.md"] = inventoryEntry{"100755", "object", true} }},
		{"object ID", func() { entries["guide.md"] = inventoryEntry{"100644", "other", true} }},
		{"tracked", func() { entries["guide.md"] = inventoryEntry{"100644", "object", false} }},
		{"ownership boolean", func() { ownership.Uncertain = !ownership.Uncertain }},
		{"ownership reason", func() { swap(&ownership.Reason, &otherRoot) }},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			baseline := identity()
			test.mutate()
			if baseline == identity() {
				t.Fatal("identity did not bind field")
			}
		})
	}
}

func TestRadiographyExclusionPrecedenceAndSafety(t *testing.T) {
	for _, test := range []struct{ file, mode, reason, evidence string }{
		{"README.sh", "100755", "non_documentation_path", "path_extension_or_known_non_documentation_name"},
		{"docs/linked.md", "120000", "symlink_documentation", "symlink_mode"},
		{"generated/linked.md", "120000", "symlink_documentation", "symlink_mode"},
		{"docs/generated/api.md", "100755", "generated_or_vendor", "generated_or_vendor_directory"},
		{"docs/executable.md", "100755", "executable_documentation", "executable_mode"},
		{"../outside.md", "100644", "unsafe_path", "path_not_safe_to_read"},
	} {
		t.Run(test.file, func(t *testing.T) {
			reason, evidence := radiographyExclusion(test.file, test.mode)
			if reason != test.reason || evidence != test.evidence {
				t.Fatalf("radiographyExclusion(%q, %q) = %q, %q", test.file, test.mode, reason, evidence)
			}
		})
	}
}

func TestResolverResolvesRootsAndScopes(t *testing.T) {
	repo := newRepository(t)
	write(t, filepath.Join(repo, "README.md"), "initial\n")
	runGit(t, repo, "add", "README.md")
	runGit(t, repo, "commit", "-m", "initial")

	write(t, filepath.Join(repo, "README.md"), "commit a\n")
	runGit(t, repo, "commit", "-am", "commit a")
	write(t, filepath.Join(repo, "staged file.txt"), "staged\n")
	runGit(t, repo, "add", "staged file.txt")

	resolver := Resolver{GitPath: "git"}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	relative, err := filepath.Rel(cwd, repo)
	if err != nil {
		t.Fatal(err)
	}
	for _, root := range []string{repo, filepath.Join(repo, "."), relative} {
		t.Run(root, func(t *testing.T) {
			evidence, err := resolver.Resolve(context.Background(), root, domain.Scope{Kind: domain.ScopeStaged})
			if err != nil {
				t.Fatalf("Resolve() error = %v", err)
			}
			if evidence.Root != repo || len(evidence.ChangedPaths) != 1 || evidence.ChangedPaths[0] != "staged file.txt" {
				t.Fatalf("evidence = %#v", evidence)
			}
		})
	}
	if err := runGitError(repo, "reset"); err != nil {
		t.Fatal(err)
	}
	_, err = resolver.Resolve(context.Background(), repo, domain.Scope{Kind: domain.ScopeStaged})
	if !errors.Is(err, domain.ErrEmptyScope) {
		t.Fatalf("empty staged error = %v, want ErrEmptyScope", err)
	}
}

func TestResolverRejectsOutsideAndNonRepositoryRoots(t *testing.T) {
	resolver := Resolver{GitPath: "git"}
	repo := newRepository(t)
	_, err := resolver.Resolve(context.Background(), filepath.Dir(repo), domain.Scope{Kind: domain.ScopeWorktree})
	if !errors.Is(err, domain.ErrOutsideRepository) {
		t.Fatalf("outside error = %v", err)
	}
	notRepo := t.TempDir()
	_, err = resolver.Resolve(context.Background(), notRepo, domain.Scope{Kind: domain.ScopeWorktree})
	if !errors.Is(err, domain.ErrNotRepository) {
		t.Fatalf("non-repository error = %v", err)
	}
}

func TestResolverCanonicalizesRangeAndBindsIdentityToContent(t *testing.T) {
	repo := newRepository(t)
	write(t, filepath.Join(repo, "README.md"), "first\n")
	runGit(t, repo, "add", "README.md")
	runGit(t, repo, "commit", "-m", "first")
	first := strings.TrimSpace(gitOutput(t, repo, "rev-parse", "HEAD"))

	write(t, filepath.Join(repo, "README.md"), "second\n")
	runGit(t, repo, "commit", "-am", "second")
	second := strings.TrimSpace(gitOutput(t, repo, "rev-parse", "HEAD"))

	resolver := Resolver{GitPath: "git"}
	evidence, err := resolver.Resolve(context.Background(), repo, domain.Scope{Kind: domain.ScopeRange, Range: "HEAD~1..HEAD"})
	if err != nil {
		t.Fatal(err)
	}
	if evidence.Scope.Range != first+".."+second {
		t.Fatalf("resolved range = %q, want canonical object IDs", evidence.Scope.Range)
	}
	if evidence.Identity == "" {
		t.Fatal("identity must be content-bound")
	}

	write(t, filepath.Join(repo, "README.md"), "third\n")
	runGit(t, repo, "commit", "-am", "third")
	evidenceChanged, err := resolver.Resolve(context.Background(), repo, domain.Scope{Kind: domain.ScopeRange, Range: second + "..HEAD"})
	if err != nil {
		t.Fatal(err)
	}
	if evidence.Identity == evidenceChanged.Identity {
		t.Fatal("same-path ranges with different content must have different identities")
	}
}

func TestResolverIdentityIgnoresAttributesAndGitConfiguration(t *testing.T) {
	repo := newRepository(t)
	write(t, filepath.Join(repo, "README.md"), "before\n")
	runGit(t, repo, "add", "README.md")
	runGit(t, repo, "commit", "-m", "initial")
	write(t, filepath.Join(repo, "README.md"), "after\n")
	runGit(t, repo, "add", "README.md")

	resolver := Resolver{GitPath: "git"}
	baseline, err := resolver.Resolve(context.Background(), repo, domain.Scope{Kind: domain.ScopeStaged})
	if err != nil {
		t.Fatal(err)
	}
	global := filepath.Join(t.TempDir(), "gitconfig")
	marker := filepath.Join(t.TempDir(), "diff-ran")
	poison := filepath.Join(t.TempDir(), "poison-diff")
	write(t, poison, "#!/bin/sh\ntouch "+marker+"\n")
	if err := os.Chmod(poison, 0o700); err != nil {
		t.Fatal(err)
	}
	write(t, global, "[diff \"poison\"]\n\ttextconv = "+poison+"\n[diff]\n\texternal = "+poison+"\n")
	t.Setenv("GIT_CONFIG_GLOBAL", global)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "0")
	write(t, filepath.Join(repo, ".gitattributes"), "README.md -diff\n")
	write(t, filepath.Join(repo, ".git", "info", "attributes"), "README.md -diff\n")
	runGit(t, repo, "config", "diff.poison.textconv", poison)
	runGit(t, repo, "config", "diff.external", poison)
	t.Setenv("GIT_CONFIG_COUNT", "1")
	t.Setenv("GIT_CONFIG_KEY_0", "diff.external")
	t.Setenv("GIT_CONFIG_VALUE_0", poison)

	isolation, err := resolver.Resolve(context.Background(), repo, domain.Scope{Kind: domain.ScopeStaged})
	if err != nil {
		t.Fatal(err)
	}
	if isolation.Identity != baseline.Identity {
		t.Fatalf("identity changed under attributes/config: baseline %s isolation %s", baseline.Identity, isolation.Identity)
	}
	if _, err := os.Stat(marker); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("configured diff program ran: %v", err)
	}

	write(t, filepath.Join(repo, "README.md"), "different\n")
	runGit(t, repo, "add", "README.md")
	changed, err := resolver.Resolve(context.Background(), repo, domain.Scope{Kind: domain.ScopeStaged})
	if err != nil {
		t.Fatal(err)
	}
	if changed.Identity == baseline.Identity {
		t.Fatal("same-path content change must change identity")
	}
}

func TestResolverRangeScopeIgnoresGitConfiguration(t *testing.T) {
	repo := scopeRepository(t)
	first := strings.TrimSpace(gitOutput(t, repo, "rev-parse", "HEAD~1"))
	baseline := resolveScope(t, repo, domain.Scope{Kind: domain.ScopeRange, Range: first + "..HEAD"})
	poisonGit(t, repo)
	assertSameEvidence(t, baseline, resolveScope(t, repo, domain.Scope{Kind: domain.ScopeRange, Range: first + "..HEAD"}))
	write(t, filepath.Join(repo, "README.md"), "range changed\n")
	runGit(t, repo, "commit", "-am", "range changed")
	changed := resolveScope(t, repo, domain.Scope{Kind: domain.ScopeRange, Range: strings.TrimSpace(gitOutput(t, repo, "rev-parse", "HEAD~1")) + "..HEAD"})
	if changed.Identity == baseline.Identity {
		t.Fatal("same-path range content change must change identity")
	}
}

func TestResolverStagedScopeIgnoresGitConfiguration(t *testing.T) {
	repo := scopeRepository(t)
	write(t, filepath.Join(repo, "README.md"), "staged\n")
	runGit(t, repo, "add", "README.md")
	baseline := resolveScope(t, repo, domain.Scope{Kind: domain.ScopeStaged})
	poisonGit(t, repo)
	assertSameEvidence(t, baseline, resolveScope(t, repo, domain.Scope{Kind: domain.ScopeStaged}))
	write(t, filepath.Join(repo, "README.md"), "staged changed\n")
	runGit(t, repo, "add", "README.md")
	if changed := resolveScope(t, repo, domain.Scope{Kind: domain.ScopeStaged}); changed.Identity == baseline.Identity {
		t.Fatal("same-path staged content change must change identity")
	}
}

func TestResolverWorktreeScopeIgnoresGitConfiguration(t *testing.T) {
	repo := scopeRepository(t)
	write(t, filepath.Join(repo, "README.md"), "worktree\n")
	baseline := resolveScope(t, repo, domain.Scope{Kind: domain.ScopeWorktree})
	poisonGit(t, repo)
	assertSameEvidence(t, baseline, resolveScope(t, repo, domain.Scope{Kind: domain.ScopeWorktree}))
	runGit(t, repo, "checkout", "--", "README.md")
	if err := os.Chmod(filepath.Join(repo, "README.md"), 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := (Resolver{GitPath: "git"}).Resolve(context.Background(), repo, domain.Scope{Kind: domain.ScopeWorktree}); !errors.Is(err, domain.ErrEmptyScope) {
		t.Fatalf("mode-only worktree change = %v, want ErrEmptyScope", err)
	}
	write(t, filepath.Join(repo, "README.md"), "worktree changed\n")
	if changed := resolveScope(t, repo, domain.Scope{Kind: domain.ScopeWorktree}); changed.Identity == baseline.Identity {
		t.Fatal("same-path worktree content change must change identity")
	}
}

func TestResolverClassifiesRevisionContentAndInitialRootDiscoveryOutputOverflow(t *testing.T) {
	repo := scopeRepository(t)
	_, err := (Resolver{GitPath: "git"}).Resolve(context.Background(), repo, domain.Scope{Kind: domain.ScopeRange, Range: "missing..HEAD"})
	if !errors.Is(err, domain.ErrInvalidRevision) {
		t.Fatalf("revision error = %v", err)
	}
	reader := filepath.Join(t.TempDir(), "git-reader")
	write(t, reader, "#!/bin/sh\ncase \"$*\" in *show-toplevel*) echo '"+repo+"' ;; *ls-files*) printf '100644 deadbeef 0\\tREADME.md\\0' ;; *) exit 1 ;; esac\n")
	if err := os.Chmod(reader, 0o700); err != nil {
		t.Fatal(err)
	}
	_, err = (Resolver{GitPath: reader}).Resolve(context.Background(), repo, domain.Scope{Kind: domain.ScopeWorktree})
	if !errors.Is(err, domain.ErrContentRead) {
		t.Fatalf("content error = %v", err)
	}
	git := filepath.Join(t.TempDir(), "git")
	write(t, git, "#!/bin/sh\nyes x | head -c 1048577\n")
	if err := os.Chmod(git, 0o700); err != nil {
		t.Fatal(err)
	}
	_, err = (Resolver{GitPath: git}).Resolve(context.Background(), repo, domain.Scope{Kind: domain.ScopeWorktree})
	if err != domain.ErrOutputLimit {
		t.Fatalf("initial root discovery overflow error = %v, want exactly ErrOutputLimit", err)
	}
}

func TestResolverClassifiesIdentityStageOutputOverflow(t *testing.T) {
	repo := scopeRepository(t)
	git := filepath.Join(t.TempDir(), "git")
	write(t, git, "#!/bin/sh\ncase \"$*\" in *show-toplevel*) printf '%s\\n' '"+repo+"' ;; *'diff --'*) yes x | head -c 1048577 ;; esac\n")
	if err := os.Chmod(git, 0o700); err != nil {
		t.Fatal(err)
	}

	_, err := (Resolver{GitPath: git}).Resolve(context.Background(), repo, domain.Scope{Kind: domain.ScopeStaged})
	if err != domain.ErrOutputLimit {
		t.Fatalf("identity overflow error = %v, want exactly ErrOutputLimit", err)
	}
}

func TestResolverClassifiesDiscoveryStageOutputOverflow(t *testing.T) {
	repo := scopeRepository(t)
	git := filepath.Join(t.TempDir(), "git")
	write(t, git, "#!/bin/sh\ncase \"$*\" in *show-toplevel*) printf '%s\\n' '"+repo+"' ;; *'diff --'*) yes x | head -c 1048577 ;; esac\n")
	if err := os.Chmod(git, 0o700); err != nil {
		t.Fatal(err)
	}

	_, err := (Resolver{GitPath: git}).Resolve(context.Background(), repo, domain.Scope{Kind: domain.ScopeStaged})
	if err != domain.ErrOutputLimit {
		t.Fatalf("discovery overflow error = %v, want exactly ErrOutputLimit", err)
	}
}

func TestResolverWorktreeAndNULDelimitedPaths(t *testing.T) {
	repo := newRepository(t)
	write(t, filepath.Join(repo, "initial.txt"), "initial\n")
	runGit(t, repo, "add", "initial.txt")
	runGit(t, repo, "commit", "-m", "initial")
	name := "docs/line\nbreak.md"
	if err := os.MkdirAll(filepath.Join(repo, "docs"), 0o700); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(repo, name), "before\n")
	runGit(t, repo, "add", name)
	runGit(t, repo, "commit", "-m", "add newline path")
	write(t, filepath.Join(repo, name), "changed\n")

	evidence, err := (Resolver{GitPath: "git"}).Resolve(context.Background(), repo, domain.Scope{Kind: domain.ScopeWorktree})
	if err != nil {
		t.Fatal(err)
	}
	if len(evidence.ChangedPaths) != 1 || evidence.ChangedPaths[0] != name {
		t.Fatalf("NUL-delimited paths = %#v, want %q", evidence.ChangedPaths, name)
	}
}

func TestResolverBoundsStreamingOutputAndClassifiesUnavailableGit(t *testing.T) {
	resolver := Resolver{GitPath: filepath.Join(t.TempDir(), "missing-git")}
	_, err := resolver.Resolve(context.Background(), t.TempDir(), domain.Scope{Kind: domain.ScopeWorktree})
	if !errors.Is(err, domain.ErrGitUnavailable) {
		t.Fatalf("unavailable Git error = %v", err)
	}

	writer := &boundedBuffer{limit: 2}
	if _, err := writer.Write([]byte("abc")); !errors.Is(err, domain.ErrOutputLimit) {
		t.Fatalf("bounded writer error = %v, want ErrOutputLimit", err)
	}
	if writer.Len() != 2 {
		t.Fatalf("buffered bytes = %d, want 2", writer.Len())
	}
}

func TestResolverUsesRawObjectIDsForOversizedStagedContent(t *testing.T) {
	repo := newRepository(t)
	write(t, filepath.Join(repo, "large.bin"), "before\n")
	runGit(t, repo, "add", "large.bin")
	runGit(t, repo, "commit", "-m", "before")
	write(t, filepath.Join(repo, "large.bin"), strings.Repeat("content-bound\n", maxOutput/len("content-bound\n")+1))
	runGit(t, repo, "add", "large.bin")

	evidence, err := (Resolver{GitPath: "git"}).Resolve(context.Background(), repo, domain.Scope{Kind: domain.ScopeStaged})
	if err != nil {
		t.Fatal(err)
	}
	if evidence.Identity == "" {
		t.Fatal("raw object identity must be present")
	}
}

func scopeRepository(t *testing.T) string {
	t.Helper()
	repo := newRepository(t)
	write(t, filepath.Join(repo, "README.md"), "before\n")
	runGit(t, repo, "add", "README.md")
	runGit(t, repo, "commit", "-m", "before")
	write(t, filepath.Join(repo, "README.md"), "after\n")
	runGit(t, repo, "commit", "-am", "after")
	return repo
}

func resolveScope(t *testing.T, repo string, scope domain.Scope) domain.Evidence {
	t.Helper()
	evidence, err := (Resolver{GitPath: "git"}).Resolve(context.Background(), repo, scope)
	if err != nil {
		t.Fatal(err)
	}
	return evidence
}

func assertSameEvidence(t *testing.T, want, got domain.Evidence) {
	t.Helper()
	if want.Identity != got.Identity || want.Scope != got.Scope || strings.Join(want.ChangedPaths, "\x00") != strings.Join(got.ChangedPaths, "\x00") {
		t.Fatalf("evidence changed: %#v != %#v", want, got)
	}
}

func poisonGit(t *testing.T, repo string) {
	t.Helper()
	poison := filepath.Join(t.TempDir(), "poison")
	write(t, poison, "#!/bin/sh\nexit 99\n")
	if err := os.Chmod(poison, 0o700); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(repo, ".gitattributes"), "README.md diff=poison\n")
	write(t, filepath.Join(repo, ".git", "info", "attributes"), "README.md diff=poison\n")
	global, system := filepath.Join(t.TempDir(), "global"), filepath.Join(t.TempDir(), "system")
	write(t, global, "[diff]\n\texternal = "+poison+"\n")
	write(t, system, "[diff \"poison\"]\n\ttextconv = "+poison+"\n")
	t.Setenv("GIT_CONFIG_GLOBAL", global)
	t.Setenv("GIT_CONFIG_SYSTEM", system)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "0")
	t.Setenv("GIT_CONFIG_COUNT", "2")
	t.Setenv("GIT_CONFIG_KEY_0", "diff.external")
	t.Setenv("GIT_CONFIG_VALUE_0", poison)
	t.Setenv("GIT_CONFIG_KEY_1", "diff.poison.textconv")
	t.Setenv("GIT_CONFIG_VALUE_1", poison)
	runGit(t, repo, "config", "core.filemode", "false")
	runGit(t, repo, "config", "diff.renames", "true")
	runGit(t, repo, "config", "diff.external", poison)
	runGit(t, repo, "config", "diff.poison.textconv", poison)
}

func newRepository(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	runGit(t, repo, "init")
	runGit(t, repo, "config", "user.email", "test@example.com")
	runGit(t, repo, "config", "user.name", "Test User")
	return repo
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	if err := runGitError(dir, args...); err != nil {
		t.Fatal(err)
	}
}

func runGitError(dir string, args ...string) error {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	return cmd.Run()
}

func gitOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()
	output, err := exec.Command("git", append([]string{"-C", dir}, args...)...).Output()
	if err != nil {
		t.Fatal(err)
	}
	return string(output)
}
