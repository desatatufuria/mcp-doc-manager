package agent

import (
	"encoding/json"
	"os"
)

func jsonAgentEntry(data []byte, key, binary string, commandString bool) (owned, valid bool, err error) {
	var document map[string]any
	if err := json.Unmarshal(data, &document); err != nil {
		return false, false, ErrMalformedConfig
	}
	servers, exists := document[key]
	if !exists {
		return false, false, nil
	}
	table, ok := servers.(map[string]any)
	if !ok {
		return false, false, ErrUnsupportedConfig
	}
	entry, exists := table["docmanager"]
	if !exists {
		return false, false, nil
	}
	owned, valid = ownedJSONAgentEntry(entry, binary, commandString)
	return owned, valid, nil
}

func jsonAgentHasEntry(data []byte, key string) bool {
	var document map[string]any
	if json.Unmarshal(data, &document) != nil {
		return false
	}
	table, ok := document[key].(map[string]any)
	if !ok {
		return false
	}
	_, exists := table["docmanager"]
	return exists
}

func ownedJSONAgentEntry(value any, binary string, commandString bool) (owned, valid bool) {
	entry, ok := value.(map[string]any)
	if !ok {
		return false, false
	}
	marker, markerOK := entry["_docmanager"].(string)
	if !markerOK {
		return false, false
	}
	if commandString {
		command, commandOK := entry["command"].(string)
		args, argsOK := entry["args"].([]any)
		return marker == "managed/v1" && commandOK && command == binary && argsOK && len(args) == 1 && args[0] == "mcp", true
	}
	command, commandOK := entry["command"].([]any)
	return marker == "managed/v1" && commandOK && len(command) == 2 && command[0] == binary && command[1] == "mcp", true
}

func mutateJSONAgent(config, guide, key, binary string, commandString, add bool) error {
	data, err := os.ReadFile(config)
	if os.IsNotExist(err) {
		data = []byte(`{}`)
	} else if err != nil {
		return err
	}
	var document map[string]any
	if err := json.Unmarshal(data, &document); err != nil {
		return ErrMalformedConfig
	}
	servers, exists := document[key]
	if !exists {
		servers = map[string]any{}
		document[key] = servers
	}
	table, ok := servers.(map[string]any)
	if !ok {
		return ErrUnsupportedConfig
	}
	owned, valid, err := jsonAgentEntry(data, key, binary, commandString)
	if err != nil {
		return err
	}
	if add {
		if _, exists := table["docmanager"]; exists && !valid {
			return ErrOwnership
		}
		if valid && !owned {
			return ErrDrift
		}
		if !valid {
			if commandString {
				table["docmanager"] = map[string]any{"command": binary, "args": []string{"mcp"}, "_docmanager": "managed/v1"}
			} else {
				table["docmanager"] = map[string]any{"command": []string{binary, "mcp"}, "_docmanager": "managed/v1"}
			}
		}
	} else {
		if !valid {
			if _, exists := table["docmanager"]; exists {
				return ErrDrift
			}
			return nil
		}
		if !owned {
			return ErrDrift
		}
		delete(table, "docmanager")
	}
	updated, _ := json.MarshalIndent(document, "", "  ")
	updated = append(updated, '\n')
	if current, readErr := os.ReadFile(config); readErr == nil && !sameBytes(current, data) {
		return ErrDrift
	}
	if err := atomicWrite(config, updated); err != nil {
		return err
	}
	return mutateGuidance(guide, add)
}
