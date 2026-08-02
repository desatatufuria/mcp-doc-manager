package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"

	agentadapter "github.com/desatatufuria/mcp-doc-manager/internal/adapters/agent"
	"github.com/desatatufuria/mcp-doc-manager/internal/app"
)

func TestInstallInteractivePlanDeclineConfirmAndEOF(t *testing.T) {
	for _, tc := range []struct {
		name, input string
		wantWrites  bool
	}{
		{name: "decline", input: "n\n"},
		{name: "EOF", input: ""},
		{name: "confirm", input: "y\n", wantWrites: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			agent := &fakeInstallAgent{status: agentadapter.Status{Installed: true, Supported: true}}
			runtime, order := fakeRuntime(strings.NewReader(tc.input), &out, agent)
			if err := runInstall([]string{"--target", "/repo", "--agent", "opencode"}, runtime); err != nil {
				t.Fatal(err)
			}
			plan := "Installation plan\n[x] Repository: /repo\n[x] OpenCode: detected, configure MCP\n[ ] Pre-push hook: disabled\nContinue? [y/N] "
			if !strings.HasPrefix(out.String(), plan) {
				t.Fatalf("output = %q", out.String())
			}
			if tc.wantWrites {
				if strings.Join(*order, ",") != "workspace,configure,probe" || !strings.Contains(out.String(), "Verify with: opencode mcp list") {
					t.Fatalf("order/output = %v, %q", *order, out.String())
				}
			} else if len(*order) != 0 {
				t.Fatalf("decline wrote: %v", *order)
			}
		})
	}
}

func TestInstallHeadlessRules(t *testing.T) {
	for _, tc := range []struct {
		name                string
		args                []string
		wantErr, wantWrites bool
	}{
		{name: "yes", args: []string{"--yes"}, wantWrites: true},
		{name: "dry run", args: []string{"--dry-run"}},
		{name: "JSON dry run", args: []string{"--json", "--dry-run"}},
		{name: "JSON yes", args: []string{"--json", "--yes"}, wantWrites: true},
		{name: "JSON requires headless choice", args: []string{"--json"}, wantErr: true},
		{name: "unsupported agent", args: []string{"--agent", "claude", "--yes"}, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			agent := &fakeInstallAgent{status: agentadapter.Status{Installed: true, Supported: true}}
			runtime, order := fakeRuntime(strings.NewReader("must not be read"), &out, agent)
			err := runInstall(tc.args, runtime)
			if (err != nil) != tc.wantErr {
				t.Fatalf("error = %v", err)
			}
			if (len(*order) != 0) != tc.wantWrites {
				t.Fatalf("writes = %v", *order)
			}
			if contains(tc.args, "--json") {
				var result installResult
				if json.Unmarshal(out.Bytes(), &result) != nil || result.Operation != "install" {
					t.Fatalf("JSON = %q", out.String())
				}
			}
		})
	}
}

func TestInstallDetectionAndPartialFailureDoNotCrossRollbackBoundary(t *testing.T) {
	var out bytes.Buffer
	agent := &fakeInstallAgent{status: agentadapter.Status{Installed: true, Supported: true}, configureErr: errors.New("config refused")}
	runtime, order := fakeRuntime(strings.NewReader(""), &out, agent)
	if err := runInstall([]string{"--yes"}, runtime); err == nil || strings.Join(*order, ",") != "workspace,configure" || !strings.Contains(out.String(), "Repository initialized") {
		t.Fatalf("partial = %v, %v, %q", err, *order, out.String())
	}

	out.Reset()
	runtime, order = fakeRuntime(strings.NewReader(""), &out, &fakeInstallAgent{status: agentadapter.Status{Installed: true, Supported: true}})
	runtime.lookPath = func(string) (string, error) { return "", errors.New("missing") }
	if err := runInstall([]string{"--yes"}, runtime); err == nil || len(*order) != 0 {
		t.Fatalf("missing OpenCode = %v, writes %v", err, *order)
	}
}

func TestInstallEnablesHookOnlyWhenExplicit(t *testing.T) {
	var out bytes.Buffer
	agent := &fakeInstallAgent{status: agentadapter.Status{Installed: true, Supported: true}}
	runtime, _ := fakeRuntime(strings.NewReader(""), &out, agent)
	requested := false
	install := runtime.workspaceInstall
	runtime.workspaceInstall = func(target string, hook bool) (app.WorkspaceStatus, error) {
		requested = hook
		return install(target, hook)
	}
	if err := runInstall([]string{"--yes", "--enable-hook"}, runtime); err != nil {
		t.Fatal(err)
	}
	if !requested || !strings.Contains(out.String(), "Pre-push hook: opted-in") {
		t.Fatalf("hook request/output = %v, %q", requested, out.String())
	}
}

type fakeInstallAgent struct {
	status       agentadapter.Status
	configureErr error
	order        *[]string
}

func (a *fakeInstallAgent) Inspect(context.Context) (agentadapter.Status, error) {
	return a.status, nil
}
func (a *fakeInstallAgent) Configure(context.Context) error {
	*a.order = append(*a.order, "configure")
	return a.configureErr
}
func (a *fakeInstallAgent) Status(context.Context) (agentadapter.Status, error) {
	*a.order = append(*a.order, "probe")
	status := a.status
	status.Configured, status.Healthy = true, true
	return status, nil
}

func fakeRuntime(in io.Reader, out io.Writer, agent *fakeInstallAgent) (installRuntime, *[]string) {
	order := []string{}
	agent.order = &order
	return installRuntime{
		in: in, out: out,
		getwd:        func() (string, error) { return "/repo", nil },
		executable:   func() (string, error) { return "/bin/docmanager", nil },
		evalSymlinks: func(path string) (string, error) { return path, nil },
		lookPath:     func(string) (string, error) { return "/bin/opencode", nil },
		workspaceDoctor: func(target string) (app.WorkspaceStatus, error) {
			return app.WorkspaceStatus{Root: target, State: "absent", Hook: "absent"}, nil
		},
		workspaceInstall: func(target string, hook bool) (app.WorkspaceStatus, error) {
			order = append(order, "workspace")
			state := "absent"
			if hook {
				state = "opted-in"
			}
			return app.WorkspaceStatus{Root: target, State: "installed", Hook: state}, nil
		},
		openCode: func(string) installAgent { return agent },
	}, &order
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
