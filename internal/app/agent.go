package app

import (
	"context"
	"errors"
	"fmt"
	"sort"

	agentadapter "github.com/desatatufuria/mcp-doc-manager/internal/adapters/agent"
)

var registeredAgentNames = []string{"claude", "codex", "copilot", "opencode", "pi"}

// AgentPort is deliberately small: parsing, routing, and mutation remain adapter concerns.
type AgentPort interface {
	Status(context.Context) (agentadapter.Status, error)
	Configure(context.Context) error
	Unconfigure(context.Context) error
}

// AgentService coordinates registered agent ports without taking ownership of their config formats.
type AgentService struct {
	Adapters map[string]AgentPort
}

func (s AgentService) Execute(ctx context.Context, request Request) Result {
	if stable := ValidateRequest(request); stable != nil {
		return failure(request, stable)
	}
	ports, stable := s.selectPorts(request.Agent)
	if stable != nil {
		return failure(request, stable)
	}
	statuses, stable := inspectPorts(ctx, ports)
	if stable != nil {
		return failure(request, stable)
	}
	if request.Agent == "detected" {
		ports, statuses = installedPorts(ports, statuses)
	}
	if isInspection(request.Operation) {
		return agentResult(request, OutcomeStatus, statuses, nil, nil)
	}
	if stable := validateMutation(statuses); stable != nil {
		return failure(request, stable)
	}
	changes := plannedAgentChanges(request.Operation, ports)
	if request.DryRun {
		return agentResult(request, OutcomePlanned, statuses, changes, nil)
	}
	configure := request.Operation == OperationAgentConfigure
	applied := make([]namedAgentPort, 0, len(ports))
	for _, port := range ports {
		var err error
		if configure {
			err = port.port.Configure(ctx)
		} else {
			err = port.port.Unconfigure(ctx)
		}
		if err != nil {
			stable := stableAgentError(err)
			if rollbackErr := rollbackPorts(ctx, applied, configure); rollbackErr != nil {
				stable.Detail = fmt.Sprintf("%s; rollback failed: %v", stable.Detail, rollbackErr)
			}
			return failure(request, stable)
		}
		applied = append(applied, port)
	}
	return agentResult(request, OutcomeSuccess, statuses, changes, nil)
}

type namedAgentPort struct {
	name string
	port AgentPort
}

func (s AgentService) selectPorts(selection string) ([]namedAgentPort, *StableError) {
	names := []string{selection}
	if selection == "all" || selection == "detected" {
		names = append([]string(nil), registeredAgentNames...)
	}
	ports := make([]namedAgentPort, 0, len(names))
	for _, name := range names {
		port, ok := s.Adapters[name]
		if !ok || port == nil {
			return nil, &StableError{Code: "unsupported_agent", Classification: "unsupported_agent", Detail: "agent is not registered: " + name}
		}
		ports = append(ports, namedAgentPort{name: name, port: port})
	}
	sort.Slice(ports, func(i, j int) bool { return ports[i].name < ports[j].name })
	return ports, nil
}

func inspectPorts(ctx context.Context, ports []namedAgentPort) ([]AgentStatus, *StableError) {
	statuses := make([]AgentStatus, 0, len(ports))
	for _, port := range ports {
		status, err := port.port.Status(ctx)
		if err != nil {
			return nil, stableAgentError(err)
		}
		statuses = append(statuses, AgentStatus{Agent: port.name, Installed: status.Installed, Supported: status.Supported, Configured: status.Configured, Healthy: status.Healthy})
	}
	return statuses, nil
}

func installedPorts(ports []namedAgentPort, statuses []AgentStatus) ([]namedAgentPort, []AgentStatus) {
	selectedPorts := make([]namedAgentPort, 0, len(ports))
	selectedStatuses := make([]AgentStatus, 0, len(statuses))
	for index, status := range statuses {
		if status.Installed {
			selectedPorts = append(selectedPorts, ports[index])
			selectedStatuses = append(selectedStatuses, status)
		}
	}
	return selectedPorts, selectedStatuses
}

func isInspection(operation Operation) bool {
	return operation == OperationAgentDetect || operation == OperationAgentStatus || operation == OperationAgentDoctor
}

func validateMutation(statuses []AgentStatus) *StableError {
	for _, status := range statuses {
		if !status.Supported {
			return &StableError{Code: "unsupported_agent", Classification: "unsupported_agent", Detail: "agent is unsupported: " + status.Agent}
		}
	}
	return nil
}

func plannedAgentChanges(operation Operation, ports []namedAgentPort) []Change {
	changes := make([]Change, 0, len(ports))
	for _, port := range ports {
		changes = append(changes, Change{Kind: string(operation), Target: port.name, After: "managed"})
	}
	return changes
}

func rollbackPorts(ctx context.Context, ports []namedAgentPort, configured bool) error {
	for index := len(ports) - 1; index >= 0; index-- {
		var err error
		if configured {
			err = ports[index].port.Unconfigure(ctx)
		} else {
			err = ports[index].port.Configure(ctx)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func agentResult(request Request, outcome string, statuses []AgentStatus, changes []Change, stable *StableError) Result {
	status := &Status{Agents: statuses}
	if len(statuses) == 1 {
		copy := statuses[0]
		status.Agent = &copy
	}
	return Result{Operation: request.Operation, Target: request.Agent, Outcome: outcome, Changes: changes, Status: status, Error: stable}
}

func stableAgentError(err error) *StableError {
	code := "agent_operation"
	switch {
	case errors.Is(err, agentadapter.ErrPrerequisiteMissing), errors.Is(err, agentadapter.ErrUnsupportedConfig):
		code = "unsupported_agent"
	case errors.Is(err, agentadapter.ErrDrift):
		code = "drift"
	case errors.Is(err, agentadapter.ErrMalformedConfig):
		code = "malformed_config"
	case errors.Is(err, agentadapter.ErrLocked):
		code = "locked"
	case errors.Is(err, agentadapter.ErrProbeFailed):
		code = "probe_failed"
	case errors.Is(err, agentadapter.ErrOwnership), errors.Is(err, agentadapter.ErrUnsafeRoute):
		code = "ownership"
	}
	return &StableError{Code: code, Classification: code, Detail: err.Error()}
}
