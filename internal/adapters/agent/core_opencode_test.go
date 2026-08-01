package agent

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestOpenCodeRegistryFixturesAndDetection(t *testing.T) {
	registry, err := os.ReadFile("../../../testdata/agent/opencode/registry-v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct{ Schema, Agent string }
	if err := json.Unmarshal(registry, &fixture); err != nil {
		t.Fatal(err)
	}
	if fixture.Schema != fixtureProvenanceV1 || fixture.Agent != "opencode" {
		t.Fatalf("fixture = %#v", fixture)
	}
	for _, name := range []string{"valid.json", "valid.jsonc"} {
		data, err := os.ReadFile(filepath.Join("../../../testdata/agent/opencode", name))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := decodeConfig(data); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}
	root := t.TempDir()
	missing := NewOpenCode(OpenCodeOptions{Root: root, Binary: "/bin/docmanager", Installed: func() bool { return false }})
	status, err := missing.Status(context.Background())
	if err != nil || status.Installed || !status.Supported || status.Configured || status.Healthy {
		t.Fatalf("missing status = %#v, %v", status, err)
	}
	unsupported := NewOpenCode(OpenCodeOptions{Root: root, Provenance: "unknown", Installed: func() bool { return true }})
	status, err = unsupported.Status(context.Background())
	if err != nil || !status.Installed || status.Supported {
		t.Fatalf("unsupported status = %#v, %v", status, err)
	}
}

func TestOpenCodeConfigureAndUnconfigureJSON(t *testing.T) {
	root := t.TempDir()
	config := filepath.Join(root, "opencode.json")
	writeAgentFile(t, config, `{"mcp":{"other":{"command":["other"]}}}`)
	a := NewOpenCode(OpenCodeOptions{Root: root, Binary: "/opt/docmanager", Installed: func() bool { return true }})
	if err := a.Configure(context.Background()); err != nil {
		t.Fatal(err)
	}
	configured, err := a.Status(context.Background())
	if err != nil || !configured.Configured || !configured.Healthy {
		t.Fatalf("status = %#v, %v", configured, err)
	}
	if err := a.Unconfigure(context.Background()); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(config)
	if strings.Contains(string(data), `"docmanager"`) || !strings.Contains(string(data), `"other"`) {
		t.Fatalf("JSON unconfigure = %s", data)
	}
}

func TestOpenCodeConfigureStatusAndUnconfigureJSONC(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	root := filepath.Join(xdg, "opencode")
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Fatal(err)
	}
	config := filepath.Join(root, "opencode.jsonc")
	writeAgentFile(t, config, "// preserved comment\n{\n  \"mcp\": {\"other\": {\"command\": [\"other\"]}}\n}\n")
	runs := 0
	a := NewOpenCode(OpenCodeOptions{Binary: "/opt/docmanager", Installed: func() bool { return true }, Run: func(_ context.Context, argv ...string) error {
		runs++
		if got := strings.Join(argv, "|"); got != "/opt/docmanager|mcp|--version" {
			t.Fatalf("argv = %q", got)
		}
		return nil
	}})

	before, err := a.Status(context.Background())
	if err != nil || !before.Installed || !before.Supported || before.Configured || before.Healthy {
		t.Fatalf("initial status = %#v, %v", before, err)
	}
	if err := a.Configure(context.Background()); err != nil {
		t.Fatal(err)
	}
	configured, err := os.ReadFile(config)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(configured), "// preserved comment") || !strings.Contains(string(configured), `"docmanager"`) || !strings.Contains(string(configured), `"/opt/docmanager"`) || !strings.Contains(string(configured), `"mcp"`) || !strings.Contains(string(configured), managedGuidanceBegin) {
		t.Fatalf("configured JSONC did not preserve/add managed content: %s", configured)
	}
	after, err := a.Status(context.Background())
	if err != nil || !after.Installed || !after.Supported || !after.Configured || !after.Healthy || runs == 0 {
		t.Fatalf("configured status = %#v, %v, runs=%d", after, err, runs)
	}
	if err := a.Configure(context.Background()); err != nil {
		t.Fatalf("idempotent configure: %v", err)
	}
	if err := a.Unconfigure(context.Background()); err != nil {
		t.Fatal(err)
	}
	removed, _ := os.ReadFile(config)
	if strings.Contains(string(removed), `"docmanager"`) || strings.Contains(string(removed), managedGuidanceBegin) || !strings.Contains(string(removed), `"other"`) || !strings.Contains(string(removed), "// preserved comment") {
		t.Fatalf("unconfigure content = %s", removed)
	}
}

