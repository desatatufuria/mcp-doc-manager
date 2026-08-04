package assets

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
)

//go:embed AGENTS.md
var agents string

//go:embed skills/docmanager/SKILL.md
var skill string

//go:embed .githooks/pre-push
var hook string

//go:embed opencode.md
var opencode string

const config = "{\"hook_mode\":\"warn\",\"schema\":\"docmanager/config/v1\"}\n"

type Guidance struct {
	Path, Version, Digest, Content string
}

func guidance(path, version, content string) Guidance {
	digest := sha256.Sum256([]byte(content))
	return Guidance{Path: path, Version: version, Digest: hex.EncodeToString(digest[:]), Content: content}
}

var OpenCodeGuidance = guidance("guidance/opencode.md", "v1", opencode)

// OpenCodeGuidanceSnapshots are catalogued predecessor bytes eligible for replacement.
var OpenCodeGuidanceSnapshots = []Guidance{
	guidance("guidance/opencode.md", "v0", "# DocManager OpenCode Guidance\n\nVersion: v0\n\nUse explicit documentation-impact review only.\n"),
}

var Files = map[string]string{
	"config.json":                         config,
	"guidance/AGENTS.md":                  agents,
	OpenCodeGuidance.Path:                 OpenCodeGuidance.Content,
	"guidance/skills/docmanager/SKILL.md": skill,
	"hooks/pre-push.json":                 hook,
}
