package agent

import (
	"bytes"
	"encoding/json"
	"strings"
)

const managedGuidanceBegin = "DOCMANAGER-MANAGED-GUIDANCE-v1"

func decodeConfig(data []byte) (map[string]any, error) {
	plain := stripJSONC(string(data))
	var config map[string]any
	if err := json.Unmarshal([]byte(plain), &config); err != nil {
		return nil, ErrMalformedConfig
	}
	mcp, ok := config["mcp"]
	if mcp == nil {
		config["mcp"] = map[string]any{}
	} else if !ok || !isObject(mcp) {
		return nil, ErrUnsupportedConfig
	}
	return config, nil
}

func isObject(value any) bool { _, ok := value.(map[string]any); return ok }

func stripJSONC(in string) string {
	var out strings.Builder
	inString, escaped := false, false
	for i := 0; i < len(in); i++ {
		if inString {
			out.WriteByte(in[i])
			if in[i] == '"' && !escaped {
				inString = false
			}
			escaped = in[i] == '\\' && !escaped
			continue
		}
		if in[i] == '"' {
			inString = true
			out.WriteByte(in[i])
			continue
		}
		if i+1 < len(in) && in[i] == '/' && in[i+1] == '/' {
			for i < len(in) && in[i] != '\n' {
				i++
			}
			if i < len(in) {
				out.WriteByte('\n')
			}
			continue
		}
		if i+1 < len(in) && in[i] == '/' && in[i+1] == '*' {
			i += 2
			for i+1 < len(in) && !(in[i] == '*' && in[i+1] == '/') {
				i++
			}
			i++
			continue
		}
		out.WriteByte(in[i])
	}
	return out.String()
}

func encodeConfig(config map[string]any, jsonc bool, comments string) ([]byte, error) {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return nil, err
	}
	if jsonc && comments != "" {
		return append([]byte(comments), append(data, '\n')...), nil
	}
	return append(data, '\n'), nil
}

func leadingComments(data []byte) string {
	var lines []string
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "//") {
			lines = append(lines, line)
		} else if strings.TrimSpace(line) != "" {
			break
		}
	}
	if len(lines) == 0 {
		return ""
	}
	return strings.Join(lines, "\n") + "\n"
}

func sameBytes(a, b []byte) bool { return bytes.Equal(a, b) }
