package git

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/desatatufuria/mcp-doc-manager/internal/domain"
)

const maxOutput = 1 << 20

type Resolver struct{ GitPath string }

type rootBinding struct {
	path string
	info os.FileInfo
}

type rootBindingKey struct{}

type Radiography struct {
	Root          string
	Identity      string
	Documentation []DocumentationEvidence
	Exclusions    []Exclusion
	Ownership     OwnershipUncertainty
}

type DocumentationEvidence struct {
	Path           string
	Digest         string
	Classification string
	Evidence       string
}

type Exclusion struct {
	Path     string
	Reason   string
	Evidence string
}

type OwnershipUncertainty struct {
	Uncertain bool
	Reason    string
}

func (r Resolver) ValidateRoot(ctx context.Context, root string) error {
	_, err := r.repositoryRoot(ctx, root)
	return err
}

func (r Resolver) repositoryRoot(ctx context.Context, root string) (rootBinding, error) {
	if !filepath.IsAbs(root) || root != filepath.Clean(root) {
		return rootBinding{}, domain.ErrOutsideRepository
	}
	absRoot := filepath.Clean(root)
	physicalRoot, err := filepath.EvalSymlinks(absRoot)
	if err != nil || physicalRoot != absRoot {
		return rootBinding{}, domain.ErrOutsideRepository
	}
	info, err := os.Lstat(absRoot)
	if err != nil || !info.IsDir() {
		return rootBinding{}, domain.ErrOutsideRepository
	}
	binding := rootBinding{path: physicalRoot, info: info}
	git := r.GitPath
	if git == "" {
		git = "git"
	}
	top, err := r.run(withRootBinding(ctx, binding), git, physicalRoot, "rev-parse", "--show-toplevel")
	if err != nil {
		if errors.Is(err, domain.ErrOutsideRepository) {
			return rootBinding{}, domain.ErrOutsideRepository
		}
		if errors.Is(err, domain.ErrOutputLimit) {
			return rootBinding{}, domain.ErrOutputLimit
		}
		if errors.Is(err, exec.ErrNotFound) || errors.Is(err, os.ErrNotExist) {
			return rootBinding{}, domain.ErrGitUnavailable
		}
		if hasRepositoryChild(physicalRoot) {
			return rootBinding{}, domain.ErrOutsideRepository
		}
		return rootBinding{}, domain.ErrNotRepository
	}
	physicalTop, err := filepath.EvalSymlinks(filepath.Clean(strings.TrimSpace(string(top))))
	if err != nil || physicalRoot != physicalTop {
		return rootBinding{}, domain.ErrOutsideRepository
	}
	if err := sameRoot(binding); err != nil {
		return rootBinding{}, err
	}
	return binding, nil
}

