package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	agentadapter "github.com/desatatufuria/mcp-doc-manager/internal/adapters/agent"
)

func TestAgentServiceReadOnlySelectionsPreserveIndependentDimensions(t *testing.T) {
	service := AgentService{Adapters: fakeAgentRegistry(map[string]*fakeAgent{
		"opencode": {status: agentadapter.Status{Installed: true, Supported: true, Configured: true, Healthy: true}},
		"codex":    {status: agentadapter.Status{Installed: true, Supported: true}},
		"claude":   {status: agentadapter.Status{Supported: true, Configured: true}},
		"copilot":  {status: agentadapter.Status{Installed: true, Supported: false}},
		"pi":       {status: agentadapter.Status{Installed: true, Supported: true, Configured: true}},
	})}

	for _, tc := range []struct {
		name, selection string
		want            []string
	}{
		{name: "all", selection: "all", want: []string{"claude", "codex", "copilot", "opencode", "pi"}},
		{name: "detected", selection: "detected", want: []string{"codex", "copilot", "opencode", "pi"}},
		{name: "explicit", selection: "claude", want: []string{"claude"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := service.Execute(context.Background(), Request{Operation: OperationAgentStatus, Agent: tc.selection})
			if result.Outcome != OutcomeStatus || result.Error != nil || result.Status == nil {
				t.Fatalf("result = %#v", result)
			}
			got := make([]string, len(result.Status.Agents))
			for i, status := range result.Status.Agents {
				got[i] = status.Agent
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("agents = %v, want %v", got, tc.want)
			}
		})
	}

	result := service.Execute(context.Background(), Request{Operation: OperationAgentDoctor, Agent: "opencode"})
	status := result.Status.Agent
	if result.Outcome != OutcomeStatus || status == nil || !status.Installed || !status.Supported || !status.Configured || !status.Healthy {
		t.Fatalf("doctor result = %#v", result)
	}
}

func TestAgentServicePlansBeforeWritesAndRollsBackPartialConfigure(t *testing.T) {
	claude := &fakeAgent{status: agentadapter.Status{Supported: true}, unconfigureErr: errors.New("rollback failed")}
	codex := &fakeAgent{status: agentadapter.Status{Supported: true}, configureErr: errors.New("later failure")}
	service := AgentService{Adapters: fakeAgentRegistry(map[string]*fakeAgent{
		"opencode": {status: agentadapter.Status{Supported: true}}, "codex": codex, "claude": claude, "copilot": {status: agentadapter.Status{Supported: true}}, "pi": {status: agentadapter.Status{Supported: true}},
	})}

	planned := service.Execute(context.Background(), Request{Operation: OperationAgentConfigure, Agent: "all", Binary: "/tmp/docmanager", DryRun: true})
	if planned.Outcome != OutcomePlanned || len(planned.Changes) != 5 || claude.configures != 0 || codex.configures != 0 {
		t.Fatalf("dry-run result = %#v, writes = %d/%d", planned, claude.configures, codex.configures)
	}

	result := service.Execute(context.Background(), Request{Operation: OperationAgentConfigure, Agent: "all", Binary: "/tmp/docmanager"})
	if result.Outcome != OutcomeFailure || result.Error == nil || result.Error.Code != "agent_operation" || !strings.Contains(result.Error.Detail, "later failure; rollback failed") || claude.configures != 1 || claude.unconfigures != 1 {
		t.Fatalf("rollback result = %#v, error = %#v, claude = %#v", result, result.Error, claude)
	}
}

