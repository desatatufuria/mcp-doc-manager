package domain

type Evidence struct {
	Root                 string
	Scope                Scope
	Identity             string
	ChangedPaths         []string
	DocumentationDigests map[string]string `json:"-"`
}
