package agent

import (
	"bytes"
	"encoding/json"
	"strings"
)

const managedGuidanceBegin = "DOCMANAGER-MANAGED-GUIDANCE-v1"

func decodeConfig(data []byte) (map[string]any, error) {
	var config map[string]any
	if err := json.Unmarshal(data, &config); err != nil {
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
	text := string(data)
	index := 0
	for index < len(text) {
		for index < len(text) && strings.ContainsRune(" \t\r\n", rune(text[index])) {
			index++
		}
		switch {
		case strings.HasPrefix(text[index:], "//"):
			if newline := strings.IndexByte(text[index:], '\n'); newline >= 0 {
				index += newline + 1
			} else {
				index = len(text)
			}
		case strings.HasPrefix(text[index:], "/*"):
			end := strings.Index(text[index+2:], "*/")
			if end < 0 {
				return ""
			}
			index += end + 4
		default:
			return text[:index]
		}
	}
	return text
}

func sameBytes(a, b []byte) bool { return bytes.Equal(a, b) }
