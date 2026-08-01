package app

import (
	"context"
	"path/filepath"
	"testing"

	releaseadapter "github.com/desatatufuria/mcp-doc-manager/internal/adapters/release"
)

func TestReleaseServiceReturnsClosedResults(t *testing.T) {
	dir := t.TempDir()
	service := ReleaseService{Lifecycle: releaseadapter.Lifecycle{StateDir: filepath.Join(dir, "state")}}
	result := service.Execute(context.Background(), Request{Operation: OperationReleaseStatus, InstallDir: dir})
	if result.Outcome != OutcomeStatus || result.Status == nil || result.Status.Release == nil || result.Status.Release.Installed || result.Error != nil {
		t.Fatalf("status result = %#v", result)
	}
	result = service.Execute(context.Background(), Request{Operation: OperationReleaseStatus, InstallDir: dir, DryRun: true})
	if result.Outcome != OutcomeFailure || result.Error == nil || result.Error.Code != "invalid_input" {
		t.Fatalf("read-only mutation result = %#v", result)
	}
}