func TestOpenCodeRejectsUnsafeStatesWithoutWriting(t *testing.T) {
	for _, tc := range []struct {
		name, content, route string
		setup                func(t *testing.T, root, config string)
		want                 error
	}{
		{name: "malformed", content: `{`, want: ErrMalformedConfig},
		{name: "unknown shape", content: `{"mcp": []}`, want: ErrUnsupportedConfig},
		{name: "conflicting user entry", content: `{"mcp":{"docmanager":{"command":["user"]}}}`, want: ErrOwnership},
		{name: "route escape", route: "relative", want: ErrUnsafeRoute},
		{name: "symlink", setup: func(t *testing.T, root, config string) {
			target := filepath.Join(root, "target.json")
			writeAgentFile(t, target, `{}`)
			if err := os.Symlink(target, config); err != nil {
				t.Fatal(err)
			}
		}, want: ErrUnsafeRoute},
		{name: "lock", content: `{}`, setup: func(t *testing.T, root, _ string) { writeAgentFile(t, filepath.Join(root, ".docmanager.lock"), "held") }, want: ErrLocked},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			config := filepath.Join(root, "opencode.json")
			if tc.content != "" {
				writeAgentFile(t, config, tc.content)
			}
			if tc.setup != nil {
				tc.setup(t, root, config)
			}
			before := snapshotAgent(t, root)
			route := root
			if tc.route != "" {
				route = tc.route
			}
			a := NewOpenCode(OpenCodeOptions{Root: route, Binary: "/bin/docmanager", Installed: func() bool { return true }})
			if err := a.Configure(context.Background()); !errors.Is(err, tc.want) {
				t.Fatalf("Configure error = %v, want %v", err, tc.want)
			}
			if after := snapshotAgent(t, root); after != before {
				t.Fatalf("filesystem changed\nbefore: %s\nafter: %s", before, after)
			}
		})
	}
}

func TestOpenCodeRejectsDriftAndProbeFailuresWithoutWriting(t *testing.T) {
	root := t.TempDir()
	config := filepath.Join(root, "opencode.json")
	writeAgentFile(t, config, `{"mcp":{"docmanager":{"command":["/old/docmanager","mcp"],"_docmanager":"managed/v1"}},"docmanager_guidance":"DOCMANAGER-MANAGED-GUIDANCE-v1"}`)
	a := NewOpenCode(OpenCodeOptions{Root: root, Binary: "/new/docmanager", Installed: func() bool { return true }})
	before := snapshotAgent(t, root)
	if err := a.Configure(context.Background()); !errors.Is(err, ErrDrift) {
		t.Fatalf("drift = %v", err)
	}
	if after := snapshotAgent(t, root); after != before {
		t.Fatal("drift wrote config")
	}

	writeAgentFile(t, config, `{"mcp":{"docmanager":{"command":["/bin/docmanager; echo nope","mcp"],"_docmanager":"managed/v1"}},"docmanager_guidance":"DOCMANAGER-MANAGED-GUIDANCE-v1"}`)
	before = snapshotAgent(t, root)
	for _, run := range []func(context.Context, ...string) error{
		func(context.Context, ...string) error { return errors.New("literal $HOME; must not execute shell") },
		func(context.Context, ...string) error { return context.DeadlineExceeded },
	} {
		a := NewOpenCode(OpenCodeOptions{Root: root, Binary: "/bin/docmanager; echo nope", Installed: func() bool { return true }, Run: run, ProbeTimeout: time.Millisecond})
		if _, err := a.Status(context.Background()); !errors.Is(err, ErrProbeFailed) {
			t.Fatalf("probe = %v", err)
		}
		if after := snapshotAgent(t, root); after != before {
			t.Fatal("probe wrote config")
		}
	}
}

func writeAgentFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
func snapshotAgent(t *testing.T, root string) string {
	t.Helper()
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	var out string
	for _, e := range entries {
		b, _ := os.ReadFile(filepath.Join(root, e.Name()))
		out += e.Name() + ":" + string(b) + "\n"
	}
	return out
}
