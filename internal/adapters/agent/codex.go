package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type CodexOptions struct {
	Home, Binary, Provenance string
	Installed                func() bool
	Run                      func(context.Context, ...string) error
	ProbeTimeout             time.Duration
}
type Codex struct{ options CodexOptions }

func NewCodex(options CodexOptions) *Codex {
	if options.Provenance == "" {
		options.Provenance = fixtureProvenanceV1
	}
	if options.ProbeTimeout == 0 {
		options.ProbeTimeout = time.Second
	}
	return &Codex{options}
}
func codexPaths(home string) (string, string, error) {
	if !filepath.IsAbs(home) || filepath.Clean(home) != home {
		return "", "", ErrUnsafeRoute
	}
	root := filepath.Join(home, ".codex")
	if info, err := os.Lstat(root); err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", "", ErrUnsafeRoute
	}
	config := filepath.Join(root, "config.toml")
	if info, err := os.Lstat(config); err == nil && (!info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0) {
		return "", "", ErrUnsafeRoute
	}
	return config, filepath.Join(root, "AGENTS.md"), nil
}

func (a *Codex) Status(ctx context.Context) (Status, error) {
	_ = ctx
	status := Status{Installed: a.options.Installed != nil && a.options.Installed(), Supported: a.options.Provenance == fixtureProvenanceV1}
	if !status.Supported {
		return status, nil
	}
	config, guide, err := codexPaths(a.options.Home)
	if err != nil {
		return status, err
	}
	data, err := os.ReadFile(config)
	if os.IsNotExist(err) {
		return status, nil
	}
	if err != nil {
		return status, err
	}
	owned, valid, err := codexEntry(data, a.options.Binary)
	if err != nil {
		return status, err
	}
	if !valid {
		return status, ErrOwnership
	}
	guidance, err := os.ReadFile(guide)
	if err != nil && !os.IsNotExist(err) {
		return status, err
	}
	status.Configured = owned && strings.Contains(string(guidance), managedGuidanceBegin)
	if !status.Configured || !status.Installed {
		return status, nil
	}
	if err := a.probe(ctx); err != nil {
		return status, err
	}
	status.Healthy = true
	return status, nil
}
func (a *Codex) Configure(ctx context.Context) error   { return a.mutate(ctx, true) }
func (a *Codex) Unconfigure(ctx context.Context) error { return a.mutate(ctx, false) }
func (a *Codex) mutate(_ context.Context, add bool) error {
	if a.options.Provenance != fixtureProvenanceV1 {
		return ErrUnsupportedConfig
	}
	config, guide, err := codexPaths(a.options.Home)
	if err != nil {
		return err
	}
	release, err := lock(filepath.Dir(config))
	if err != nil {
		return err
	}
	defer release()
	data, err := os.ReadFile(config)
	if os.IsNotExist(err) {
		data = nil
	} else if err != nil {
		return err
	}
	owned, valid, err := codexEntry(data, a.options.Binary)
	if err != nil {
		return err
	}
	updated := string(data)
	if add {
		if !valid && strings.Contains(updated, "[mcp_servers.docmanager]") {
			return ErrOwnership
		}
		if valid && !owned {
			return ErrDrift
		}
		if !valid {
			updated = strings.TrimRight(updated, "\n") + "\n\n[mcp_servers.docmanager]\ncommand = \"" + a.options.Binary + "\"\nargs = [\"mcp\"]\n_docmanager = \"managed/v1\"\n"
		}
	} else {
		if !valid {
			if strings.Contains(updated, "[mcp_servers.docmanager]") {
				return ErrDrift
			}
			return nil
		}
		if !owned {
			return ErrDrift
		}
		updated = removeCodexBlock(updated)
	}
	if current, readErr := os.ReadFile(config); readErr == nil && !sameBytes(current, data) {
		return ErrDrift
	}
	if err := atomicWrite(config, []byte(updated)); err != nil {
		return err
	}
	return mutateGuidance(guide, add)
}
func codexEntry(data []byte, binary string) (owned, valid bool, err error) {
	text := string(data)
	if strings.Count(text, "[") != strings.Count(text, "]") {
		return false, false, ErrMalformedConfig
	}
	i := strings.Index(text, "[mcp_servers.docmanager]")
	if i < 0 {
		return false, false, nil
	}
	next := strings.Index(text[i+1:], "\n[")
	block := text[i:]
	if next >= 0 {
		block = text[i : i+1+next]
	}
	valid = strings.Contains(block, `_docmanager = "managed/v1"`)
	owned = strings.Contains(block, `command = "`+binary+`"`) && strings.Contains(block, `args = ["mcp"]`) && strings.Contains(block, `_docmanager = "managed/v1"`)
	return owned, valid, nil
}
func removeCodexBlock(text string) string {
	start := strings.Index(text, "[mcp_servers.docmanager]")
	if start < 0 {
		return text
	}
	end := strings.Index(text[start+1:], "\n[")
	if end < 0 {
		return strings.TrimRight(text[:start], "\n") + "\n"
	}
	return text[:start] + text[start+1+end+1:]
}
func (a *Codex) probe(parent context.Context) error {
	if a.options.Run == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(parent, a.options.ProbeTimeout)
	defer cancel()
	if err := a.options.Run(ctx, a.options.Binary, "mcp", "--version"); err != nil {
		return ErrProbeFailed
	}
	return nil
}