func (r Resolver) Resolve(ctx context.Context, root string, scope domain.Scope) (domain.Evidence, error) {
	if err := scope.Validate(); err != nil {
		return domain.Evidence{}, err
	}
	git := r.GitPath
	if git == "" {
		git = "git"
	}
	binding, err := r.repositoryRoot(ctx, root)
	if err != nil {
		return domain.Evidence{}, err
	}
	ctx = withRootBinding(ctx, binding)
	repoRoot := binding.path
	resolvedScope := scope
	args := []string{"diff", "--no-ext-diff", "--no-textconv", "--no-renames", "--no-prefix", "--name-only", "-z"}
	var paths []string
	switch scope.Kind {
	case domain.ScopeRange:
		resolvedRange, err := r.resolveRange(ctx, git, repoRoot, scope.Range)
		if err != nil {
			return domain.Evidence{}, fmt.Errorf("resolve range: %w", err)
		}
		resolvedScope.Range = resolvedRange
		args = append(args, resolvedRange)
	case domain.ScopeInitial:
		target, err := r.resolveInitial(ctx, git, repoRoot, scope.Range)
		if err != nil {
			return domain.Evidence{}, fmt.Errorf("resolve initial scope: %w", err)
		}
		resolvedScope.Range = target
		pathsRaw, err := r.run(ctx, git, repoRoot, "log", "--root", "--format=", "--no-renames", "--name-only", "-z", target)
		if err != nil {
			return domain.Evidence{}, fmt.Errorf("resolve initial scope: %w", err)
		}
		paths = uniquePaths(nulPaths(pathsRaw))
	case domain.ScopeStaged:
		args = append(args, "--cached")
	}
	if scope.Kind == domain.ScopeWorktree {
		paths, err = r.worktreePaths(ctx, git, repoRoot)
	} else if scope.Kind != domain.ScopeInitial {
		var pathsRaw []byte
		pathsRaw, err = r.run(ctx, git, repoRoot, args...)
		paths = nulPaths(pathsRaw)
	}
	if err != nil {
		if errors.Is(err, domain.ErrOutputLimit) {
			return domain.Evidence{}, domain.ErrOutputLimit
		}
		return domain.Evidence{}, fmt.Errorf("resolve scope: %w", err)
	}
	if len(paths) == 0 {
		return domain.Evidence{}, domain.ErrEmptyScope
	}
	identityRaw, err := r.canonicalIdentity(ctx, git, repoRoot, resolvedScope, paths)
	if err != nil {
		if errors.Is(err, domain.ErrOutputLimit) {
			return domain.Evidence{}, domain.ErrOutputLimit
		}
		return domain.Evidence{}, fmt.Errorf("read identity: %w", domain.ErrContentRead)
	}
	digest := sha256.Sum256(identityRaw)
	evidence := domain.Evidence{Root: repoRoot, Scope: resolvedScope, Identity: "sha256:" + hex.EncodeToString(digest[:]), ChangedPaths: paths}
	report, err := domain.Analyze(evidence)
	if err != nil {
		return domain.Evidence{}, err
	}
	documentationPaths := report.Candidates
	if len(documentationPaths) == 0 {
		documentationPaths = paths
	}
	evidence.DocumentationDigests, err = r.documentationDigests(ctx, git, repoRoot, resolvedScope, documentationPaths)
	if err != nil {
		if errors.Is(err, domain.ErrOutputLimit) {
			return domain.Evidence{}, domain.ErrOutputLimit
		}
		return domain.Evidence{}, fmt.Errorf("read documentation: %w", domain.ErrContentRead)
	}
	return evidence, nil
}

func (r Resolver) documentationDigests(ctx context.Context, git, root string, scope domain.Scope, paths []string) (map[string]string, error) {
	digests := make(map[string]string, len(paths))
	for _, path := range paths {
		if filepath.IsAbs(path) || strings.Contains(path, "\x00") || path == ".." || strings.HasPrefix(path, ".."+string(filepath.Separator)) {
			return nil, domain.ErrContentRead
		}
		var object string
		switch scope.Kind {
		case domain.ScopeRange:
			object = strings.SplitN(scope.Range, "..", 2)[1] + ":" + path
		case domain.ScopeInitial:
			object = scope.Range + ":" + path
		case domain.ScopeStaged:
			object = ":" + path
		case domain.ScopeWorktree:
			fullPath := filepath.Join(root, path)
			resolved, err := filepath.EvalSymlinks(fullPath)
			if errors.Is(err, os.ErrNotExist) {
				digests[path] = "missing"
				continue
			} else if err != nil {
				return nil, err
			}
			relative, err := filepath.Rel(root, resolved)
			if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
				return nil, domain.ErrContentRead
			}
			content, err := os.ReadFile(resolved)
			if err != nil {
				return nil, err
			}
			digest := sha256.Sum256(content)
			digests[path] = "sha256:" + hex.EncodeToString(digest[:])
			continue
		default:
			return nil, domain.ErrUnsupportedRequest
		}
		content, err := r.run(ctx, git, root, "cat-file", "blob", object)
		if err != nil {
			if scope.Kind != domain.ScopeWorktree && errors.As(err, new(*exec.ExitError)) {
				digests[path] = "missing"
				continue
			}
			return nil, err
		}
		digest := sha256.Sum256(content)
		digests[path] = "sha256:" + hex.EncodeToString(digest[:])
	}
	return digests, nil
}

func (r Resolver) canonicalIdentity(ctx context.Context, git, root string, scope domain.Scope, paths []string) ([]byte, error) {
	switch scope.Kind {
	case domain.ScopeRange:
		endpoints := strings.SplitN(scope.Range, "..", 2)
		return r.run(ctx, git, root, "diff-tree", "--no-renames", "--no-commit-id", "-r", "--raw", "-z", endpoints[0], endpoints[1])
	case domain.ScopeInitial:
		return r.run(ctx, git, root, "log", "--root", "--format=", "--no-renames", "--raw", "-z", scope.Range)
	case domain.ScopeStaged:
		return r.run(ctx, git, root, "diff-index", "--cached", "--no-renames", "--raw", "-z", "HEAD")
	case domain.ScopeWorktree:
		return r.noFilterWorktreeIdentity(ctx, git, root, paths)
	default:
		return nil, domain.ErrUnsupportedRequest
	}
}

