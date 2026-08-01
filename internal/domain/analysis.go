package domain

import (
	"path"
	"sort"
	"strings"
)

type Outcome string

const (
	OutcomeUpdate   Outcome = "update"
	OutcomeCreate   Outcome = "create"
	OutcomeNoImpact Outcome = "no-impact"
)

type Report struct {
	Outcome    Outcome
	Candidates []string
	Rationale  string
	Confidence float64
	Evidence   Evidence
	Receipt    Receipt
}

func Analyze(evidence Evidence) (Report, error) {
	if len(evidence.ChangedPaths) == 0 {
		return Report{}, ErrEmptyScope
	}
	paths := append([]string(nil), evidence.ChangedPaths...)
	sort.Strings(paths)
	for _, changed := range paths {
		if isDocumentation(changed) {
			return Report{Outcome: OutcomeUpdate, Candidates: []string{changed}, Rationale: "A tracked documentation path changed.", Confidence: 0.95, Evidence: evidence}, nil
		}
	}
	for _, changed := range paths {
		if isDocumentationCandidate(changed) {
			return Report{Outcome: OutcomeCreate, Candidates: []string{"README.md"}, Rationale: "A repository-facing path changed without an existing documentation update.", Confidence: 0.70, Evidence: evidence}, nil
		}
	}
	return Report{Outcome: OutcomeNoImpact, Rationale: "The selected paths have no documentation impact.", Confidence: 0.85, Evidence: evidence}, nil
}

func isDocumentation(changed string) bool {
	name := strings.ToLower(path.Base(changed))
	if name == "readme.sh" {
		return false
	}
	ext := strings.ToLower(path.Ext(changed))
	return name == "readme.md" || ext == ".md" || ext == ".mdx" || strings.HasPrefix(strings.ToLower(changed), "docs/")
}

func isDocumentationCandidate(changed string) bool {
	if strings.Contains(changed, "\x00") || strings.HasPrefix(changed, "/") || strings.HasPrefix(changed, "../") {
		return false
	}
	name := path.Base(changed)
	return name != "" && !strings.HasSuffix(name, ".go") || strings.HasPrefix(changed, "cmd/") || name == "requirements.txt" || name == "CMakeLists.txt" || name == "README.sh"
}
