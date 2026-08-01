package assets

import _ "embed"

//go:embed AGENTS.md
var agents string

//go:embed skills/docmanager/SKILL.md
var skill string

//go:embed .githooks/pre-push
var hook string

const config = "{\"hook_mode\":\"warn\",\"schema\":\"docmanager/config/v1\"}\n"

var Files = map[string]string{
	"config.json":                         config,
	"guidance/AGENTS.md":                  agents,
	"guidance/skills/docmanager/SKILL.md": skill,
	"hooks/pre-push.json":                 hook,
}
