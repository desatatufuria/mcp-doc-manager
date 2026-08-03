package git

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
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
	for _, root := range []string{repo} {
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
	_, err := resolver.Resolve(context.Background(), repo, domain.Scope{Kind: domain.ScopeStaged})
	if !errors.Is(err, domain.ErrEmptyScope) {
		t.Fatalf("empty staged error = %v, want ErrEmptyScope", err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	relative, err := filepath.Rel(cwd, repo)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := resolver.Resolve(context.Background(), relative, domain.Scope{Kind: domain.ScopeStaged}); !errors.Is(err, domain.ErrOutsideRepository) {
		t.Fatalf("relative root error = %v, want ErrOutsideRepository", err)
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
	if err := os.Mkdir(filepath.Join(repo, "sub"), 0o700); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(t.TempDir(), "repo-link")
	if err := os.Symlink(repo, link); err != nil {
		t.Fatal(err)
	}

	for _, root := range []string{repo + "/sub/..", repo + "/.", filepath.Join(repo, "sub"), link} {
		t.Run(root, func(t *testing.T) {
			if _, err := resolver.Radiograph(context.Background(), root); !errors.Is(err, domain.ErrOutsideRepository) {
				t.Fatalf("Radiograph(%q) error = %v, want ErrOutsideRepository", root, err)
			}
		})
	}

	if report, err := resolver.Radiograph(context.Background(), repo); err != nil || report.Root != repo {
		t.Fatalf("Radiograph(%q) = %#v, %v", repo, report, err)
	}
}

func TestResolverRadiographUsesReadOnlyGitCommands(t *testing.T) {
	repo := newRepository(t)
	write(t, filepath.Join(repo, "README.md"), "fixture\n")
	runGit(t, repo, "add", "README.md")
	runGit(t, repo, "commit", "-m", "fixture")
	write(t, filepath.Join(repo, "README.md"), "second\n")
	runGit(t, repo, "commit", "-am", "second")
	write(t, filepath.Join(repo, "README.md"), "staged\n")
	runGit(t, repo, "add", "README.md")
	commands := filepath.Join(t.TempDir(), "commands")
	proxy := filepath.Join(t.TempDir(), "git-proxy")
	write(t, proxy, "#!/bin/sh\nprintf '%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s\\n' \"$*\" \"$GIT_CONFIG_NOSYSTEM\" \"$GIT_CONFIG_GLOBAL\" \"$GIT_CONFIG_SYSTEM\" \"$GIT_CONFIG_COUNT\" \"$GIT_OPTIONAL_LOCKS\" \"$GIT_NO_LAZY_FETCH\" \"$GIT_NO_REPLACE_OBJECTS\" \"$GIT_TERMINAL_PROMPT\" \"$GIT_PAGER\" \"$GIT_ASKPASS\" \"$GIT_EXTERNAL_DIFF\" \"$GIT_DIFF_OPTS\" \"$LC_ALL\" \"$TZ\" \"$GIT_CONFIG_KEY_0\" \"$GIT_CONFIG_VALUE_0\" \"$GIT_DIR\" \"$GIT_WORK_TREE\" \"$GIT_OBJECT_DIRECTORY\" \"$GIT_ALTERNATE_OBJECT_DIRECTORIES\" >> "+commands+"\nexec git \"$@\"\n")
	if err := os.Chmod(proxy, 0o700); err != nil {
		t.Fatal(err)
	}
	marker, poison, include := filepath.Join(t.TempDir(), "marker"), filepath.Join(t.TempDir(), "poison"), filepath.Join(t.TempDir(), "include")
	write(t, poison, "#!/bin/sh\ntouch "+marker+"\n")
	if err := os.Chmod(poison, 0o700); err != nil {
		t.Fatal(err)
	}
	write(t, include, "[core]\nfsmonitor = "+poison+"\n[filter \"poison\"]\nprocess = "+poison+"\n")
	runGit(t, repo, "config", "include.path", include)
	write(t, filepath.Join(repo, ".gitattributes"), "README.md filter=poison\n")
	for key := range map[string]string{"GIT_CONFIG_KEY_0": "x", "GIT_CONFIG_VALUE_0": "poison", "GIT_DIR": "poison", "GIT_WORK_TREE": "poison", "GIT_OBJECT_DIRECTORY": "poison", "GIT_ALTERNATE_OBJECT_DIRECTORIES": "poison"} {
		t.Setenv(key, "poison")
	}
	resolver := Resolver{GitPath: proxy}
	for _, scope := range []domain.Scope{{Kind: domain.ScopeRange, Range: "HEAD~1..HEAD"}, {Kind: domain.ScopeInitial, Range: "HEAD"}, {Kind: domain.ScopeStaged}, {Kind: domain.ScopeWorktree}} {
		if scope.Kind == domain.ScopeWorktree {
			write(t, filepath.Join(repo, "README.md"), "worktree\n")
		}
		if _, err := resolver.Resolve(context.Background(), repo, scope); err != nil {
			t.Fatalf("Resolve(%s): %v", scope.Kind, err)
		}
	}
	if _, err := resolver.Radiograph(context.Background(), repo); err != nil {
		t.Fatal(err)
	}
	recorded, err := os.ReadFile(commands)
	if err != nil {
		t.Fatal(err)
	}
	const args = "-c core.attributesfile=/dev/null -c diff.external= -c diff.textconv= -c core.fsmonitor=false -c core.untrackedCache=false -c maintenance.auto=false -c gc.auto=0 -c fetch.writeCommitGraph=false -c credential.helper= -c core.hooksPath=/dev/null -C "
	const env = "1|/dev/null|/dev/null|0|0|1|1|0|cat|/bin/false|||C|UTC||||||"
	seen := map[string]bool{}
	for _, got := range strings.Split(strings.TrimSpace(string(recorded)), "\n") {
		if !strings.HasPrefix(got, args+repo+" ") || !strings.HasSuffix(got, "|"+env) {
			t.Fatalf("unsafe Git command = %q", got)
		}
		for _, operation := range []string{"rev-parse", "diff", "diff-tree", "log", "diff-index", "ls-files", "hash-object", "cat-file"} {
			seen[operation] = seen[operation] || strings.Contains(strings.Split(got, "|")[0], " "+operation+" ")
		}
	}
	for operation, found := range seen {
		if !found {
			t.Fatalf("missing Git operation %q", operation)
		}
	}
	if _, err := os.Stat(marker); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("repository config executed a helper: %v", err)
	}
}

func TestRadiographPathsPreserveFramedPaths(t *testing.T) {
	for _, test := range []struct {
		name string
		raw  string
		want []string
		ok   bool
	}{
		{"LF preserves spaces", "/tmp/git paths/a\n/tmp/git paths/b\n", []string{"/tmp/git paths/a", "/tmp/git paths/b"}, true},
		{"CRLF preserves spaces", "/tmp/git paths/a\r\n/tmp/git paths/b\r\n", []string{"/tmp/git paths/a", "/tmp/git paths/b"}, true},
		{"embedded blank record", "/tmp/a\n\n/tmp/b\n", nil, false},
		{"stray carriage return", "/tmp/a\rbroken\n/tmp/b\n", nil, false},
		{"missing terminal newline", "/tmp/a\n/tmp/b", nil, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, ok := newlinePaths([]byte(test.raw))
			if ok != test.ok || !reflect.DeepEqual(got, test.want) {
				t.Fatalf("newlinePaths(%q) = %#v, %t; want %#v, %t", test.raw, got, ok, test.want, test.ok)
			}
		})
	}
}

func TestResolverRadiographPathsResolveOrdinaryAndLinkedWorktrees(t *testing.T) {
	spacedTMP := filepath.Join(t.TempDir(), "git paths")
	if err := os.Mkdir(spacedTMP, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TMPDIR", spacedTMP)
	ordinary := newRepositoryAt(t, filepath.Join(spacedTMP, "ordinary repository"))
	write(t, filepath.Join(ordinary, "README.md"), "fixture\n")
	runGit(t, ordinary, "add", "README.md")
	runGit(t, ordinary, "commit", "-m", "fixture")
	linked := filepath.Join(spacedTMP, "linked worktree")
	runGit(t, ordinary, "worktree", "add", "-b", "linked", linked)
	primary := filepath.Join(ordinary, ".git", "objects")
	alternate := filepath.Join(spacedTMP, "alternate objects")
	if err := os.MkdirAll(alternate, 0o700); err != nil {
		t.Fatal(err)
	}
	relative, err := filepath.Rel(primary, alternate)
	if err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(primary, "info", "alternates"), relative+"\r\n")
	for _, root := range []string{ordinary, linked} {
		paths, err := (Resolver{GitPath: "git"}).radiographPaths(context.Background(), "git", root)
		if err != nil {
			t.Fatal(err)
		}
		wantGitDir := filepath.Join(ordinary, ".git")
		if root == linked {
			pointer, err := os.ReadFile(filepath.Join(linked, ".git"))
			if err != nil {
				t.Fatal(err)
			}
			wantGitDir = expectedCleanAbsolutePath(t, strings.TrimSuffix(strings.TrimPrefix(string(pointer), "gitdir: "), "\n"))
		}
		if paths.gitDir != wantGitDir || paths.commonDir != filepath.Join(ordinary, ".git") || paths.index != filepath.Join(wantGitDir, "index") || paths.config != filepath.Join(paths.commonDir, "config") || paths.hooks != filepath.Join(paths.commonDir, "hooks") || paths.primaryObjects != primary || !reflect.DeepEqual(paths.alternateObjects, []string{alternate}) {
			t.Fatalf("paths(%q) = %#v", root, paths)
		}
		for _, path := range append([]string{paths.gitDir, paths.commonDir, paths.index, paths.config, paths.hooks, paths.primaryObjects}, paths.alternateObjects...) {
			if !filepath.IsAbs(path) || !strings.Contains(path, "git paths") {
				t.Fatalf("path = %q", path)
			}
		}
	}
}

func TestResolverRadiographPathsRejectsLexicalAliasesAndSymlinks(t *testing.T) {
	gitDir := filepath.Join(t.TempDir(), "git-dir")
	if err := os.MkdirAll(filepath.Join(gitDir, "objects", "info"), 0o700); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(gitDir, "index"), "")
	write(t, filepath.Join(gitDir, "config"), "")
	if err := os.Mkdir(filepath.Join(gitDir, "hooks"), 0o700); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(t.TempDir(), "git-link")
	if err := os.Symlink(gitDir, link); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct{ name, gitDir, commonDir string }{
		{"dot alias", gitDir + string(filepath.Separator) + ".", gitDir},
		{"duplicate separator", strings.Replace(gitDir, string(filepath.Separator), string(filepath.Separator)+string(filepath.Separator), 1), gitDir},
		{"symlink", link, link},
		{"absolute alternate dot alias", gitDir, gitDir},
	} {
		t.Run(test.name, func(t *testing.T) {
			if test.name == "absolute alternate dot alias" {
				write(t, filepath.Join(gitDir, "objects", "info", "alternates"), gitDir+string(filepath.Separator)+"objects"+string(filepath.Separator)+".\n")
			}
			proxy := filepath.Join(t.TempDir(), "git")
			write(t, proxy, "#!/bin/sh\nprintf '%s\\n%s\\n' '"+test.gitDir+"' '"+test.commonDir+"'\n")
			if err := os.Chmod(proxy, 0o700); err != nil {
				t.Fatal(err)
			}
			if _, err := (Resolver{GitPath: proxy}).radiographPaths(context.Background(), proxy, t.TempDir()); !errors.Is(err, domain.ErrContentRead) {
				t.Fatalf("error = %v, want ErrContentRead", err)
			}
		})
	}
}

