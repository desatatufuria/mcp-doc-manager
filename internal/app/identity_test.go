package app

import "testing"

func TestValidateIdentity(t *testing.T) {
	tests := []struct {
		name  string
		input string
		code  string
	}{
		{name: "canonical", input: CanonicalModule},
		{name: "legacy", input: LegacyModule, code: "legacy_identity"},
		{name: "unknown", input: "example.com/other", code: "invalid_input"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateIdentity(tt.input)
			if tt.code == "" {
				if err != nil {
					t.Fatalf("ValidateIdentity(%q) error = %v", tt.input, err)
				}
				return
			}
			if err == nil || err.Code != tt.code {
				t.Fatalf("ValidateIdentity(%q) error = %#v, want code %q", tt.input, err, tt.code)
			}
		})
	}
}
