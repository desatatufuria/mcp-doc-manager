package app

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/gentleman-programming/repository-documentation-manager/internal/domain"
)

type EvidenceResolver interface {
	Resolve(context.Context, string, domain.Scope) (domain.Evidence, error)
}

type ReceiptLedger interface {
	Save(context.Context, domain.Report) error
	Verify(context.Context, domain.Receipt) error
}

type DocumentChangeRequest struct {
	Repository string
	Scope      domain.Scope
}

type DocumentChangeResult struct {
	Report *domain.Report
	Err    error
}

type DocumentChange struct {
	Resolver EvidenceResolver
	Ledger   ReceiptLedger
}

func (u DocumentChange) Execute(ctx context.Context, request DocumentChangeRequest) DocumentChangeResult {
	if err := request.Scope.Validate(); err != nil {
		return DocumentChangeResult{Err: err}
	}
	if u.Resolver == nil {
		return DocumentChangeResult{Err: domain.ErrUnsupportedRequest}
	}
	evidence, err := u.Resolver.Resolve(ctx, request.Repository, request.Scope)
	if err != nil {
		return DocumentChangeResult{Err: err}
	}
	report, err := domain.Analyze(evidence)
	if err != nil {
		return DocumentChangeResult{Err: err}
	}
	blobs, err := receiptBlobs(report)
	if err != nil {
		return DocumentChangeResult{Err: err}
	}
	receipt, err := domain.NewReceipt(report, blobs, receiptVersions)
	if err != nil {
		return DocumentChangeResult{Err: err}
	}
	report.Receipt = receipt
	if u.Ledger != nil {
		if err := u.Ledger.Save(ctx, report); err != nil {
			return DocumentChangeResult{Err: domain.ErrLedgerFailure}
		}
	}
	return DocumentChangeResult{Report: &report}
}

var receiptVersions = domain.ReceiptVersions{Schema: "receipt/v1", Tool: "docmanager/1"}

func receiptBlobs(report domain.Report) ([]domain.DocumentationBlob, error) {
	paths := report.Candidates
	if len(paths) == 0 {
		paths = report.Evidence.ChangedPaths
	}
	blobs := make([]domain.DocumentationBlob, 0, len(paths))
	for _, path := range paths {
		if filepath.IsAbs(path) || strings.Contains(path, "\x00") || path == ".." || strings.HasPrefix(path, ".."+string(filepath.Separator)) {
			return nil, domain.ErrContentRead
		}
		digest, ok := report.Evidence.DocumentationDigests[path]
		if !ok {
			return nil, domain.ErrContentRead
		}
		blobs = append(blobs, domain.DocumentationBlob{Path: path, Digest: digest})
	}
	return blobs, nil
}