func expectedCleanAbsolutePath(t *testing.T, path string) string {
	if !filepath.IsAbs(path) || path != filepath.Clean(path) {
		t.Fatalf("path is not clean and absolute: %q", path)
	}
	physical, err := filepath.EvalSymlinks(path)
	if err != nil || physical != path {
		t.Fatalf("path resolves through a symlink: %q, %q, %v", path, physical, err)
	}
	return path
}

func TestResolverRadiographPathsFailClosed(t *testing.T) {
	for _, output := range []string{"relative\n/absolute\n", "/only-one\n", "\n/absolute\n"} {
		t.Run(strings.ReplaceAll(output, "\n", "_"), func(t *testing.T) {
			proxy := filepath.Join(t.TempDir(), "git")
			write(t, proxy, "#!/bin/sh\nprintf '%s' '"+output+"'\n")
			if err := os.Chmod(proxy, 0o700); err != nil {
				t.Fatal(err)
			}
			if _, err := (Resolver{GitPath: proxy}).radiographPaths(context.Background(), proxy, t.TempDir()); !errors.Is(err, domain.ErrContentRead) {
				t.Fatalf("error = %v, want ErrContentRead", err)
			}
		})
	}
}

func TestResolverRadiographPathsRejectMissingResolvedPaths(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing")
	proxy := filepath.Join(t.TempDir(), "git")
	write(t, proxy, "#!/bin/sh\nprintf '%s\\n%s\\n' '"+missing+"' '"+missing+"'\n")
	if err := os.Chmod(proxy, 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := (Resolver{GitPath: proxy}).radiographPaths(context.Background(), proxy, t.TempDir()); !errors.Is(err, domain.ErrContentRead) {
		t.Fatalf("error = %v, want ErrContentRead", err)
	}
}

func TestRadiographSnapshotOraclePreservesOrdinaryAndLinkedWorktrees(t *testing.T) {
	space := filepath.Join(t.TempDir(), "snapshot worktrees")
	if err := os.MkdirAll(space, 0o700); err != nil {
		t.Fatal(err)
	}
	ordinary := newRepositoryAt(t, filepath.Join(space, "ordinary"))
	if err := os.MkdirAll(filepath.Join(ordinary, ".docmanager"), 0o700); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(ordinary, "README.md"), "fixture\n")
	write(t, filepath.Join(ordinary, ".docmanager", "state"), "catalog\n")
	runGit(t, ordinary, "add", ".")
	runGit(t, ordinary, "commit", "-m", "fixture")
	linked := filepath.Join(space, "linked")
	runGit(t, ordinary, "worktree", "add", "-b", "linked", linked)
	primary := filepath.Join(ordinary, ".git", "objects")
	alternate := filepath.Join(space, "alternate objects")
	if err := os.MkdirAll(alternate, 0o700); err != nil {
		t.Fatal(err)
	}
	relative, err := filepath.Rel(primary, alternate)
	if err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(primary, "info", "alternates"), relative+"\n")
	write(t, filepath.Join(alternate, "sentinel"), "alternate\n")

	resolver := Resolver{GitPath: "git"}
	for _, root := range []string{ordinary, linked} {
		t.Run(filepath.Base(root), func(t *testing.T) {
			paths, err := resolver.radiographPaths(context.Background(), "git", root)
			if err != nil {
				t.Fatal(err)
			}
			before, err := snapshotRadiograph(root, paths)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := resolver.Radiograph(context.Background(), root); err != nil {
				t.Fatal(err)
			}
			after, err := snapshotRadiograph(root, paths)
			if err != nil {
				t.Fatal(err)
			}
			if before != after {
				t.Fatalf("Radiograph changed repository state: before %s after %s", before, after)
			}
		})
	}
}

