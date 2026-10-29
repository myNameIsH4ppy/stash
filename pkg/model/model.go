// Package model holds the types shared by the API server and its clients.
// It lives in its own package so neither side has to import the other.
package model

import "time"

// Note is a saved bookmark/note.
type Note struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	URL       string    `json:"url"`
	Tags      []string  `json:"tags"`
	CreatedAt time.Time `json:"created_at"`
}

// NoteInput is what a caller supplies to create or replace a note. The server
// owns ID and CreatedAt, so they are deliberately absent here.
type NoteInput struct {
	Title string   `json:"title"`
	URL   string   `json:"url"`
	Tags  []string `json:"tags"`
}