func (r Resolver) Radiograph(ctx context.Context, root string) (Radiography, error) {
	binding, err := r.repositoryRoot(ctx, root)
	if err != nil {
		return Radiography{}, err
	}
	git := r.GitPath
	if git == "" {
		git = "git"
	}
	return r.radiograph(withRootBinding(ctx, binding), git, binding.path)
}

func (r Resolver) radiograph(ctx context.Context, git, repoRoot string) (Radiography, error) {
	raw, err := r.run(ctx, git, repoRoot, "ls-files", "-s", "-z")
	if err != nil {
		return Radiography{}, err
	}
	entries := map[string]inventoryEntry{}
	for _, rawEntry := range bytes.Split(raw, []byte{0}) {
		parts := bytes.SplitN(rawEntry, []byte{'\t'}, 2)
		fields := bytes.Fields(parts[0])
		if len(parts) == 2 && len(fields) >= 2 {
			entries[string(parts[1])] = inventoryEntry{mode: string(fields[0]), objectID: string(fields[1]), tracked: true}
		}
	}
	untracked, err := r.run(ctx, git, repoRoot, "ls-files", "--others", "--exclude-standard", "-z")
	if err != nil {
		return Radiography{}, err
	}
	for _, rawPath := range bytes.Split(untracked, []byte{0}) {
		file := string(rawPath)
		if file != "" {
			entries[file] = inventoryEntry{mode: "untracked"}
		}
	}
	files := make([]string, 0, len(entries))
	for file := range entries {
		files = append(files, file)
	}
	sort.Strings(files)
	report := Radiography{Root: repoRoot, Ownership: OwnershipUncertainty{Uncertain: true, Reason: "ownership_not_inferred_from_repository_evidence"}}
	for _, file := range files {
		entry := entries[file]
		reason, evidence := radiographyExclusion(file, entry.mode)
		if reason != "" {
			report.Exclusions = append(report.Exclusions, Exclusion{Path: file, Reason: reason, Evidence: evidence})
			continue
		}
		if !isDocumentationPath(file) {
			continue
		}
		digest := ""
		if entry.tracked {
			content, err := r.run(ctx, git, repoRoot, "cat-file", "blob", entry.objectID)
			if err != nil {
				return Radiography{}, fmt.Errorf("read radiography documentation: %w", domain.ErrContentRead)
			}
			sum := sha256.Sum256(content)
			digest = "sha256:" + hex.EncodeToString(sum[:])
		}
		classification, source := documentationClassification(file)
		if !entry.tracked {
			classification, source = "uncertain", "untracked_content_not_read"
		}
		report.Documentation = append(report.Documentation, DocumentationEvidence{Path: file, Digest: digest, Classification: classification, Evidence: source})
	}
	report.Identity = radiographyIdentity(repoRoot, report.Ownership, entries, report.Documentation, report.Exclusions)
	return report, nil
}

type inventoryEntry struct {
	mode, objectID string
	tracked        bool
}

func radiographyExclusion(file, mode string) (string, string) {
	lower := strings.ToLower(file)
	base := strings.ToLower(path.Base(file))
	if !safeInventoryPath(file) || mode == "unavailable" {
		return "unsafe_path", "path_not_safe_to_read"
	}
	if base == "requirements.txt" || base == "cmakelists.txt" || base == "readme.sh" || (!strings.HasSuffix(base, ".md") && !strings.HasSuffix(base, ".mdx")) {
		return "non_documentation_path", "path_extension_or_known_non_documentation_name"
	}
	if strings.HasPrefix(mode, "120") || strings.HasPrefix(mode, "L") {
		return "symlink_documentation", "symlink_mode"
	}
	for _, component := range strings.Split(lower, "/") {
		if component == "vendor" || component == "generated" || component == "gen" {
			return "generated_or_vendor", "generated_or_vendor_directory"
		}
	}
	if strings.HasPrefix(mode, "1007") || strings.HasPrefix(mode, "1005") || strings.Contains(mode, "x") {
		return "executable_documentation", "executable_mode"
	}
	return "", ""
}

