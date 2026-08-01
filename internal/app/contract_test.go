package app

import (
	"context"
	"testing"
)

func TestValidateRequestRejectsClosedOperationInputsBeforeExecution(t *testing.T) {
	tests := []struct {
		name    string
		request Request
	}{
		{name: "release install missing manifest", request: Request{Operation: OperationReleaseInstall, InstallDir: "/bin"}},
		{name: "agent status dry run forbidden", request: Request{Operation: OperationAgentStatus, Agent: "opencode", DryRun: true}},
		{name: "workspace status hook forbidden", request: Request{Operation: OperationWorkspaceStatus, Target: "/repo", EnableHook: true}},
		{name: "unknown operation", request: Request{Operation: "unknown"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Execute(context.Background(), tt.request)
			if result.Outcome != OutcomeFailure || result.Error == nil || result.Error.Code != "invalid_input" {
				t.Fatalf("Execute(%#v) = %#v, want invalid_input failure", tt.request, result)
			}
			if len(result.Changes) != 0 {
				t.Fatalf("invalid request changes = %#v, want no writes", result.Changes)
			}
		})
	}
}

func TestExecuteReturnsHeadlessJSONUnionAndDryRunPlanWithoutChanges(t *testing.T) {
	result := Execute(context.Background(), Request{
		Operation:  OperationAgentConfigure,
		Agent:      "opencode",
		Binary:     "/usr/local/bin/docmanager",
		ConfigRoot: t.TempDir(),
		DryRun:     true,
	})

	if result.Outcome != OutcomePlanned || result.Error != nil || len(result.Changes) != 0 {
		t.Fatalf("dry-run result = %#v, want planned union without changes", result)
	}
	if result.Operation != OperationAgentConfigure || result.Target != "opencode" {
		t.Fatalf("dry-run identity = %#v", result)
	}
}
