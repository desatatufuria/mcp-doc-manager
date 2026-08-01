package agent

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type ClaudeOptions struct {
	Home, Binary, Provenance string
	Installed                func() bool
	Run                      func(context.Context, ...string) error
	ProbeTimeout             time.Duration
}
type Claude struct{ options ClaudeOptions }

func NewClaude(options ClaudeOptions) *Claude {
	if options.Provenance == "" {
		options.Provenance = fixtureProvenanceV1
	}
	if options.ProbeTimeout == 0 {
		options.ProbeTimeout = time.Second
	}
	return &Claude{options}
}
func claudePaths(home string) (string, string, error) {
	if !filepath.IsAbs(home) || filepath.Clean(home) != home {
		return "", "", ErrUnsafeRoute
	}
	if info, err := os.Lstat(home); err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", "", ErrUnsafeRoute
	}
	config := filepath.Join(home, ".claude.json")
	if info, err := os.Lstat(config); err == nil && (!info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0) {
		return "", "", ErrUnsafeRoute
	}
	return config, filepath.Join(home, ".claude", "CLAUDE.md"), nil
}
func (a *Claude) Status(ctx context.Context) (Status, error) {
	status := Status{Installed: a.options.Installed != nil && a.options.Installed(), Supported: a.options.Provenance == fixtureProvenanceV1}
	if !status.Supported {
		return status, nil
	}
	config, guide, err := claudePaths(a.options.Home)
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
	owned, valid, parsed, err := claudeEntry(data, a.options.Binary)
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
	status.Configured = owned && parsed && strings.Contains(string(guidance), managedGuidanceBegin)
	if !status.Configured || !status.Installed {
		return status, nil
	}
	if err := a.probe(ctx); err != nil {
		return status, err
	}
	status.Healthy = true
	return status, nil
}
func (a *Claude) Configure(ctx context.Context) error   { return a.mutate(ctx, true) }
func (a *Claude) Unconfigure(ctx context.Context) error { return a.mutate(ctx, false) }
func (a *Claude) mutate(_ context.Context, add bool) error {
	if a.options.Provenance != fixtureProvenanceV1 {
		return ErrUnsupportedConfig
	}
	config, guide, err := claudePaths(a.options.Home)
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
		data = []byte(`{}`)
	} else if err != nil {
		return err
	}
	owned, valid, _, err := claudeEntry(data, a.options.Binary)
	if err != nil {
		return err
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		return ErrMalformedConfig
	}
	servers, ok := doc["mcpServers"].(map[string]any)
	if !ok {
		servers = map[string]any{}
		doc["mcpServers"] = servers
	}
	if add {
		if _, exists := servers["docmanager"]; exists && !valid {
			return ErrOwnership
		}
		if valid && !owned {
			return ErrDrift
		}
		if !valid {
			servers["docmanager"] = map[string]any{"command": a.options.Binary, "args": []string{"mcp"}, "_docmanager": "managed/v1"}
		}
	} else {
		if !valid {
			return nil
		}
		if !owned {
			return ErrDrift
		}
		delete(servers, "docmanager")
	}
	updated, _ := json.MarshalIndent(doc, "", "  ")
	updated = append(updated, '\n')
	if current, readErr := os.ReadFile(config); readErr == nil && !sameBytes(current, data) {
		return ErrDrift
	}
	if err := atomicWrite(config, updated); err != nil {
		return err
	}
	if err := os.Chmod(config, 0o600); err != nil {
		return err
	}
	return mutateGuidance(guide, add)
}
func claudeEntry(data []byte, binary string) (owned, valid, parsed bool, err error) {
	var doc map[string]any
	if err = json.Unmarshal(data, &doc); err != nil {
		return false, false, false, ErrMalformedConfig
	}
	parsed = true
	servers, exists := doc["mcpServers"]
	if !exists {
		return false, false, true, nil
	}
	table, ok := servers.(map[string]any)
	if !ok {
		return false, false, true, ErrUnsupportedConfig
	}
	entry, exists := table["docmanager"]
	if !exists {
		return false, false, true, nil
	}
	_, valid = entry.(map[string]any)["_docmanager"]
	owned, _ = ownedClaudeEntry(entry, binary)
	return owned, valid, true, nil
}
func ownedClaudeEntry(value any, binary string) (bool, bool) {
	entry, ok := value.(map[string]any)
	if !ok {
		return false, false
	}
	command, commandOK := entry["command"].(string)
	marker, markerOK := entry["_docmanager"].(string)
	args, argsOK := entry["args"].([]any)
	return commandOK && markerOK && argsOK && command == binary && marker == "managed/v1" && len(args) == 1 && args[0] == "mcp", commandOK && markerOK && argsOK
}
func (a *Claude) probe(parent context.Context) error {
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
func mutateGuidance(path string, add bool) error {
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	text := string(data)
	if add {
		if strings.Contains(text, managedGuidanceBegin) {
			return nil
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			return err
		}
		return atomicWrite(path, []byte(strings.TrimRight(text, "\n")+"\n\n"+managedGuidanceBegin+"\nUse docmanager mcp for documentation analysis.\n"))
	}
	if !strings.Contains(text, managedGuidanceBegin) {
		return nil
	}
	start := strings.Index(text, managedGuidanceBegin)
	return atomicWrite(path, []byte(strings.TrimRight(text[:start], "\n")+"\n"))
}
