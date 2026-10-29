package main

import (
	"bytes"
	"context"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/YOUR-GITHUB-USER/stash/internal/api"
	"github.com/YOUR-GITHUB-USER/stash/internal/store"
)

func newServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(api.NewHandler(store.NewFileStore(filepath.Join(t.TempDir(), "n.json"))))
	t.Cleanup(srv.Close)
	return srv
}

func runCLI(t *testing.T, args ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	err := run(context.Background(), &out, args)
	return out.String(), err
}

func TestCLIFlow(t *testing.T) {
	srv := newServer(t)
	t.Setenv("STASH_API_URL", srv.URL) // env var path

	out, err := runCLI(t, "add", "Learn Go", "--url", "https://go.dev", "-t", "lang,go")
	if err != nil || !strings.HasPrefix(out, "added ") {
		t.Fatalf("add: %q, %v", out, err)
	}
	id := strings.TrimSpace(strings.TrimPrefix(out, "added "))
	if _, err := runCLI(t, "add", "Other", "--tag", "misc"); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		args     []string
		contains []string
		excludes []string
	}{
		{"list all", []string{"list"}, []string{"Learn Go", "Other"}, nil},
		{"list by tag", []string{"list", "--tag", "go"}, []string{"Learn Go"}, []string{"Other"}},
		{"list no match", []string{"list", "--tag", "zzz"}, []string{"no notes"}, nil},
		{"get", []string{"get", id}, []string{"Learn Go", "https://go.dev", "lang, go"}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := runCLI(t, tt.args...)
			if err != nil {
				t.Fatal(err)
			}
			for _, s := range tt.contains {
				if !strings.Contains(out, s) {
					t.Errorf("output missing %q:\n%s", s, out)
				}
			}
			for _, s := range tt.excludes {
				if strings.Contains(out, s) {
					t.Errorf("output should not contain %q:\n%s", s, out)
				}
			}
		})
	}

	if _, err := runCLI(t, "rm", id); err != nil {
		t.Fatal(err)
	}
	if _, err := runCLI(t, "get", id); err == nil || !strings.Contains(err.Error(), "404") {
		t.Errorf("get after rm: got %v, want 404 error", err)
	}
}

func TestAPIURLFlagOverridesEnv(t *testing.T) {
	srv := newServer(t)
	t.Setenv("STASH_API_URL", "http://127.0.0.1:1") // nothing listens here

	if _, err := runCLI(t, "list"); err == nil {
		t.Fatal("expected connection error using env URL")
	}
	if _, err := runCLI(t, "--api-url", srv.URL, "list"); err != nil {
		t.Errorf("flag should win over env: %v", err)
	}
}
