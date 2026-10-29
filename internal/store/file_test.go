package store

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/YOUR-GITHUB-USER/stash/pkg/model"
)

func TestFileStoreCRUD(t *testing.T) {
	path := filepath.Join(t.TempDir(), "notes.json")
	s := NewFileStore(path)

	notes, err := s.List()
	if err != nil || len(notes) != 0 {
		t.Fatalf("empty store: got %v, %v", notes, err)
	}

	created, err := s.Create(model.NoteInput{Title: "Go", URL: "https://go.dev"})
	if err != nil {
		t.Fatal(err)
	}
	if created.ID == "" || created.CreatedAt.IsZero() || created.Tags == nil {
		t.Errorf("Create did not fill server-owned fields: %+v", created)
	}

	// A fresh store on the same file proves the data really hit the disk.
	got, err := NewFileStore(path).Get(created.ID)
	if err != nil || got.Title != "Go" {
		t.Fatalf("Get after reopen: %+v, %v", got, err)
	}

	updated, err := s.Update(created.ID, model.NoteInput{Title: "Golang", Tags: []string{"lang"}})
	if err != nil || updated.Title != "Golang" || updated.CreatedAt != created.CreatedAt {
		t.Fatalf("Update: %+v, %v", updated, err)
	}

	if err := s.Delete(created.ID); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		call func() error
	}{
		{"get", func() error { _, err := s.Get(created.ID); return err }},
		{"update", func() error { _, err := s.Update(created.ID, model.NoteInput{Title: "x"}); return err }},
		{"delete", func() error { return s.Delete(created.ID) }},
	}
	for _, tt := range tests {
		t.Run(tt.name+" missing", func(t *testing.T) {
			if err := tt.call(); !errors.Is(err, ErrNotFound) {
				t.Errorf("got %v, want ErrNotFound", err)
			}
		})
	}
}
