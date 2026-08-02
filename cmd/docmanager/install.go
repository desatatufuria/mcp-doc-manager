package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	agentadapter "github.com/desatatufuria/mcp-doc-manager/internal/adapters/agent"
	"github.com/desatatufuria/mcp-doc-manager/internal/app"
)

type installAgent interface {
	Inspect(context.Context) (agentadapter.Status, error)
	Configure(context.Context) error
	Status(context.Context) (agentadapter.Status, error)
}

type installRuntime struct {
	in               io.Reader
	out              io.Writer
	getwd            func() (string, error)
	executable       func() (string, error)
	evalSymlinks     func(string) (string, error)
	lookPath         func(string) (string, error)
	workspaceDoctor  func(string) (app.WorkspaceStatus, error)
	workspaceInstall func(string, bool) (app.WorkspaceStatus, error)
	openCode         func(string) installAgent
}

type installResult struct {
	Operation  string           `json:"operation"`
	Outcome    string           `json:"outcome"`
	Repository string           `json:"repository,omitempty"`
	Agent      string           `json:"agent,omitempty"`
	Hook       string           `json:"hook,omitempty"`
	Workspace  bool             `json:"workspace_initialized"`
	Configured bool             `json:"opencode_configured"`
	Error      *app.StableError `json:"error,omitempty"`
}

func productionInstallRuntime(in io.Reader, out io.Writer) installRuntime {
	return installRuntime{
		in: in, out: out, getwd: os.Getwd, executable: os.Executable,
		evalSymlinks: filepath.EvalSymlinks, lookPath: exec.LookPath,
		workspaceDoctor: app.WorkspaceDoctor, workspaceInstall: app.WorkspaceInstall,
		openCode: func(binary string) installAgent {
			return agentadapter.NewOpenCode(agentadapter.OpenCodeOptions{
				Binary:    binary,
				Installed: func() bool { _, err := exec.LookPath("opencode"); return err == nil },
				Run: func(ctx context.Context, argv ...string) error {
					if len(argv) == 0 {
						return agentadapter.ErrProbeFailed
					}
					return exec.CommandContext(ctx, argv[0], argv[1:]...).Run()
				},
			})
		},
	}
}

