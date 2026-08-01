# Testing Conventions

Run focused package tests before the repository suite:

```bash
go test ./path/to/package
go test ./...
go vet ./...
```

Use table-driven tests for multiple scenarios, `t.TempDir()` for filesystem state, and skippable integration tests for external commands. Keep each test beside the behavior it verifies.
