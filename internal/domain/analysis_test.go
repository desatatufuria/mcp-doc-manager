package domain

import "testing"

func TestAnalyzeClassifiesUntrustedDocumentationLikePathsWithoutExecution(t *testing.T) {
	tests := []struct {
		name    string
		paths   []string
		outcome Outcome
	}{
		{"requirements are not executed", []string{"requirements.txt"}, OutcomeCreate},
		{"cmake is not executed", []string{"CMakeLists.txt"}, OutcomeCreate},
		{"executable markdown is classified", []string{"docs/guide.mdx"}, OutcomeUpdate},
		{"readme shell is not executed", []string{"README.sh"}, OutcomeCreate},
		{"source has no impact", []string{"internal/service.go"}, OutcomeNoImpact},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report, err := Analyze(Evidence{ChangedPaths: tt.paths})
			if err != nil {
				t.Fatalf("Analyze() error = %v", err)
			}
			if report.Outcome != tt.outcome {
				t.Fatalf("outcome = %q, want %q", report.Outcome, tt.outcome)
			}
			if report.Confidence <= 0 || report.Confidence > 1 {
				t.Fatalf("confidence = %v, want (0,1]", report.Confidence)
			}
		})
	}
}

func TestAnalyzeReturnsExactlyThreeReadOnlyOutcomes(t *testing.T) {
	tests := []struct {
		name        string
		paths       []string
		wantOutcome Outcome
		wantPaths   int
	}{
		{"existing documentation", []string{"README.md"}, OutcomeUpdate, 1},
		{"documentation candidate", []string{"cmd/server.go"}, OutcomeCreate, 1},
		{"irrelevant source", []string{"internal/format.go"}, OutcomeNoImpact, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report, err := Analyze(Evidence{ChangedPaths: tt.paths})
			if err != nil {
				t.Fatal(err)
			}
			if report.Outcome != tt.wantOutcome || len(report.Candidates) != tt.wantPaths {
				t.Fatalf("report = %#v", report)
			}
			if report.Rationale == "" {
				t.Fatal("rationale must be present")
			}
		})
	}
}
