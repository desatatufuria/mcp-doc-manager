package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// CopilotOptions keeps VS Code configuration routing explicit for isolated callers.
type CopilotOptions struct {
	ConfigRoot, Platform, Binary, Provenance string
	Installed                                func() bool
	Run                                      func(context.Context, ...string) error
	ProbeTimeout                             time.Duration
}

type Copilot struct{ options CopilotOptions }

func NewCopilot(options CopilotOptions) *Copilot {
	if options.Provenance == "" {
		options.Provenance = fixtureProvenanceV1
	}
	if options.ProbeTimeout == 0 {
		options.ProbeTimeout = time.Second
	}
	return &Copilot{options: options}
}

func copilotPaths(root, platform string) (string, string, error) {
	if !filepath.IsAbs(root) || filepath.Clean(root) != root {
		return "", "", ErrUnsafeRoute
	}
	info, err := os.Lstat(root)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", "", ErrUnsafeRoute
	}
	var user string
	switch platform {
	case "linux", "windows":
		user = filepath.Join(root, "Code", "User")
	case "darwin":
		user = filepath.Join(root, "Library", "Application Support", "Code", "User")
	default:
		return "", "", ErrUnsupportedConfig
	}
	for _, name := range []string{"mcp.json", "docmanager.instructions.md"} {
		path := filepath.Join(user, name)
		if info, err := os.Lstat(path); err == nil && (!info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0) {
			return "", "", ErrUnsafeRoute
		}
	}
	return filepath.Join(user, "mcp.json"), filepath.Join(user, "docmanager.instructions.md"), nil
}

func (a *Copilot) Status(ctx context.Context) (Status, error) {
	status := Status{Installed: a.options.Installed != nil && a.options.Installed(), Supported: a.options.Provenance == fixtureProvenanceV1}
	if !status.Supported {
		return status, nil
	}
	config, guide, err := copilotPaths(a.options.ConfigRoot, a.options.Platform)
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
	owned, valid, err := jsonAgentEntry(data, "servers", a.options.Binary, false)
	if err != nil {
		return status, err
	}
	if !valid && jsonAgentHasEntry(data, "servers") {
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

func (a *Copilot) Configure(ctx context.Context) error   { return a.mutate(ctx, true) }
func (a *Copilot) Unconfigure(ctx context.Context) error { return a.mutate(ctx, false) }

func (a *Copilot) mutate(_ context.Context, add bool) error {
	if a.options.Provenance != fixtureProvenanceV1 {
		return ErrUnsupportedConfig
	}
	if !filepath.IsAbs(a.options.Binary) {
		return ErrUnsafeRoute
	}
	config, guide, err := copilotPaths(a.options.ConfigRoot, a.options.Platform)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(config), 0o700); err != nil {
		return err
	}
	release, err := lock(filepath.Dir(config))
	if err != nil {
		return err
	}
	defer release()
	return mutateJSONAgent(config, guide, "servers", a.options.Binary, false, add)
}

func (a *Copilot) probe(parent context.Context) error {
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
