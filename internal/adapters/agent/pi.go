package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// PiOptions models an already-installed Pi MCP adapter; it never installs packages.
type PiOptions struct {
	Home, Binary, Provenance string
	Installed                func() bool
	AdapterAvailable         func() bool
	Run                      func(context.Context, ...string) error
	ProbeTimeout             time.Duration
}

type Pi struct{ options PiOptions }

func NewPi(options PiOptions) *Pi {
	if options.Provenance == "" {
		options.Provenance = fixtureProvenanceV1
	}
	if options.ProbeTimeout == 0 {
		options.ProbeTimeout = time.Second
	}
	return &Pi{options: options}
}

func piPaths(home string) (string, string, error) {
	if !filepath.IsAbs(home) || filepath.Clean(home) != home {
		return "", "", ErrUnsafeRoute
	}
	info, err := os.Lstat(home)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", "", ErrUnsafeRoute
	}
	root := filepath.Join(home, ".pi", "agent")
	for _, name := range []string{"mcp.json", "AGENTS.md"} {
		path := filepath.Join(root, name)
		if info, err := os.Lstat(path); err == nil && (!info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0) {
			return "", "", ErrUnsafeRoute
		}
	}
	return filepath.Join(root, "mcp.json"), filepath.Join(root, "AGENTS.md"), nil
}

func (a *Pi) Status(ctx context.Context) (Status, error) {
	status := Status{Installed: a.options.Installed != nil && a.options.Installed(), Supported: a.options.Provenance == fixtureProvenanceV1 && a.adapterAvailable()}
	if !status.Supported {
		return status, nil
	}
	config, guide, err := piPaths(a.options.Home)
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
	owned, valid, err := jsonAgentEntry(data, "mcpServers", a.options.Binary, true)
	if err != nil {
		return status, err
	}
	if !valid && jsonAgentHasEntry(data, "mcpServers") {
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

func (a *Pi) Configure(ctx context.Context) error   { return a.mutate(ctx, true) }
func (a *Pi) Unconfigure(ctx context.Context) error { return a.mutate(ctx, false) }

func (a *Pi) mutate(_ context.Context, add bool) error {
	if a.options.Provenance != fixtureProvenanceV1 {
		return ErrUnsupportedConfig
	}
	if !a.adapterAvailable() {
		return ErrPrerequisiteMissing
	}
	if !filepath.IsAbs(a.options.Binary) {
		return ErrUnsafeRoute
	}
	config, guide, err := piPaths(a.options.Home)
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
	return mutateJSONAgent(config, guide, "mcpServers", a.options.Binary, true, add)
}

func (a *Pi) adapterAvailable() bool {
	return a.options.AdapterAvailable != nil && a.options.AdapterAvailable()
}

func (a *Pi) probe(parent context.Context) error {
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
