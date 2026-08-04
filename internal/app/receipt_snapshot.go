package app

import "github.com/desatatufuria/mcp-doc-manager/internal/domain"

// ReceiptSnapshot records the explicit review boundary for one analysis.
type ReceiptSnapshot struct {
	Scope          domain.Scope
	ConsideredDocs map[string]string
	Reviewed       bool
	Consumed       bool
}

// ReceiptEligibility returns a stable refusal reason without invoking verification.
func ReceiptEligibility(snapshot ReceiptSnapshot, scope domain.Scope, docs map[string]string) string {
	if !snapshot.Reviewed {
		return "review_required"
	}
	if snapshot.Consumed {
		return "already_verified"
	}
	if snapshot.Scope != scope {
		return "scope_changed"
	}
	for path, digest := range snapshot.ConsideredDocs {
		if docs[path] != digest {
			return "considered_bytes_changed"
		}
	}
	return "eligible"
}

// ConsumeReceipt records the single successful verification in the caller-owned snapshot.
func ConsumeReceipt(snapshot *ReceiptSnapshot, scope domain.Scope, docs map[string]string) string {
	if snapshot == nil {
		return "review_required"
	}
	if eligibility := ReceiptEligibility(*snapshot, scope, docs); eligibility != "eligible" {
		return eligibility
	}
	snapshot.Consumed = true
	return "eligible"
}
