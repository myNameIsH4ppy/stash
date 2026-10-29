// Command stash-mcp is an MCP server exposing the stash API as tools.
// It speaks the protocol over stdin/stdout, so it must never print anything
// else to stdout; diagnostics go to stderr.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/myNameIsH4ppy/stash/pkg/client"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	apiURL := os.Getenv("STASH_API_URL")
	if apiURL == "" {
		apiURL = "http://localhost:8080"
	}

	if err := newServer(client.New(apiURL)).Run(ctx, &mcp.StdioTransport{}); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