func safeInventoryPath(file string) bool {
	clean := path.Clean(file)
	return file != "" && !filepath.IsAbs(file) && !path.IsAbs(file) && !strings.Contains(file, "\x00") && clean != ".." && !strings.HasPrefix(clean, "../")
}
func isDocumentationPath(file string) bool {
	lower := strings.ToLower(path.Base(file))
	return strings.HasSuffix(lower, ".md") || strings.HasSuffix(lower, ".mdx")
}
func documentationClassification(file string) (string, string) {
	if strings.Contains(strings.ToLower(path.Base(file)), ".generated.") {
		return "uncertain", "generated_name_heuristic"
	}
	return "uncertain", "maintenance_evidence_absent"
}

func radiographyIdentity(root string, ownership OwnershipUncertainty, entries map[string]inventoryEntry, documentation []DocumentationEvidence, exclusions []Exclusion) string {
	files := make([]string, 0, len(entries))
	for file := range entries {
		files = append(files, file)
	}
	sort.Strings(files)
	var canonical bytes.Buffer
	write := func(values ...string) {
		for _, value := range values {
			canonical.WriteString(value)
			canonical.WriteByte(0)
		}
	}
	write(root, fmt.Sprintf("%t", ownership.Uncertain), ownership.Reason)
	for _, file := range files {
		entry := entries[file]
		write(file, entry.mode, entry.objectID, fmt.Sprintf("%t", entry.tracked))
	}
	for _, document := range documentation {
		write("documentation", document.Path, document.Digest, document.Classification, document.Evidence)
	}
	for _, exclusion := range exclusions {
		write("exclusion", exclusion.Path, exclusion.Reason, exclusion.Evidence)
	}
	sum := sha256.Sum256(canonical.Bytes())
	return "sha256:" + hex.EncodeToString(sum[:])
}

func (r Resolver) resolveInitial(ctx context.Context, git, root, raw string) (string, error) {
	if strings.TrimSpace(raw) == "" || strings.Contains(raw, "..") {
		return "", domain.ErrInvalidScope
	}
	objectID, err := r.run(ctx, git, root, "rev-parse", "--verify", raw+"^{commit}")
	if err != nil {
		return "", domain.ErrInvalidRevision
	}
	return strings.TrimSpace(string(objectID)), nil
}

func uniquePaths(paths []string) []string {
	seen := make(map[string]struct{}, len(paths))
	result := make([]string, 0, len(paths))
	for _, path := range paths {
		if path != "" {
			seen[path] = struct{}{}
		}
	}
	for path := range seen {
		result = append(result, path)
	}
	sort.Strings(result)
	return result
}

func (r Resolver) worktreePaths(ctx context.Context, git, root string) ([]string, error) {
	raw, err := r.run(ctx, git, root, "ls-files", "-s", "-z")
	if err != nil {
		return nil, err
	}
	var paths []string
	for _, item := range bytes.Split(raw, []byte{0}) {
		parts := bytes.SplitN(item, []byte{'\t'}, 2)
		if len(parts) != 2 {
			continue
		}
		path := string(parts[1])
		if _, err := os.Lstat(filepath.Join(root, path)); errors.Is(err, os.ErrNotExist) {
			paths = append(paths, path)
			continue
		} else if err != nil {
			return nil, domain.ErrContentRead
		}
		objectID, err := r.run(ctx, git, root, "hash-object", "--no-filters", "--", path)
		if errors.Is(err, domain.ErrOutputLimit) {
			return nil, err
		}
		if err != nil {
			return nil, domain.ErrContentRead
		}
		fields := bytes.Fields(parts[0])
		if len(fields) > 1 && string(bytes.TrimSpace(objectID)) != string(fields[1]) {
			paths = append(paths, path)
		}
	}
	return paths, nil
}

func (r Resolver) noFilterWorktreeIdentity(ctx context.Context, git, root string, paths []string) ([]byte, error) {
	paths = append([]string(nil), paths...)
	sort.Strings(paths)
	var canonical bytes.Buffer
	for _, path := range paths {
		canonical.WriteString(path)
		canonical.WriteByte(0)
		if _, err := os.Lstat(filepath.Join(root, path)); errors.Is(err, os.ErrNotExist) {
			canonical.WriteString("deleted")
			canonical.WriteByte(0)
			continue
		} else if err != nil {
			return nil, err
		}
		objectID, err := r.run(ctx, git, root, "hash-object", "--no-filters", "--", path)
		if err != nil {
			return nil, err
		}
		canonical.Write(bytes.TrimSpace(objectID))
		canonical.WriteByte(0)
	}
	return canonical.Bytes(), nil
}

