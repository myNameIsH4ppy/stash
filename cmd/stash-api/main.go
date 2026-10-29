// Command stash-api serves the stash notes API over HTTP.
package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/myNameIsH4ppy/stash/internal/api"
	"github.com/myNameIsH4ppy/stash/internal/store"
)

func main() {
	addr := flag.String("addr", "localhost:8080", "address to listen on")
	data := flag.String("data", "stash.json", "path of the JSON data file")
	flag.Parse()

	srv := &http.Server{
		Addr:              *addr,
		Handler:           api.NewHandler(store.NewFileStore(*data)),
		ReadHeaderTimeout: 5 * time.Second,
	}

	// Ctrl-C / SIGTERM cancels ctx, which lets in-flight requests finish.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv.Shutdown(shutdownCtx)
	}()

	log.Printf("stash-api listening on %s, data in %s", *addr, *data)
	if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
