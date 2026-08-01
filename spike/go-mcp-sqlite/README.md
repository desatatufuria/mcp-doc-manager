# Go MCP and SQLite Compatibility Spike

This spike proves that the selected Go MCP SDK and pure-Go SQLite driver work together before product code is created.

## Pass criteria

The spike passes only when all checks succeed:

1. `go test ./...` starts the MCP server over stdio, invokes `sqlite_round_trip`, and reads the persisted SQLite row back from a separate connection.
2. `CGO_ENABLED=0 go build -o /tmp/go-mcp-sqlite-spike .` produces a native binary and `/tmp/go-mcp-sqlite-spike --package-smoke` succeeds.
3. `CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o /tmp/go-mcp-sqlite-darwin-arm64 .`, `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /tmp/go-mcp-sqlite-linux-amd64 .`, and `CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o /tmp/go-mcp-sqlite-windows-amd64.exe .` compile without CGo.
4. The GitHub Actions matrix runs the tests and native package smoke on macOS, Linux, and Windows.

Any failed criterion is a technology failure for the proposed Go stack. Record the failure, reassess Rust, and do not create the root module or product `cmd/` and `internal/` packages.

## Local verification

```bash
go test ./...
CGO_ENABLED=0 go build -o /tmp/go-mcp-sqlite-spike .
/tmp/go-mcp-sqlite-spike --package-smoke
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o /tmp/go-mcp-sqlite-darwin-arm64 .
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /tmp/go-mcp-sqlite-linux-amd64 .
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o /tmp/go-mcp-sqlite-windows-amd64.exe .
```