func (r Resolver) resolveRange(ctx context.Context, git, root, raw string) (string, error) {
	if strings.Count(raw, "..") != 1 {
		return "", domain.ErrInvalidScope
	}
	parts := strings.SplitN(raw, "..", 2)
	if parts[0] == "" || parts[1] == "" {
		return "", domain.ErrInvalidScope
	}
	resolved := make([]string, 2)
	for i, endpoint := range parts {
		objectID, err := r.run(ctx, git, root, "rev-parse", "--verify", endpoint+"^{commit}")
		if err != nil {
			return "", domain.ErrInvalidRevision
		}
		resolved[i] = strings.TrimSpace(string(objectID))
	}
	return strings.Join(resolved, ".."), nil
}

func (r Resolver) run(parent context.Context, git, root string, args ...string) ([]byte, error) {
	if binding, ok := parent.Value(rootBindingKey{}).(rootBinding); ok {
		if binding.path != root {
			return nil, domain.ErrOutsideRepository
		}
		if err := sameRoot(binding); err != nil {
			return nil, err
		}
	}
	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	defer cancel()
	gitPath, err := exec.LookPath(git)
	if err != nil {
		return nil, err
	}
	cmdArgs := append([]string{"-c", "core.attributesfile=/dev/null", "-c", "diff.external=", "-c", "diff.textconv=", "-c", "core.fsmonitor=false", "-c", "core.untrackedCache=false", "-c", "maintenance.auto=false", "-c", "gc.auto=0", "-c", "fetch.writeCommitGraph=false", "-c", "credential.helper=", "-c", "core.hooksPath=/dev/null", "-C", root}, args...)
	cmd := exec.CommandContext(ctx, gitPath, cmdArgs...)
	cmd.Env = []string{
		"GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null",
		"GIT_CONFIG_COUNT=0", "GIT_OPTIONAL_LOCKS=0", "GIT_NO_LAZY_FETCH=1", "GIT_NO_REPLACE_OBJECTS=1",
		"GIT_TERMINAL_PROMPT=0", "GIT_ASKPASS=/bin/false", "GIT_PAGER=cat", "GIT_EXTERNAL_DIFF=", "GIT_DIFF_OPTS=", "LC_ALL=C", "TZ=UTC",
	}
	out := &boundedBuffer{limit: maxOutput}
	cmd.Stdout, cmd.Stderr = out, out
	err = cmd.Run()
	if binding, ok := parent.Value(rootBindingKey{}).(rootBinding); ok {
		if rootErr := sameRoot(binding); rootErr != nil {
			return nil, rootErr
		}
	}
	if errors.Is(out.err, domain.ErrOutputLimit) {
		return nil, domain.ErrOutputLimit
	}
	return out.Bytes(), err
}

func withRootBinding(ctx context.Context, binding rootBinding) context.Context {
	return context.WithValue(ctx, rootBindingKey{}, binding)
}

func sameRoot(binding rootBinding) error {
	info, err := os.Lstat(binding.path)
	if err != nil || !info.IsDir() || !os.SameFile(binding.info, info) {
		return domain.ErrOutsideRepository
	}
	return nil
}

type boundedBuffer struct {
	mu    sync.Mutex
	buf   bytes.Buffer
	limit int
	err   error
}

func (b *boundedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	remaining := b.limit - b.buf.Len()
	if remaining <= 0 {
		b.err = domain.ErrOutputLimit
		return 0, b.err
	}
	if len(p) > remaining {
		_, _ = b.buf.Write(p[:remaining])
		b.err = domain.ErrOutputLimit
		return remaining, b.err
	}
	return b.buf.Write(p)
}

func (b *boundedBuffer) Bytes() []byte {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]byte(nil), b.buf.Bytes()...)
}

func (b *boundedBuffer) Len() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Len()
}

func nulPaths(raw []byte) []string {
	var paths []string
	for _, item := range strings.Split(string(raw), "\x00") {
		if item != "" {
			paths = append(paths, item)
		}
	}
	return paths
}

func hasRepositoryChild(root string) bool {
	entries, err := os.ReadDir(root)
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if entry.IsDir() {
			if _, err := os.Stat(filepath.Join(root, entry.Name(), ".git")); err == nil {
				return true
			}
		}
	}
	return false
}
