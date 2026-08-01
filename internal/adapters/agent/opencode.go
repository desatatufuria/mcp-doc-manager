package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
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
	installed := a.options.Installed != nil && a.options.Installed()
	status := Status{Installed: installed, Supported: a.options.Provenance == fixtureProvenanceV1}
	if !status.Supported {
		return status, nil
	}
	path, err := openCodeRoute(a.options.Root)
	if err != nil {
		return status, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return status, nil
	}
	if err != nil {
		return status, err
	}
	config, err := decodeConfig(data)
	if err != nil {
		return status, err
	}
	status.Configured, err = managedConfig(config, a.options.Binary)
	if err != nil {
		return status, err
	}
	if !status.Configured || !installed {
		return status, nil
	}
	if err := a.probe(ctx); err != nil {
		return status, err
	}
	status.Healthy = true
	return status, nil
}

func (a *OpenCode) Configure(ctx context.Context) error   { return a.mutate(ctx, true) }
func (a *OpenCode) Unconfigure(ctx context.Context) error { return a.mutate(ctx, false) }

func (a *OpenCode) mutate(ctx context.Context, configure bool) error {
	if a.options.Provenance != fixtureProvenanceV1 {
		return ErrUnsupportedConfig
	}
	path, err := openCodeRoute(a.options.Root)
	if err != nil {
		return err
	}
	release, err := lock(filepath.Dir(path))
	if err != nil {
		return err
	}
	defer release()
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		data = []byte(`{}`)
	} else if err != nil {
		return err
	}
	config, err := decodeConfig(data)
	if err != nil {
		return err
	}
	mcp := config["mcp"].(map[string]any)
	if configure {
		if current, found := mcp["docmanager"]; found {
			owned, valid := ownedEntry(current, a.options.Binary)
			if !owned {
				if valid {
					return ErrDrift
				}
				return ErrOwnership
			}
		} else {
			mcp["docmanager"] = map[string]any{"command": []string{a.options.Binary, "mcp"}, "_docmanager": "managed/v1"}
		}
		config["docmanager_guidance"] = managedGuidanceBegin + "\nUse docmanager mcp for documentation analysis."
	} else {
		current, found := mcp["docmanager"]
		if !found {
			return nil
		}
		owned, _ := ownedEntry(current, a.options.Binary)
		if !owned {
			return ErrDrift
		}
		delete(mcp, "docmanager")
		if text, _ := config["docmanager_guidance"].(string); strings.HasPrefix(text, managedGuidanceBegin) {
			delete(config, "docmanager_guidance")
		} else {
			return ErrDrift
		}
	}
	updated, err := encodeConfig(config, filepath.Ext(path) == ".jsonc", leadingComments(data))
	if err != nil {
		return err
	}
	current, err := os.ReadFile(path)
	if err == nil && !sameBytes(current, data) {
		return ErrDrift
	}
	return atomicWrite(path, updated)
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
	if !valid {
		return false, ErrOwnership
	}
	guidance, _ := config["docmanager_guidance"].(string)
	return owned && strings.HasPrefix(guidance, managedGuidanceBegin), nil
}
func ownedEntry(value any, binary string) (owned, valid bool) {
	entry, ok := value.(map[string]any)
	if !ok {
		return false, false
	}
	marker, _ := entry["_docmanager"].(string)
	raw, ok := entry["command"].([]any)
	if !ok || len(raw) != 2 {
		return false, false
	}
	first, firstOK := raw[0].(string)
	second, secondOK := raw[1].(string)
	return marker == "managed/v1" && first == binary && second == "mcp", firstOK && secondOK
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
