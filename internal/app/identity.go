package app

const (
	CanonicalModule = "github.com/desatatufuria/mcp-doc-manager"
	LegacyModule    = "github.com/gentleman-programming/repository-documentation-manager"
)

// ValidateIdentity rejects legacy and unknown external module references.
func ValidateIdentity(identity string) *StableError {
	switch identity {
	case CanonicalModule:
		return nil
	case LegacyModule:
		return &StableError{Code: "legacy_identity", Classification: "legacy_identity", Detail: "the previous module identity is not canonical"}
	default:
		return &StableError{Code: "invalid_input", Classification: "invalid_input", Detail: "unknown module identity"}
	}
}
