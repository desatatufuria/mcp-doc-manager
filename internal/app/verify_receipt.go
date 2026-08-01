package app

import (
	"context"
	"reflect"

	"github.com/desatatufuria/mcp-doc-manager/internal/domain"
)

type VerifyReceipt struct {
	Resolver EvidenceResolver
	Ledger   ReceiptLedger
}

func (u VerifyReceipt) Execute(ctx context.Context, request DocumentChangeRequest, receipt domain.Receipt) error {
	if u.Resolver == nil || u.Ledger == nil {
		return domain.ErrUnsupportedRequest
	}
	if err := receipt.ValidateDigest(); err != nil {
		return domain.ErrReceiptMismatch
	}
	if err := request.Scope.Validate(); err != nil {
		return err
	}
	evidence, err := u.Resolver.Resolve(ctx, request.Repository, request.Scope)
	if err != nil {
		return err
	}
	report, err := domain.Analyze(evidence)
	if err != nil {
		return err
	}
	blobs, err := receiptBlobs(report)
	if err != nil {
		return err
	}
	expected, err := domain.NewReceipt(report, blobs, receiptVersions)
	if err != nil || !reflect.DeepEqual(expected, receipt) {
		return domain.ErrReceiptMismatch
	}
	return u.Ledger.Verify(ctx, receipt)
}