func runInstall(args []string, runtime installRuntime) error {
	fs := flag.NewFlagSet("install", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	target := fs.String("target", "", "exact Git root")
	agent := fs.String("agent", "", "agent")
	enableHook := fs.Bool("enable-hook", false, "enable pre-push hook")
	dryRun := fs.Bool("dry-run", false, "plan without writes")
	asJSON := fs.Bool("json", false, "emit JSON")
	yes := fs.Bool("yes", false, "apply without prompting")
	if err := fs.Parse(args); err != nil || fs.NArg() != 0 {
		return installFailure(runtime.out, *asJSON, "", false, "invalid command flags")
	}
	if *agent != "" && *agent != "opencode" {
		return installFailure(runtime.out, *asJSON, "", false, "only opencode is supported")
	}
	if *asJSON && !*yes && !*dryRun {
		return installFailure(runtime.out, true, "", false, "--json requires --yes or --dry-run")
	}
	if *target == "" {
		var err error
		*target, err = runtime.getwd()
		if err != nil {
			return installFailure(runtime.out, *asJSON, "", false, err.Error())
		}
	}
	workspace, err := runtime.workspaceDoctor(*target)
	if err != nil {
		return installFailure(runtime.out, *asJSON, *target, false, err.Error())
	}
	*target = workspace.Root
	binary, err := runtime.executable()
	if err == nil {
		binary, err = filepath.Abs(binary)
	}
	if err == nil {
		binary, err = runtime.evalSymlinks(binary)
	}
	if err != nil || !filepath.IsAbs(binary) {
		return installFailure(runtime.out, *asJSON, *target, false, "cannot determine current docmanager executable")
	}
	if _, err := runtime.lookPath("opencode"); err != nil {
		return installFailureCode(runtime.out, *asJSON, *target, false, "unsupported_agent", "opencode is not detected")
	}
	opencode := runtime.openCode(binary)
	status, err := opencode.Inspect(context.Background())
	if err != nil {
		return installAdapterFailure(runtime.out, *asJSON, *target, err)
	}
	if !status.Installed || !status.Supported {
		return installFailureCode(runtime.out, *asJSON, *target, false, "unsupported_agent", "opencode is not detected or supported")
	}

	result := installResult{Operation: "install", Outcome: "planned", Repository: *target, Agent: "opencode", Hook: "disabled"}
	if *enableHook {
		result.Hook = "enabled"
	}
	if *dryRun {
		return renderInstall(runtime.out, *asJSON, result)
	}
	if !*yes {
		fmt.Fprintf(runtime.out, "Installation plan\n[x] Repository: %s\n[x] OpenCode: detected, configure MCP\n", *target)
		if *enableHook {
			fmt.Fprintln(runtime.out, "[x] Pre-push hook: enabled")
		} else {
			fmt.Fprintln(runtime.out, "[ ] Pre-push hook: disabled")
		}
		fmt.Fprint(runtime.out, "Continue? [y/N] ")
		scanner := bufio.NewScanner(runtime.in)
		if !scanner.Scan() || (strings.ToLower(strings.TrimSpace(scanner.Text())) != "y" && strings.ToLower(strings.TrimSpace(scanner.Text())) != "yes") {
			fmt.Fprintln(runtime.out, "Installation cancelled.")
			return nil
		}
	}

	installed, err := runtime.workspaceInstall(*target, *enableHook)
	if err != nil {
		return installFailure(runtime.out, *asJSON, *target, false, err.Error())
	}
	result.Workspace = true
	result.Hook = installed.Hook
	if err := opencode.Configure(context.Background()); err != nil {
		return partialInstallFailure(runtime.out, *asJSON, result, "configuration", err)
	}
	result.Configured = true
	status, err = opencode.Status(context.Background())
	if err != nil || !status.Configured || !status.Healthy {
		if err == nil {
			err = agentadapter.ErrProbeFailed
		}
		return partialInstallFailure(runtime.out, *asJSON, result, "verification", err)
	}
	result.Outcome = "success"
	if *asJSON {
		return json.NewEncoder(runtime.out).Encode(result)
	}
	fmt.Fprintf(runtime.out, "Repository initialized: %s\nOpenCode configured: MCP enabled\nPre-push hook: %s\nVerify with: opencode mcp list\n", *target, result.Hook)
	return nil
}

func renderInstall(out io.Writer, asJSON bool, result installResult) error {
	if asJSON {
		return json.NewEncoder(out).Encode(result)
	}
	fmt.Fprintf(out, "Installation plan\n[x] Repository: %s\n[x] OpenCode: detected, configure MCP\n", result.Repository)
	if result.Hook == "enabled" {
		fmt.Fprintln(out, "[x] Pre-push hook: enabled")
	} else {
		fmt.Fprintln(out, "[ ] Pre-push hook: disabled")
	}
	return nil
}

func installFailure(out io.Writer, asJSON bool, target string, partial bool, detail string) error {
	return installFailureCode(out, asJSON, target, partial, "invalid_input", detail)
}

func installFailureCode(out io.Writer, asJSON bool, target string, partial bool, code, detail string) error {
	stable := &app.StableError{Code: code, Classification: code, Detail: detail}
	if asJSON {
		_ = json.NewEncoder(out).Encode(installResult{Operation: "install", Outcome: "failure", Repository: target, Workspace: partial, Error: stable})
	}
	return stable
}

func installAdapterFailure(out io.Writer, asJSON bool, target string, err error) error {
	code := "agent_operation"
	switch {
	case errors.Is(err, agentadapter.ErrOwnership), errors.Is(err, agentadapter.ErrUnsafeRoute):
		code = "ownership"
	case errors.Is(err, agentadapter.ErrMalformedConfig):
		code = "malformed_config"
	case errors.Is(err, agentadapter.ErrUnsupportedConfig):
		code = "unsupported_agent"
	}
	return installFailureCode(out, asJSON, target, false, code, err.Error())
}

func partialInstallFailure(out io.Writer, asJSON bool, result installResult, phase string, err error) error {
	stable := &app.StableError{Code: "partial_install", Classification: "partial_install", Detail: "repository initialized; OpenCode " + phase + " failed: " + err.Error()}
	result.Outcome = "partial"
	result.Error = stable
	if asJSON {
		_ = json.NewEncoder(out).Encode(result)
	} else {
		fmt.Fprintf(out, "Repository initialized: %s\nOpenCode %s failed: %v\n", result.Repository, phase, err)
	}
	return stable
}
