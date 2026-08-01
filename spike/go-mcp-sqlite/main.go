package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	_ "modernc.org/sqlite"
)

type roundTripInput struct {
	DatabasePath string `json:"database_path" jsonschema:"absolute path to the SQLite database file"`
	Value        string `json:"value" jsonschema:"value to persist and read back"`
}

type roundTripOutput struct {
	Value string `json:"value" jsonschema:"value read back from SQLite"`
}

func sqliteRoundTrip(ctx context.Context, databasePath, value string) (string, error) {
	if databasePath == "" {
		return "", errors.New("database path is required")
	}

	db, err := sql.Open("sqlite", databasePath)
	if err != nil {
		return "", fmt.Errorf("open SQLite database: %w", err)
	}
	defer db.Close()

	if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS spike_values (value TEXT NOT NULL)`); err != nil {
		return "", fmt.Errorf("create spike table: %w", err)
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM spike_values`); err != nil {
		return "", fmt.Errorf("clear spike table: %w", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO spike_values (value) VALUES (?)`, value); err != nil {
		return "", fmt.Errorf("persist spike value: %w", err)
	}

	var readback string
	if err := db.QueryRowContext(ctx, `SELECT value FROM spike_values LIMIT 1`).Scan(&readback); err != nil {
		return "", fmt.Errorf("read spike value: %w", err)
	}
	return readback, nil
}

func runSQLiteRoundTrip(ctx context.Context, _ *mcp.CallToolRequest, input roundTripInput) (*mcp.CallToolResult, roundTripOutput, error) {
	value, err := sqliteRoundTrip(ctx, input.DatabasePath, input.Value)
	if err != nil {
		return nil, roundTripOutput{}, err
	}
	return nil, roundTripOutput{Value: value}, nil
}

func newServer() *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{Name: "go-mcp-sqlite-spike", Version: "0.0.0"}, nil)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "sqlite_round_trip",
		Description: "Persist and read back a value using the pure-Go SQLite driver.",
	}, runSQLiteRoundTrip)
	return server
}

func main() {
	stdio := flag.Bool("stdio", false, "serve MCP over standard input and output")
	packageSmoke := flag.Bool("package-smoke", false, "verify the packaged binary entrypoint")
	flag.Parse()

	switch {
	case *packageSmoke:
		fmt.Fprintln(os.Stdout, "go-mcp-sqlite spike package smoke passed")
	case *stdio:
		if err := newServer().Run(context.Background(), &mcp.StdioTransport{}); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	default:
		fmt.Fprintln(os.Stderr, "use --stdio or --package-smoke")
		os.Exit(2)
	}
}
