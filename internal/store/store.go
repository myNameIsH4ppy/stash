// Package store defines how notes are persisted.
package store

import (
	"errors"

	"github.com/myNameIsH4ppy/stash/pkg/model"
)

// ErrNotFound is returned when a note ID does not exist.
var ErrNotFound = errors.New("note not found")

// Store is the persistence contract the HTTP handlers depend on. The handlers
// never know whether notes live in a JSON file, SQLite, or memory, so the
// backend can be swapped without touching them.
type Store interface {
	List() ([]model.Note, error)
	Get(id string) (model.Note, error)
	// Create assigns the ID and creation time.
	Create(in model.NoteInput) (model.Note, error)
	Update(id string, in model.NoteInput) (model.Note, error)
	Delete(id string) error
}
