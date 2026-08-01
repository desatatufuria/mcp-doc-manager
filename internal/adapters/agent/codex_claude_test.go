package agent

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCodexConfigureStatusAndUnconfigure(t *testing.T) {
	home := t.TempDir()
	config := filepath.Join(home, ".codex", "config.toml")
	writeAgentFile(t, config, "model = \"o3\"\n\n[mcp_servers.other]\ncommand = \"other\"\n")
	a := NewCodex(CodexOptions{Home: home, Binary: "/opt/docmanager", Installed: func() bool { return true }})
	if err := a.Configure(context.Background()); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(config)
	guidance, _ := os.ReadFile(filepath.Join(home, ".codex", "AGENTS.md"))
	if !strings.Contains(string(data), "model = \"o3\"") || !strings.Contains(string(data), "[mcp_servers.docmanager]") || !strings.Contains(string(data), `command = "/opt/docmanager"`) || !strings.Contains(string(guidance), managedGuidanceBegin) {
		t.Fatalf("configured = %s", data)
	}
	status, err := a.Status(context.Background())
	if err != nil || !status.Installed || !status.Supported || !status.Configured || !status.Healthy {
		t.Fatalf("status=%#v err=%v", status, err)
	}
	if err := a.Configure(context.Background()); err != nil {
		t.Fatalf("idempotence: %v", err)
	}
	if err := a.Unconfigure(context.Background()); err != nil {
		t.Fatal(err)
	}
	data, _ = os.ReadFile(config)
	guidance, _ = os.ReadFile(filepath.Join(home, ".codex", "AGENTS.md"))
	if strings.Contains(string(data), "docmanager") || !strings.Contains(string(data), "[mcp_servers.other]") || strings.Contains(string(guidance), managedGuidanceBegin) {
		t.Fatalf("unconfigured = %s", data)
	}
}

func TestClaudeConfigureStatusAndUnconfigurePreservesModeAndOAuth(t *testing.T) {
	home := t.TempDir()
	config := filepath.Join(home, ".claude.json")
	writeAgentFile(t, config, `{"oauthAccount":{"token":"keep"},"mcpServers":{"other":{"command":"other"}}}`)
	if err := os.Chmod(config, 0o600); err != nil {
		t.Fatal(err)
	}
	a := NewClaude(ClaudeOptions{Home: home, Binary: "/opt/docmanager", Installed: func() bool { return true }})
	if err := a.Configure(context.Background()); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(config)
	guidance, _ := os.ReadFile(filepath.Join(home, ".claude", "CLAUDE.md"))
	info, _ := os.Stat(config)
	if !strings.Contains(string(data), `"oauthAccount"`) || !strings.Contains(string(data), `"docmanager"`) || !strings.Contains(string(guidance), managedGuidanceBegin) || info.Mode().Perm() != 0o600 {
		t.Fatalf("configured=%s mode=%o", data, info.Mode().Perm())
	}
	status, err := a.Status(context.Background())
	if err != nil || !status.Configured || !status.Healthy {
		t.Fatalf("status=%#v err=%v", status, err)
	}
	if err := a.Unconfigure(context.Background()); err != nil {
		t.Fatal(err)
	}
	data, _ = os.ReadFile(config)
	if strings.Contains(string(data), `"docmanager"`) || !strings.Contains(string(data), `"oauthAccount"`) {
		t.Fatalf("unconfigured=%s", data)
	}
}

