package app

import "context"

type Operation string

const (
	OperationReleaseInstall     Operation = "release.install"
	OperationReleaseUpgrade     Operation = "release.upgrade"
	OperationReleaseRollback    Operation = "release.rollback"
	OperationReleaseStatus      Operation = "release.status"
	OperationReleaseDoctor      Operation = "release.doctor"
	OperationAgentConfigure     Operation = "agent.configure"
	OperationAgentUnconfigure   Operation = "agent.unconfigure"
	OperationAgentDetect        Operation = "agent.detect"
	OperationAgentStatus        Operation = "agent.status"
	OperationAgentDoctor        Operation = "agent.doctor"
	OperationWorkspaceInstall   Operation = "workspace.install"
	OperationWorkspaceUninstall Operation = "workspace.uninstall"
	OperationWorkspaceDoctor    Operation = "workspace.doctor"
	OperationWorkspaceStatus    Operation = "workspace.status"
)

const (
	OutcomeSuccess = "success"
	OutcomeStatus  = "status"
	OutcomePlanned = "planned"
	OutcomeFailure = "failure"
)

type Change struct {
	Kind   string `json:"kind"`
	Target string `json:"target"`
	Before string `json:"before,omitempty"`
	After  string `json:"after,omitempty"`
}

type AgentStatus struct {
	Agent      string `json:"agent"`
	Installed  bool   `json:"installed"`
	Supported  bool   `json:"supported"`
	Configured bool   `json:"configured"`
	Healthy    bool   `json:"healthy"`
	Detail     string `json:"detail,omitempty"`
}

type ReleaseStatus struct {
	Installed bool   `json:"installed"`
	Healthy   bool   `json:"healthy"`
	Version   string `json:"version,omitempty"`
	Platform  string `json:"platform,omitempty"`
}

type WorkspaceStatus struct {
	Root  string `json:"root"`
	State string `json:"state"`
	Hook  string `json:"hook"`
}

type Status struct {
	Agent     *AgentStatus     `json:"agent,omitempty"`
	Agents    []AgentStatus    `json:"agents,omitempty"`
	Release   *ReleaseStatus   `json:"release,omitempty"`
	Workspace *WorkspaceStatus `json:"workspace,omitempty"`
}

type StableError struct {
	Code           string `json:"code"`
	Classification string `json:"classification"`
	Detail         string `json:"detail,omitempty"`
}

func (e *StableError) Error() string {
	if e == nil {
		return ""
	}
	return e.Code
}

type Result struct {
	Operation Operation    `json:"operation"`
	Target    string       `json:"target,omitempty"`
	Outcome   string       `json:"outcome"`
	Changes   []Change     `json:"changes"`
	Status    *Status      `json:"status,omitempty"`
	Error     *StableError `json:"error,omitempty"`
}

type Request struct {
	Operation   Operation
	ManifestURL string
	InstallDir  string
	Version     string
	Agent       string
	Binary      string
	ConfigRoot  string
	Target      string
	EnableHook  bool
	DryRun      bool
}

// OperationPort is the future adapter boundary for operations that mutate external state.
type OperationPort interface {
	Execute(context.Context, Request) Result
}

func Execute(_ context.Context, request Request) Result {
	if stable := ValidateRequest(request); stable != nil {
		return failure(request, stable)
	}
	if request.DryRun {
		return Result{Operation: request.Operation, Target: targetFor(request), Outcome: OutcomePlanned, Changes: []Change{}}
	}
	switch request.Operation {
	case OperationReleaseStatus, OperationReleaseDoctor:
		return Result{Operation: request.Operation, Outcome: OutcomeStatus, Changes: []Change{}, Status: &Status{Release: &ReleaseStatus{Platform: "unconfigured"}}}
	case OperationAgentDetect, OperationAgentStatus, OperationAgentDoctor:
		return Result{Operation: request.Operation, Target: request.Agent, Outcome: OutcomeStatus, Changes: []Change{}, Status: &Status{Agent: &AgentStatus{Agent: request.Agent, Detail: "not configured"}}}
	case OperationWorkspaceStatus, OperationWorkspaceDoctor:
		return Result{Operation: request.Operation, Target: request.Target, Outcome: OutcomeStatus, Changes: []Change{}, Status: &Status{Workspace: &WorkspaceStatus{Root: request.Target, State: "absent", Hook: "absent"}}}
	default:
		return failure(request, &StableError{Code: "unsupported_operation", Classification: "unsupported_operation", Detail: "operation is not implemented in this release"})
	}
}

