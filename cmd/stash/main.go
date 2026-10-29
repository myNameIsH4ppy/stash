// Command stash is the CLI for the stash API.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
)

func main() {
	// Ctrl-C cancels ctx, which aborts any in-flight HTTP request.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if err := run(ctx, os.Stdout, os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
