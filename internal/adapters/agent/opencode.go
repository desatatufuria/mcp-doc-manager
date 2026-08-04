package agent

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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
	if a.options.Guidance.Path != "" {
		status.Configured, err = managedPairConfig(config, a.options.Binary, a.options.Guidance)
	} else {
		status.Configured, err = managedConfig(config, a.options.Binary)
	}
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
	if a.options.Guidance.Path != "" {
		return a.mutatePair(route, data, config, mcp, configure)
	}
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

func (a *OpenCode) mutatePair(route route, data []byte, config, mcp map[string]any, configure bool) error {
	if err := validGuidance(a.options.Guidance); err != nil {
		return err
	}
	instructions, _, err := configInstructions(config)
	if err != nil {
		return err
	}
	entry, hasMCP := mcp["docmanager"]
	pathIndex := -1
	for index, value := range instructions {
		if value == a.options.Guidance.Path {
			pathIndex = index
		}
	}
	ownedMCP, validMCP := ownedEntry(entry, a.options.Binary)
	if hasMCP && (!validMCP || !ownedMCP) {
		return ErrOwnership
	}
	if configure {
		if hasMCP != (pathIndex >= 0) {
			return ErrOwnership
		}
		if hasMCP {
			return nil
		}
		mcp["docmanager"] = map[string]any{"type": "local", "command": []string{a.options.Binary, "mcp"}, "enabled": true}
		instructions = append(instructions, a.options.Guidance.Path)
		config["instructions"] = instructions
	} else {
		if !hasMCP && pathIndex < 0 {
			return nil
		}
		if !hasMCP || pathIndex < 0 {
			return ErrOwnership
		}
		delete(mcp, "docmanager")
		config["instructions"] = append(instructions[:pathIndex], instructions[pathIndex+1:]...)
	}
	updated, err := encodeConfig(config, filepath.Ext(route.path) == ".jsonc", leadingComments(data))
	if err != nil {
		return err
	}
	current, err := os.ReadFile(route.path)
	if err == nil && !sameBytes(current, data) {
		return ErrDrift
	}
	return atomicWrite(route.path, updated)
}

func configInstructions(config map[string]any) ([]any, bool, error) {
	raw, found := config["instructions"]
	if !found {
		return nil, false, nil
	}
	instructions, ok := raw.([]any)
	if !ok {
		return nil, false, ErrUnsupportedConfig
	}
	for _, value := range instructions {
		if _, ok := value.(string); !ok {
			return nil, false, ErrUnsupportedConfig
		}
	}
	return append([]any(nil), instructions...), true, nil
}

func validGuidance(identity GuidanceIdentity) error {
	if identity.Path == "" || identity.Version == "" || identity.Digest == "" || !filepath.IsAbs(identity.Path) || filepath.Clean(identity.Path) != identity.Path {
		return ErrOwnership
	}
	info, err := os.Lstat(identity.Path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return ErrOwnership
	}
	content, err := os.ReadFile(identity.Path)
	if err != nil {
		return ErrOwnership
	}
	digest := sha256.Sum256(content)
	if hex.EncodeToString(digest[:]) != identity.Digest {
		return ErrDrift
	}
	return nil
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

func managedPairConfig(config map[string]any, binary string, guidance GuidanceIdentity) (bool, error) {
	entry, hasMCP := config["mcp"].(map[string]any)["docmanager"]
	instructions, _, err := configInstructions(config)
	if err != nil {
		return false, err
	}
	matches := 0
	for _, instruction := range instructions {
		if instruction == guidance.Path {
			matches++
		}
	}
	if !hasMCP && matches == 0 {
		return false, nil
	}
	owned, valid := ownedEntry(entry, binary)
	if !hasMCP || matches != 1 || !owned || !valid {
		return false, ErrOwnership
	}
	if err := validGuidance(guidance); err != nil {
		return false, err
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