func ValidateRequest(request Request) *StableError {
	invalid := func(detail string) *StableError {
		return &StableError{Code: "invalid_input", Classification: "invalid_input", Detail: detail}
	}
	forbidden := func(values ...string) bool {
		for _, value := range values {
			if value != "" {
				return true
			}
		}
		return false
	}
	switch request.Operation {
	case OperationReleaseInstall, OperationReleaseUpgrade:
		if request.ManifestURL == "" || request.InstallDir == "" || forbidden(request.Agent, request.Binary, request.ConfigRoot, request.Target) || request.EnableHook {
			return invalid("release install requires manifest_url and install_dir only")
		}
	case OperationReleaseRollback:
		if request.InstallDir == "" || request.ManifestURL != "" || request.Version != "" || forbidden(request.Agent, request.Binary, request.ConfigRoot, request.Target) || request.EnableHook || request.DryRun {
			return invalid("release rollback requires install_dir only")
		}
	case OperationReleaseStatus, OperationReleaseDoctor:
		if request.ManifestURL != "" || request.Version != "" || forbidden(request.Agent, request.Binary, request.ConfigRoot, request.Target) || request.EnableHook || request.DryRun {
			return invalid("release inspection accepts install_dir only")
		}
	case OperationAgentConfigure:
		if request.Agent == "" || request.Binary == "" || forbidden(request.ManifestURL, request.InstallDir, request.Version, request.Target) || request.EnableHook {
			return invalid("agent configure requires agent and binary")
		}
	case OperationAgentUnconfigure:
		if request.Agent == "" || request.Binary != "" || forbidden(request.ManifestURL, request.InstallDir, request.Version, request.Target) || request.EnableHook {
			return invalid("agent unconfigure requires agent")
		}
	case OperationAgentDetect, OperationAgentStatus, OperationAgentDoctor:
		if request.Agent == "" || request.Binary != "" || forbidden(request.ManifestURL, request.InstallDir, request.Version, request.Target) || request.EnableHook || request.DryRun {
			return invalid("agent inspection requires agent")
		}
	case OperationWorkspaceInstall:
		if request.Target == "" || forbidden(request.ManifestURL, request.InstallDir, request.Version, request.Agent, request.Binary, request.ConfigRoot) {
			return invalid("workspace install requires target")
		}
	case OperationWorkspaceUninstall:
		if request.Target == "" || forbidden(request.ManifestURL, request.InstallDir, request.Version, request.Agent, request.Binary, request.ConfigRoot) || request.EnableHook {
			return invalid("workspace uninstall requires target")
		}
	case OperationWorkspaceDoctor, OperationWorkspaceStatus:
		if request.Target == "" || forbidden(request.ManifestURL, request.InstallDir, request.Version, request.Agent, request.Binary, request.ConfigRoot) || request.EnableHook || request.DryRun {
			return invalid("workspace inspection requires target")
		}
	default:
		return invalid("unknown operation")
	}
	return nil
}

func targetFor(request Request) string {
	if request.Target != "" {
		return request.Target
	}
	if request.Agent != "" {
		return request.Agent
	}
	return request.InstallDir
}

func failure(request Request, stable *StableError) Result {
	return Result{Operation: request.Operation, Target: targetFor(request), Outcome: OutcomeFailure, Changes: []Change{}, Error: stable}
}