func TestAgentServicePreflightAndStableErrors(t *testing.T) {
	piRoot := t.TempDir()
	piConfig := filepath.Join(piRoot, ".pi", "agent", "mcp.json")
	if err := os.MkdirAll(filepath.Dir(piConfig), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(piConfig, []byte(`{"mcpServers":{"other":{"command":"other"}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	piBefore, err := os.ReadFile(piConfig)
	if err != nil {
		t.Fatal(err)
	}
	missingPi := agentadapter.NewPi(agentadapter.PiOptions{Home: piRoot, Binary: "/tmp/docmanager", Installed: func() bool { return true }, AdapterAvailable: func() bool { return false }})
	opencode := &fakeAgent{status: agentadapter.Status{Supported: true}}
	ports := fakeAgentRegistry(map[string]*fakeAgent{
		"opencode": opencode, "codex": {status: agentadapter.Status{Supported: true}}, "claude": {status: agentadapter.Status{Supported: true}}, "copilot": {status: agentadapter.Status{Supported: true}}, "pi": {status: agentadapter.Status{Supported: true}},
	})
	ports["pi"] = missingPi
	service := AgentService{Adapters: ports}

	for _, tc := range []struct {
		name, agent, want string
		statusErr         error
	}{
		{name: "unknown", agent: "unknown", want: "unsupported_agent"},
		{name: "pi prerequisite", agent: "pi", want: "unsupported_agent"},
		{name: "drift", agent: "opencode", want: "drift", statusErr: agentadapter.ErrDrift},
	} {
		t.Run(tc.name, func(t *testing.T) {
			opencode.statusErr = tc.statusErr
			result := service.Execute(context.Background(), Request{Operation: OperationAgentConfigure, Agent: tc.agent, Binary: "/tmp/docmanager"})
			if result.Outcome != OutcomeFailure || result.Error == nil || result.Error.Code != tc.want || opencode.configures != 0 {
				t.Fatalf("result = %#v", result)
			}
		})
	}
	piAfter, err := os.ReadFile(piConfig)
	if err != nil || string(piAfter) != string(piBefore) {
		t.Fatalf("Pi prerequisite changed config: %q, %v", piAfter, err)
	}
}

func TestAgentServiceUsesRealAdapterAtTemporaryRoot(t *testing.T) {
	root := t.TempDir()
	config := filepath.Join(root, "opencode.json")
	if err := os.WriteFile(config, []byte(`{"mcp":{}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	service := AgentService{Adapters: map[string]AgentPort{
		"opencode": agentadapter.NewOpenCode(agentadapter.OpenCodeOptions{Root: root, Binary: "/tmp/docmanager", Installed: func() bool { return true }}),
	}}
	for _, operation := range []Operation{OperationAgentConfigure, OperationAgentStatus, OperationAgentUnconfigure} {
		request := Request{Operation: operation, Agent: "opencode"}
		if operation == OperationAgentConfigure {
			request.Binary = "/tmp/docmanager"
		}
		result := service.Execute(context.Background(), request)
		if result.Outcome == OutcomeFailure {
			t.Fatalf("%s result = %#v", operation, result)
		}
	}
}

func TestAgentServiceMultiAgentHarness(t *testing.T) {
	root := t.TempDir()
	fakes := map[string]*fakeAgent{
		"opencode": {status: agentadapter.Status{Installed: true, Supported: true}},
		"codex":    {status: agentadapter.Status{Installed: true, Supported: true}},
		"claude":   {status: agentadapter.Status{Installed: true, Supported: true}},
		"copilot":  {status: agentadapter.Status{Installed: true, Supported: true}},
		"pi":       {status: agentadapter.Status{Installed: true, Supported: true}},
	}
	service := AgentService{Adapters: fakeAgentRegistry(fakes)}
	requests := []Request{
		{Operation: OperationAgentConfigure, Agent: "all", Binary: filepath.Join(root, "docmanager"), DryRun: true},
		{Operation: OperationAgentConfigure, Agent: "all", Binary: filepath.Join(root, "docmanager")},
		{Operation: OperationAgentStatus, Agent: "all"},
		{Operation: OperationAgentUnconfigure, Agent: "all"},
	}
	for _, request := range requests {
		result := service.Execute(context.Background(), request)
		if result.Outcome == OutcomeFailure || len(result.Status.Agents) != 5 {
			t.Fatalf("%s result = %#v", request.Operation, result)
		}
	}
	for name, fake := range fakes {
		if fake.configures != 1 || fake.unconfigures != 1 {
			t.Fatalf("%s calls = %#v", name, fake)
		}
	}
}

type fakeAgent struct {
	status                   agentadapter.Status
	statusErr, configureErr  error
	unconfigureErr           error
	configures, unconfigures int
}

func (a *fakeAgent) Status(context.Context) (agentadapter.Status, error) {
	return a.status, a.statusErr
}
func (a *fakeAgent) Configure(context.Context) error {
	a.configures++
	return a.configureErr
}
func (a *fakeAgent) Unconfigure(context.Context) error {
	a.unconfigures++
	return a.unconfigureErr
}

func fakeAgentRegistry(fakes map[string]*fakeAgent) map[string]AgentPort {
	ports := make(map[string]AgentPort, len(fakes))
	for name, fake := range fakes {
		ports[name] = fake
	}
	return ports
}