func TestRadiographSnapshotOracleBindsEveryFieldAndFailsClosed(t *testing.T) {
	repo := newRepository(t)
	if err := os.MkdirAll(filepath.Join(repo, ".docmanager"), 0o700); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(repo, "README.md"), "fixture\n")
	write(t, filepath.Join(repo, ".docmanager", "state"), "catalog\n")
	runGit(t, repo, "add", ".")
	runGit(t, repo, "commit", "-m", "fixture")
	alternate := filepath.Join(t.TempDir(), "alternate")
	if err := os.MkdirAll(alternate, 0o700); err != nil {
		t.Fatal(err)
	}
	primary := filepath.Join(repo, ".git", "objects")
	relative, err := filepath.Rel(primary, alternate)
	if err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(primary, "info", "alternates"), relative+"\n")
	paths, err := (Resolver{GitPath: "git"}).radiographPaths(context.Background(), "git", repo)
	if err != nil {
		t.Fatal(err)
	}
	baseline, err := snapshotRadiograph(repo, paths)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name string
		path string
	}{
		{"worktree bytes", filepath.Join(repo, "README.md")},
		{"worktree mode", filepath.Join(repo, "README.md")},
		{"git dir", filepath.Join(paths.gitDir, "oracle")},
		{"common dir", filepath.Join(paths.commonDir, "oracle")},
		{"index", paths.index},
		{"config", paths.config},
		{"hooks", filepath.Join(paths.hooks, "oracle")},
		{"primary objects", filepath.Join(paths.primaryObjects, "oracle")},
		{"alternate objects", filepath.Join(paths.alternateObjects[0], "oracle")},
		{"docmanager", filepath.Join(repo, ".docmanager", "state")},
	} {
		t.Run(test.name, func(t *testing.T) {
			if test.name == "worktree mode" {
				if err := os.Chmod(test.path, 0o700); err != nil {
					t.Fatal(err)
				}
			} else {
				write(t, test.path, test.name)
			}
			changed, err := snapshotRadiograph(repo, paths)
			if err != nil {
				t.Fatal(err)
			}
			if changed == baseline {
				t.Fatal("snapshot did not bind changed field")
			}
		})
	}
	for _, test := range []struct {
		name string
		make func() error
	}{
		{"symlink", func() error { return os.Symlink("README.md", filepath.Join(repo, "linked.md")) }},
		{"special file", func() error { return exec.Command("mkfifo", filepath.Join(repo, "pipe")).Run() }},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := test.make(); err != nil {
				t.Skipf("cannot create unsafe fixture: %v", err)
			}
			if _, err := snapshotRadiograph(repo, paths); err == nil {
				t.Fatal("snapshot accepted unsafe filesystem entry")
			}
		})
	}
	missing := paths
	missing.config = filepath.Join(t.TempDir(), "missing-config")
	if _, err := snapshotRadiograph(repo, missing); err == nil {
		t.Fatal("snapshot accepted an unreadable bound path")
	}
}

