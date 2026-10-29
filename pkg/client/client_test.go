package client_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/YOUR-GITHUB-USER/stash/internal/api"
	"github.com/YOUR-GITHUB-USER/stash/internal/store"
	"github.com/YOUR-GITHUB-USER/stash/pkg/client"
	"github.com/YOUR-GITHUB-USER/stash/pkg/model"
)

// newClient starts the real API handler on a throwaway local server, so these
// tests exercise client and server together (no mocks to drift out of date).
func newClient(t *testing.T) *client.Client {
	t.Helper()
	h := api.NewHandler(store.NewFileStore(filepath.Join(t.TempDir(), "notes.json")))
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return client.New(srv.URL + "/") // trailing slash on purpose: must be tolerated
}

func TestRoundTrip(t *testing.T) {
	ctx := context.Background()
	c := newClient(t)

	created, err := c.CreateNote(ctx, model.NoteInput{Title: "Go", URL: "https://go.dev", Tags: []string{"lang"}})
	if err != nil {
		t.Fatal(err)
	}
	got, err := c.GetNote(ctx, created.ID)
	if err != nil || !reflect.DeepEqual(got, created) {
		t.Fatalf("GetNote = %+v, %v; want %+v", got, err, created)
	}

	updated, err := c.UpdateNote(ctx, created.ID, model.NoteInput{Title: "Golang"})
	if err != nil || updated.Title != "Golang" {
		t.Fatalf("UpdateNote = %+v, %v", updated, err)
	}

	if err := c.DeleteNote(ctx, created.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := c.GetNote(ctx, created.ID); !client.IsNotFound(err) {
		t.Errorf("GetNote after delete: got %v, want not-found", err)
	}
}

func TestListNotesTagFilter(t *testing.T) {
	ctx := context.Background()
	c := newClient(t)
	for _, in := range []model.NoteInput{
		{Title: "A", Tags: []string{"go", "web"}},
		{Title: "B", Tags: []string{"web"}},
		{Title: "C"},
	} {
		if _, err := c.CreateNote(ctx, in); err != nil {
			t.Fatal(err)
		}
	}
	tests := []struct {
		tag  string
		want int
	}{
		{"", 3},
		{"web", 2},
		{"go", 1},
		{"a b&c", 0}, // exercises query escaping
	}
	for _, tt := range tests {
		t.Run("tag="+tt.tag, func(t *testing.T) {
			notes, err := c.ListNotes(ctx, tt.tag)
			if err != nil || len(notes) != tt.want {
				t.Errorf("got %d notes, %v; want %d", len(notes), err, tt.want)
			}
		})
	}
}

func TestErrors(t *testing.T) {
	ctx := context.Background()
	c := newClient(t)

	_, err := c.CreateNote(ctx, model.NoteInput{}) // missing title
	var apiErr *client.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusBadRequest || apiErr.Message != "title is required" {
		t.Errorf("CreateNote invalid: got %v", err)
	}

	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := c.ListNotes(cancelled, ""); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled ctx: got %v, want context.Canceled", err)
	}
}
