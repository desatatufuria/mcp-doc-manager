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

func TestCopilotRoutesAndFixtures(t *testing.T) {
	for _, fixture := range []string{
		"../../../testdata/agent/copilot/registry-v1.json",
		"../../../testdata/agent/copilot/valid.json",
		"../../../testdata/agent/pi/registry-v1.json",
		"../../../testdata/agent/pi/valid.json",
	} {
		data, err := os.ReadFile(fixture)
		if err != nil {
			t.Fatal(err)
		}
		var document map[string]any
		if err := json.Unmarshal(data, &document); err != nil {
			t.Fatalf("%s: %v", fixture, err)
		}
		if document["schema"] != fixtureProvenanceV1 {
			t.Fatalf("%s schema = %#v", fixture, document["schema"])
		}
	}
	root := t.TempDir()
	for _, tc := range []struct {
		name, platform, want string
	}{
		{"linux xdg", "linux", "Code/User/mcp.json"},
		{"darwin library", "darwin", "Library/Application Support/Code/User/mcp.json"},
		{"windows appdata", "windows", "Code/User/mcp.json"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path, guide, err := copilotPaths(root, tc.platform)
			if err != nil {
				t.Fatal(err)
			}
			if got := filepath.ToSlash(strings.TrimPrefix(path, root+string(filepath.Separator))); got != tc.want {
				t.Fatalf("route = %q, want %q", got, tc.want)
			}
			if !strings.HasSuffix(filepath.ToSlash(guide), "Code/User/docmanager.instructions.md") {
				t.Fatalf("guidance route = %q", guide)
			}
		})
	}
}

func TestCopilotConfigureStatusAndUnconfigurePreservesServers(t *testing.T) {
	root := t.TempDir()
	config := filepath.Join(root, "Code", "User", "mcp.json")
	writeAgentFile(t, config, `{"servers":{"other":{"command":"other"}}}`)
	runs := 0
	a := NewCopilot(CopilotOptions{ConfigRoot: root, Platform: "linux", Binary: "/opt/docmanager", Installed: func() bool { return true }, Run: func(_ context.Context, argv ...string) error {
		runs++
		if got := strings.Join(argv, "|"); got != "/opt/docmanager|mcp|--version" {
			t.Fatalf("argv = %q", got)
		}
		return nil
	}})
	if err := a.Configure(context.Background()); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(config)
	guide, _ := os.ReadFile(filepath.Join(root, "Code", "User", "docmanager.instructions.md"))
	if !strings.Contains(string(data), `"other"`) || !strings.Contains(string(data), `"docmanager"`) || !strings.Contains(string(data), `"servers"`) || !strings.Contains(string(guide), managedGuidanceBegin) {
		t.Fatalf("configured config=%s guidance=%s", data, guide)
	}
	status, err := a.Status(context.Background())
	if err != nil || !status.Installed || !status.Supported || !status.Configured || !status.Healthy || runs == 0 {
		t.Fatalf("status=%#v err=%v runs=%d", status, err, runs)
	}
	if err := a.Configure(context.Background()); err != nil {
		t.Fatalf("idempotence: %v", err)
	}
	if err := a.Unconfigure(context.Background()); err != nil {
		t.Fatal(err)
	}
	data, _ = os.ReadFile(config)
	if strings.Contains(string(data), `"docmanager"`) || !strings.Contains(string(data), `"other"`) {
		t.Fatalf("unconfigured = %s", data)
	}
}

func TestPiPrerequisiteConfigureStatusAndUnconfigure(t *testing.T) {
	home := t.TempDir()
	config := filepath.Join(home, ".pi", "agent", "mcp.json")
	writeAgentFile(t, config, `{"mcpServers":{"other":{"command":"other"}}}`)
	missing := NewPi(PiOptions{Home: home, Binary: "/opt/docmanager", Installed: func() bool { return true }, AdapterAvailable: func() bool { return false }})
	status, err := missing.Status(context.Background())
	if err != nil || !status.Installed || status.Supported || status.Configured || status.Healthy {
		t.Fatalf("missing adapter status=%#v err=%v", status, err)
	}
	before := snapshotTree(t, home)
	if err := missing.Configure(context.Background()); !errors.Is(err, ErrPrerequisiteMissing) {
		t.Fatalf("missing prerequisite error=%v", err)
	}
	if after := snapshotTree(t, home); after != before {
		t.Fatal("Pi prerequisite check wrote files")
	}

	a := NewPi(PiOptions{Home: home, Binary: "/opt/docmanager", Installed: func() bool { return true }, AdapterAvailable: func() bool { return true }})
	if err := a.Configure(context.Background()); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(config)
	guide, _ := os.ReadFile(filepath.Join(home, ".pi", "agent", "AGENTS.md"))
	if !strings.Contains(string(data), `"other"`) || !strings.Contains(string(data), `"docmanager"`) || !strings.Contains(string(guide), managedGuidanceBegin) {
		t.Fatalf("configured config=%s guidance=%s", data, guide)
	}
	status, err = a.Status(context.Background())
	if err != nil || !status.Configured || !status.Healthy {
		t.Fatalf("status=%#v err=%v", status, err)
	}
	if err := a.Unconfigure(context.Background()); err != nil {
		t.Fatal(err)
	}
	data, _ = os.ReadFile(config)
	if strings.Contains(string(data), `"docmanager"`) || !strings.Contains(string(data), `"other"`) {
		t.Fatalf("unconfigured = %s", data)
	}
}