func snapshotRadiograph(root string, paths radiographPathSet) (string, error) {
	var canonical bytes.Buffer
	writeField := func(values ...string) {
		for _, value := range values {
			canonical.WriteString(value)
			canonical.WriteByte(0)
		}
	}
	var snapshotTree func(string, string, bool) error
	snapshotTree = func(label, base string, skipGitDir bool) error {
		var walk func(string) error
		walk = func(current string) error {
			relative, err := filepath.Rel(base, current)
			if err != nil {
				return err
			}
			if skipGitDir && relative == ".git" {
				return nil
			}
			info, err := os.Lstat(current)
			if err != nil {
				return err
			}
			if info.Mode()&os.ModeSymlink != 0 || (!info.IsDir() && !info.Mode().IsRegular()) {
				return fmt.Errorf("unsafe snapshot entry %q", current)
			}
			writeField(label, relative, fmt.Sprintf("%o", info.Mode()))
			if info.IsDir() {
				entries, err := os.ReadDir(current)
				if err != nil {
					return err
				}
				for _, entry := range entries {
					if err := walk(filepath.Join(current, entry.Name())); err != nil {
						return err
					}
				}
				return nil
			}
			content, err := os.ReadFile(current)
			if err != nil {
				return err
			}
			sum := sha256.Sum256(content)
			writeField(fmt.Sprintf("%x", sum))
			return nil
		}
		return walk(base)
	}
	if err := snapshotTree("worktree", root, true); err != nil {
		return "", err
	}
	for _, field := range []struct {
		label string
		path  string
	}{
		{"git-dir", paths.gitDir},
		{"common-dir", paths.commonDir},
		{"index", paths.index},
		{"config", paths.config},
		{"hooks", paths.hooks},
		{"primary-objects", paths.primaryObjects},
	} {
		if err := snapshotTree(field.label, field.path, false); err != nil {
			return "", err
		}
	}
	for _, alternate := range paths.alternateObjects {
		if err := snapshotTree("alternate-objects", alternate, false); err != nil {
			return "", err
		}
	}
	sum := sha256.Sum256(canonical.Bytes())
	return fmt.Sprintf("sha256:%x", sum), nil
}

func TestResolverRejectsRootReplacementBeforeEvidence(t *testing.T) {
	repo := newRepository(t)
	write(t, filepath.Join(repo, "README.md"), "fixture\n")
	runGit(t, repo, "add", "README.md")
	runGit(t, repo, "commit", "-m", "fixture")
	replacement := repo + "-replacement"
	proxy := filepath.Join(t.TempDir(), "git-proxy")
	write(t, proxy, "#!/bin/sh\ncase \"$*\" in *'ls-files -s'*) mv "+repo+" "+replacement+"; ln -s "+replacement+" "+repo+" ;; esac\nexec git \"$@\"\n")
	if err := os.Chmod(proxy, 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := (Resolver{GitPath: proxy}).Radiograph(context.Background(), repo); !errors.Is(err, domain.ErrOutsideRepository) {
		t.Fatalf("Radiograph replacement error = %v, want ErrOutsideRepository", err)
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
	return newRepositoryAt(t, t.TempDir())
}

func newRepositoryAt(t *testing.T, repo string) string {
	t.Helper()
	if err := os.MkdirAll(repo, 0o700); err != nil {
		t.Fatal(err)
	}
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
