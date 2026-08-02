package agent

import (
	"context"
	"os"
	"path/filepath"
	"time"
)

type OpenCode struct{ options OpenCodeOptions }

func NewOpenCode(options OpenCodeOptions) *OpenCode {
	if options.Provenance == "" {
		options.Provenance = fixtureProvenanceV1
	}
	if options.ProbeTimeout == 0 {
		options.ProbeTimeout = time.Second
	}
	return &OpenCode{options: options}
}

func (a *OpenCode) Status(ctx context.Context) (Status, error) {
	status, err := a.Inspect(ctx)
	if err != nil || !status.Configured || !status.Installed {
		return status, err
	}
	if err := a.probe(ctx); err != nil {
		return status, err
	}
	status.Healthy = true
	return status, nil
}

// Inspect performs the complete read-only preflight without running the configured command.
func (a *OpenCode) Inspect(_ context.Context) (Status, error) {
	installed := a.options.Installed != nil && a.options.Installed()
	status := Status{Installed: installed, Supported: a.options.Provenance == fixtureProvenanceV1}
	if !status.Supported {
		return status, nil
	}
	route, err := openCodeRoute(a.options.Root)
	if err != nil {
		return status, err
	}
	if !route.exists {
		return status, nil
	}
	data, err := os.ReadFile(route.path)
	if os.IsNotExist(err) {
		return status, nil
	}
	if err != nil {
		return status, err
	}
	config, err := decodeOpenCodeConfig(data, filepath.Ext(route.path) == ".jsonc")
	if err != nil {
		return status, err
	}
	status.Configured, err = managedConfig(config, a.options.Binary)
	if err != nil {
		return status, err
	}
	return status, nil
}

func (a *OpenCode) Configure(ctx context.Context) error   { return a.mutate(ctx, true) }
func (a *OpenCode) Unconfigure(ctx context.Context) error { return a.mutate(ctx, false) }

func (a *OpenCode) mutate(ctx context.Context, configure bool) error {
	if a.options.Provenance != fixtureProvenanceV1 {
		return ErrUnsupportedConfig
	}
	route, err := openCodeRoute(a.options.Root)
	if err != nil {
		return err
	}
	if route.prospective {
		if err := os.Mkdir(route.root, 0o700); err != nil {
			return ErrUnsafeRoute
		}
	}
	release, err := lock(route.root)
	if err != nil {
		return err
	}
	defer release()
	data, err := os.ReadFile(route.path)
	if os.IsNotExist(err) {
		data = []byte(`{}`)
	} else if err != nil {
		return err
	}
	config, err := decodeOpenCodeConfig(data, filepath.Ext(route.path) == ".jsonc")
	if err != nil {
		return err
	}
	mcp := config["mcp"].(map[string]any)
	if configure {
		if current, found := mcp["docmanager"]; found {
			owned, valid := ownedEntry(current, a.options.Binary)
			if !owned || !valid {
				return ErrOwnership
			}
		} else {
			mcp["docmanager"] = map[string]any{"type": "local", "command": []string{a.options.Binary, "mcp"}, "enabled": true}
		}
	} else {
		current, found := mcp["docmanager"]
		if !found {
			return nil
		}
		owned, valid := ownedEntry(current, a.options.Binary)
		if !owned || !valid {
			return ErrOwnership
		}
		delete(mcp, "docmanager")
	}
	updated, err := encodeConfig(config, filepath.Ext(route.path) == ".jsonc", leadingComments(data))
	if err != nil {
		return err
	}
	if sameBytes(updated, data) {
		return nil
	}
	current, err := os.ReadFile(route.path)
	if err == nil && !sameBytes(current, data) {
		return ErrDrift
	}
	return atomicWrite(route.path, updated)
}

func (a *OpenCode) probe(parent context.Context) error {
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
func managedConfig(config map[string]any, binary string) (bool, error) {
	current, found := config["mcp"].(map[string]any)["docmanager"]
	if !found {
		return false, nil
	}
	owned, valid := ownedEntry(current, binary)
	if !valid || !owned {
		return false, ErrOwnership
	}
	return true, nil
}
func ownedEntry(value any, binary string) (owned, valid bool) {
	entry, ok := value.(map[string]any)
	if !ok {
		return false, false
	}
	if len(entry) != 3 {
		return false, true
	}
	typeName, typeOK := entry["type"].(string)
	enabled, enabledOK := entry["enabled"].(bool)
	raw, ok := entry["command"].([]any)
	if !ok || len(raw) != 2 {
		return false, false
	}
	first, firstOK := raw[0].(string)
	second, secondOK := raw[1].(string)
	valid = typeOK && enabledOK && firstOK && secondOK
	return valid && typeName == "local" && enabled && first == binary && second == "mcp", valid
}

func decodeOpenCodeConfig(data []byte, jsonc bool) (map[string]any, error) {
	if !jsonc {
		return decodeConfig(data)
	}
	comments := leadingComments(data)
	return decodeConfig(data[len(comments):])
}
func lock(root string) (func(), error) {
	path := filepath.Join(root, ".docmanager.lock")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if os.IsExist(err) {
		return nil, ErrLocked
	}
	if err != nil {
		return nil, err
	}
	_ = file.Close()
	return func() { _ = os.Remove(path) }, nil
}
func atomicWrite(path string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".opencode.*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name)
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(name, path)
}
