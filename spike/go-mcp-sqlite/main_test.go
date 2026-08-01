package main

import (
	"context"
	"database/sql"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	_ "modernc.org/sqlite"
)

func TestMCPStdioSQLiteRoundTrip(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping MCP stdio integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	databasePath := filepath.Join(t.TempDir(), "spike.db")
	client := mcp.NewClient(&mcp.Implementation{Name: "spike-test-client", Version: "0.0.0"}, nil)
	session, err := client.Connect(ctx, &mcp.CommandTransport{
		Command: exec.Command("go", "run", ".", "--stdio"),
	}, nil)
	if err != nil {
		t.Fatalf("connect MCP client over stdio: %v", err)
	}
	defer session.Close()

	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "sqlite_round_trip",
		Arguments: map[string]any{
			"database_path": databasePath,
			"value":         "persisted-through-mcp-stdio",
		},
	})
	if err != nil {
		t.Fatalf("call SQLite round-trip tool: %v", err)
	}
	if result.IsError {
		t.Fatal("SQLite round-trip tool returned an MCP error result")
	}

	db, err := sql.Open("sqlite", databasePath)
	if err != nil {
		t.Fatalf("open persisted SQLite database: %v", err)
	}
	defer db.Close()

	var actual string
	if err := db.QueryRow(`SELECT value FROM spike_values LIMIT 1`).Scan(&actual); err != nil {
		t.Fatalf("read value persisted by MCP tool: %v", err)
	}
	if want := "persisted-through-mcp-stdio"; actual != want {
		t.Fatalf("persisted value = %q, want %q", actual, want)
	}
}

func TestPackageSmoke(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping package smoke integration test in short mode")
	}

	command := exec.Command("go", "run", ".", "--package-smoke")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("run package smoke: %v\n%s", err, output)
	}
	if string(output) != "go-mcp-sqlite spike package smoke passed\n" {
		t.Fatalf("package smoke output = %q", output)
	}
}