func TestCodexClaudeRefuseDriftRouteAndProbeWithoutWrites(t *testing.T) {
	for _, tc := range []struct {
		name        string
		makeAdapter func(string) adapter
		setup       func(*testing.T, string)
		want        error
	}{
		{"codex drift", func(home string) adapter { return NewCodex(CodexOptions{Home: home, Binary: "/new/docmanager"}) }, func(t *testing.T, home string) {
			writeAgentFile(t, filepath.Join(home, ".codex", "config.toml"), "[mcp_servers.docmanager]\ncommand = \"/old/docmanager\"\nargs = [\"mcp\"]\n_docmanager = \"managed/v1\"\n")
		}, ErrDrift},
		{"claude route", func(string) adapter { return NewClaude(ClaudeOptions{Home: "relative", Binary: "/bin/docmanager"}) }, nil, ErrUnsafeRoute},
		{"claude probe", func(home string) adapter {
			return NewClaude(ClaudeOptions{Home: home, Binary: "/bin/docmanager", Installed: func() bool { return true }, Run: func(context.Context, ...string) error { return errors.New("failed") }})
		}, func(t *testing.T, home string) {
			writeAgentFile(t, filepath.Join(home, ".claude.json"), `{"mcpServers":{"docmanager":{"command":"/bin/docmanager","args":["mcp"],"_docmanager":"managed/v1"}}}`)
			writeAgentFile(t, filepath.Join(home, ".claude", "CLAUDE.md"), managedGuidanceBegin)
		}, ErrProbeFailed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			if tc.setup != nil {
				tc.setup(t, home)
			}
			before := snapshotTree(t, home)
			a := tc.makeAdapter(home)
			var err error
			if tc.name == "codex drift" {
				err = a.Configure(context.Background())
			} else {
				_, err = a.Status(context.Background())
			}
			if !errors.Is(err, tc.want) {
				t.Fatalf("error=%v", err)
			}
			if after := snapshotTree(t, home); after != before {
				t.Fatal("status wrote files")
			}
		})
	}
}

type adapter interface {
	Configure(context.Context) error
	Unconfigure(context.Context) error
	Status(context.Context) (Status, error)
}

func TestCodexClaudeRejectUnsafeStatesWithoutWrites(t *testing.T) {
	for _, tc := range []struct {
		name, agent, content string
		setup                func(string)
		want                 error
	}{
		{"codex malformed", "codex", "[mcp_servers", nil, ErrMalformedConfig},
		{"claude malformed", "claude", `{`, nil, ErrMalformedConfig},
		{"codex conflict", "codex", "[mcp_servers.docmanager]\ncommand = \"user\"\n", nil, ErrOwnership},
		{"claude conflict", "claude", `{"mcpServers":{"docmanager":{"command":"user"}}}`, nil, ErrOwnership},
		{"codex lock", "codex", "", func(root string) { writeAgentFile(t, filepath.Join(root, ".codex", ".docmanager.lock"), "held") }, ErrLocked},
		{"claude symlink", "claude", "", func(root string) {
			target := filepath.Join(root, "target")
			writeAgentFile(t, target, `{}`)
			if err := os.Symlink(target, filepath.Join(root, ".claude.json")); err != nil {
				t.Fatal(err)
			}
		}, ErrUnsafeRoute},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			var config string
			if tc.agent == "codex" {
				config = filepath.Join(home, ".codex", "config.toml")
			} else {
				config = filepath.Join(home, ".claude.json")
			}
			if tc.content != "" {
				writeAgentFile(t, config, tc.content)
			}
			if tc.setup != nil {
				tc.setup(home)
			}
			before := snapshotTree(t, home)
			var err error
			if tc.agent == "codex" {
				err = NewCodex(CodexOptions{Home: home, Binary: "/bin/docmanager"}).Configure(context.Background())
			} else {
				err = NewClaude(ClaudeOptions{Home: home, Binary: "/bin/docmanager"}).Configure(context.Background())
			}
			if !errors.Is(err, tc.want) {
				t.Fatalf("error=%v want=%v", err, tc.want)
			}
			if after := snapshotTree(t, home); after != before {
				t.Fatal("unsafe state wrote files")
			}
		})
	}
}

func snapshotTree(t *testing.T, root string) string {
	t.Helper()
	var out []string
	if err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			b, _ := os.ReadFile(path)
			out = append(out, strings.TrimPrefix(path, root)+":"+string(b))
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return strings.Join(out, "\n")
}
