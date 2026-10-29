package store

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/YOUR-GITHUB-USER/stash/pkg/model"
)

// FileStore keeps all notes in one JSON file. It re-reads the file on every
// call: slower than caching, but there is no in-memory state to get out of
// sync, which is the right trade for a learning project.
type FileStore struct {
	path string
	mu   sync.Mutex // HTTP handlers run concurrently; serialize file access.
}

// NewFileStore returns a store backed by path. The file is created on first write.
func NewFileStore(path string) *FileStore {
	return &FileStore{path: path}
}

func (s *FileStore) load() ([]model.Note, error) {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return []model.Note{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", s.path, err)
	}
	var notes []model.Note
	if err := json.Unmarshal(data, &notes); err != nil {
		return nil, fmt.Errorf("parse %s: %w", s.path, err)
	}
	return notes, nil
}

// save writes to a temp file then renames it over the real one. Rename is
// atomic, so a crash mid-write can't leave a half-written notes file.
func (s *FileStore) save(notes []model.Note) error {
	data, err := json.MarshalIndent(notes, "", "  ")
	if err != nil {
		return fmt.Errorf("encode notes: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(s.path), filepath.Base(s.path)+".*.tmp")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	defer os.Remove(tmp.Name()) // no-op after a successful rename
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}
	if err := os.Rename(tmp.Name(), s.path); err != nil {
		return fmt.Errorf("replace %s: %w", s.path, err)
	}
	return nil
}

func (s *FileStore) List() ([]model.Note, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.load()
}

func (s *FileStore) Get(id string) (model.Note, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	notes, err := s.load()
	if err != nil {
		return model.Note{}, err
	}
	for _, n := range notes {
		if n.ID == id {
			return n, nil
		}
	}
	return model.Note{}, ErrNotFound
}

func (s *FileStore) Create(in model.NoteInput) (model.Note, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	notes, err := s.load()
	if err != nil {
		return model.Note{}, err
	}
	id, err := newID()
	if err != nil {
		return model.Note{}, err
	}
	n := model.Note{
		ID:        id,
		Title:     in.Title,
		URL:       in.URL,
		Tags:      nonNil(in.Tags),
		CreatedAt: time.Now().UTC().Truncate(time.Second),
	}
	if err := s.save(append(notes, n)); err != nil {
		return model.Note{}, err
	}
	return n, nil
}

func (s *FileStore) Update(id string, in model.NoteInput) (model.Note, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	notes, err := s.load()
	if err != nil {
		return model.Note{}, err
	}
	for i := range notes {
		if notes[i].ID == id {
			notes[i].Title = in.Title
			notes[i].URL = in.URL
			notes[i].Tags = nonNil(in.Tags)
			if err := s.save(notes); err != nil {
				return model.Note{}, err
			}
			return notes[i], nil
		}
	}
	return model.Note{}, ErrNotFound
}

func (s *FileStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	notes, err := s.load()
	if err != nil {
		return err
	}
	for i := range notes {
		if notes[i].ID == id {
			return s.save(append(notes[:i], notes[i+1:]...))
		}
	}
	return ErrNotFound
}

// newID returns 8 random hex chars: short enough to type in a CLI.
func newID() (string, error) {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate id: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// nonNil makes tags encode as [] rather than null in JSON.
func nonNil(tags []string) []string {
	if tags == nil {
		return []string{}
	}
	return tags
}