func TestPiRejectsDriftSymlinkLockAndProbeWithoutWrites(t *testing.T) {
	for _, tc := range []struct {
		name, content string
		setup         func(*testing.T, string, string)
		operation     func(adapter) error
		want          error
	}{
		{"drift", `{"mcpServers":{"docmanager":{"command":"/old/docmanager","args":["mcp"],"_docmanager":"managed/v1"}}}`, nil, func(a adapter) error { return a.Configure(context.Background()) }, ErrDrift},
		{"symlink", "", func(t *testing.T, root, config string) {
			writeAgentFile(t, filepath.Join(root, "target.json"), `{}`)
			if err := os.MkdirAll(filepath.Dir(config), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(filepath.Join(root, "target.json"), config); err != nil {
				t.Fatal(err)
			}
		}, func(a adapter) error { return a.Configure(context.Background()) }, ErrUnsafeRoute},
		{"lock", `{}`, func(t *testing.T, root, _ string) {
			writeAgentFile(t, filepath.Join(root, ".pi", "agent", ".docmanager.lock"), "held")
		}, func(a adapter) error { return a.Configure(context.Background()) }, ErrLocked},
		{"probe", `{"mcpServers":{"docmanager":{"command":"/bin/docmanager","args":["mcp"],"_docmanager":"managed/v1"}}}`, func(t *testing.T, root, _ string) {
			writeAgentFile(t, filepath.Join(root, ".pi", "agent", "AGENTS.md"), managedGuidanceBegin)
		}, func(a adapter) error { _, err := a.Status(context.Background()); return err }, ErrProbeFailed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			config := filepath.Join(root, ".pi", "agent", "mcp.json")
			if tc.content != "" {
				writeAgentFile(t, config, tc.content)
			}
			if tc.setup != nil {
				tc.setup(t, root, config)
			}
			before := snapshotTree(t, root)
			a := NewPi(PiOptions{Home: root, Binary: "/bin/docmanager", Installed: func() bool { return true }, AdapterAvailable: func() bool { return true }, Run: func(context.Context, ...string) error { return context.DeadlineExceeded }, ProbeTimeout: time.Millisecond})
			if err := tc.operation(a); !errors.Is(err, tc.want) {
				t.Fatalf("error=%v want=%v", err, tc.want)
			}
			if after := snapshotTree(t, root); after != before {
				t.Fatal("Pi unsafe state wrote files")
			}
		})
	}
}

func TestCopilotPiRejectUnsafeStatesAndProbesWithoutWrites(t *testing.T) {
	for _, tc := range []struct {
		name, agent, content string
		setup                func(*testing.T, string, string)
		want                 error
	}{
		{"copilot malformed", "copilot", `{`, nil, ErrMalformedConfig},
		{"copilot conflict", "copilot", `{"servers":{"docmanager":{"command":"user"}}}`, nil, ErrOwnership},
		{"copilot drift", "copilot", `{"servers":{"docmanager":{"command":["/old/docmanager","mcp"],"_docmanager":"managed/v1"}}}`, nil, ErrDrift},
		{"copilot symlink", "copilot", "", func(t *testing.T, root, config string) {
			writeAgentFile(t, filepath.Join(root, "target.json"), `{}`)
			if err := os.MkdirAll(filepath.Dir(config), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(filepath.Join(root, "target.json"), config); err != nil {
				t.Fatal(err)
			}
		}, ErrUnsafeRoute},
		{"copilot lock", "copilot", `{}`, func(t *testing.T, root, _ string) {
			writeAgentFile(t, filepath.Join(root, "Code", "User", ".docmanager.lock"), "held")
		}, ErrLocked},
		{"pi malformed", "pi", `{`, nil, ErrMalformedConfig},
		{"pi conflict", "pi", `{"mcpServers":{"docmanager":{"command":"user"}}}`, nil, ErrOwnership},
		{"pi route", "pi", "", nil, ErrUnsafeRoute},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			var config string
			if tc.agent == "copilot" {
				config = filepath.Join(root, "Code", "User", "mcp.json")
			} else {
				config = filepath.Join(root, ".pi", "agent", "mcp.json")
			}
			if tc.content != "" {
				writeAgentFile(t, config, tc.content)
			}
			if tc.setup != nil {
				tc.setup(t, root, config)
			}
			before := snapshotTree(t, root)
			var err error
			if tc.agent == "copilot" {
				err = NewCopilot(CopilotOptions{ConfigRoot: root, Platform: "linux", Binary: "/new/docmanager"}).Configure(context.Background())
			} else {
				home := root
				if tc.name == "pi route" {
					home = "relative"
				}
				err = NewPi(PiOptions{Home: home, Binary: "/bin/docmanager", AdapterAvailable: func() bool { return true }}).Configure(context.Background())
			}
			if !errors.Is(err, tc.want) {
				t.Fatalf("error=%v want=%v", err, tc.want)
			}
			if after := snapshotTree(t, root); after != before {
				t.Fatal("unsafe state wrote files")
			}
		})
	}

	root := t.TempDir()
	config := filepath.Join(root, "Code", "User", "mcp.json")
	writeAgentFile(t, config, `{"servers":{"docmanager":{"command":["/bin/docmanager;echo-nope","mcp"],"_docmanager":"managed/v1"}}}`)
	writeAgentFile(t, filepath.Join(root, "Code", "User", "docmanager.instructions.md"), managedGuidanceBegin)
	before := snapshotTree(t, root)
	a := NewCopilot(CopilotOptions{ConfigRoot: root, Platform: "linux", Binary: "/bin/docmanager;echo-nope", Installed: func() bool { return true }, Run: func(context.Context, ...string) error { return context.DeadlineExceeded }, ProbeTimeout: time.Millisecond})
	if _, err := a.Status(context.Background()); !errors.Is(err, ErrProbeFailed) {
		t.Fatalf("probe=%v", err)
	}
	if after := snapshotTree(t, root); after != before {
		t.Fatal("probe wrote files")
	}
}
