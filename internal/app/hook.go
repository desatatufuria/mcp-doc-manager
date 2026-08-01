package app

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"os/exec"
	"strings"
	"time"

	"github.com/desatatufuria/mcp-doc-manager/internal/domain"
)

type HookResult struct {
	Status string `json:"status"`
}

type ReceiptCatalog interface {
	Receipts(context.Context) ([]domain.Receipt, error)
}

type PrePushVerifier struct {
	Resolver EvidenceResolver
	Ledger   ReceiptLedger
	Receipts ReceiptCatalog
}

// Validate reads standard pre-push update records and verifies stored receipts without a shell.
func (v PrePushVerifier) Validate(ctx context.Context, input io.Reader, mode, repository string) (HookResult, error) {
	if mode != "warn" && mode != "fail" {
		return HookResult{}, domain.ErrUnsupportedRequest
	}
	if v.Resolver == nil || v.Ledger == nil || v.Receipts == nil || repository == "" {
		return HookResult{}, domain.ErrUnsupportedRequest
	}
	scanner := bufio.NewScanner(input)
	scanner.Buffer(make([]byte, 256), 4096)
	lines := 0
	for scanner.Scan() {
		scope, err := prePushScope(ctx, repository, scanner.Text())
		if err != nil {
			return hookAmbiguous(mode)
		}
		lines++
		if lines > 32 {
			return hookAmbiguous(mode)
		}
		if scope != nil {
			if err := v.verifyScope(ctx, repository, *scope); err != nil {
				return HookResult{}, err
			}
		}
	}
	if scanner.Err() != nil || lines == 0 {
		return hookAmbiguous(mode)
	}
	return HookResult{Status: "valid"}, nil
}

func (v PrePushVerifier) verifyScope(ctx context.Context, repository string, scope domain.Scope) error {
	evidence, err := v.Resolver.Resolve(ctx, repository, scope)
	if err != nil {
		return err
	}
	scope = evidence.Scope
	receipts, err := v.Receipts.Receipts(ctx)
	if err != nil {
		return domain.ErrLedgerFailure
	}
	for _, receipt := range receipts {
		if receipt.Scope != scope {
			continue
		}
		if err := (VerifyReceipt{Resolver: v.Resolver, Ledger: v.Ledger}).Execute(ctx, DocumentChangeRequest{Repository: repository, Scope: scope}, receipt); err == nil {
			return nil
		}
	}
	return domain.ErrReceiptMismatch
}

func prePushScope(ctx context.Context, repository, line string) (*domain.Scope, error) {
	fields := strings.Fields(line)
	if len(fields) != 4 {
		return nil, domain.ErrUnsupportedRequest
	}
	objectIDLength, err := hookObjectIDLength(ctx, repository)
	if err != nil || !validObjectID(fields[1], objectIDLength) || !validObjectID(fields[3], objectIDLength) || !validRef(ctx, repository, fields[2]) {
		return nil, domain.ErrUnsupportedRequest
	}
	if zeroObjectID(fields[1]) {
		if fields[0] != "(delete)" || zeroObjectID(fields[3]) {
			return nil, domain.ErrUnsupportedRequest
		}
		return nil, nil
	}
	if !validRef(ctx, repository, fields[0]) {
		return nil, domain.ErrUnsupportedRequest
	}
	if zeroObjectID(fields[3]) {
		return &domain.Scope{Kind: domain.ScopeInitial, Range: fields[1]}, nil
	}
	return &domain.Scope{Kind: domain.ScopeRange, Range: fields[3] + ".." + fields[1]}, nil
}

func validRef(ctx context.Context, repository, ref string) bool {
	if !strings.HasPrefix(ref, "refs/") || strings.ContainsAny(ref, " \t\n\x00") {
		return false
	}
	_, err := hookGitCommand(ctx, repository, "check-ref-format", ref)
	return err == nil
}

func validObjectID(value string, length int) bool {
	if len(value) != length {
		return false
	}
	for _, r := range value {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f')) {
			return false
		}
	}
	return true
}

func zeroObjectID(value string) bool {
	return strings.Trim(value, "0") == ""
}

func hookObjectIDLength(ctx context.Context, repository string) (int, error) {
	value, err := hookGitCommand(ctx, repository, "hash-object", "--stdin")
	if err != nil {
		return 0, err
	}
	return len(strings.TrimSpace(value)), nil
}

var errHookOutputLimit = errors.New("hook git output limit")

func hookGitCommand(parent context.Context, repository string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	defer cancel()
	output := &hookOutput{limit: 4096}
	cmd := exec.CommandContext(ctx, "git", append([]string{"-c", "core.attributesfile=/dev/null", "-C", repository}, args...)...)
	cmd.Env = []string{"GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null", "GIT_TERMINAL_PROMPT=0", "LC_ALL=C", "TZ=UTC"}
	cmd.Stdin, cmd.Stdout, cmd.Stderr = strings.NewReader(""), output, output
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return output.String(), nil
}

type hookOutput struct {
	bytes.Buffer
	limit int
}

func (b *hookOutput) Write(p []byte) (int, error) {
	if b.Len()+len(p) > b.limit {
		return 0, errHookOutputLimit
	}
	return b.Buffer.Write(p)
}

func hookAmbiguous(mode string) (HookResult, error) {
	if mode == "fail" {
		return HookResult{}, domain.ErrUnsupportedRequest
	}
	return HookResult{Status: "warning"}, nil
}
