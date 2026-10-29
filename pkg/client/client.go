// Package client is a Go client for the stash API. The CLI, MCP server and
// Terraform provider all go through it so the HTTP details live in one place.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/YOUR-GITHUB-USER/stash/pkg/model"
)

// Client talks to a stash API server.
type Client struct {
	baseURL string
	http    *http.Client
}

// New returns a Client for the API at baseURL (e.g. "http://localhost:8080").
func New(baseURL string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		// http.DefaultClient has no timeout; a hung server would hang us forever.
		http: &http.Client{Timeout: 30 * time.Second},
	}
}

// APIError is returned when the server replies with a non-2xx status.
// Callers can use errors.As to inspect StatusCode (e.g. 404).
type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("stash api: %d %s: %s", e.StatusCode, http.StatusText(e.StatusCode), e.Message)
}

// IsNotFound reports whether err is an APIError with status 404.
func IsNotFound(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound
}

// ListNotes returns all notes, or only those carrying tag if tag is non-empty.
func (c *Client) ListNotes(ctx context.Context, tag string) ([]model.Note, error) {
	path := "/notes"
	if tag != "" {
		path += "?tag=" + url.QueryEscape(tag)
	}
	var notes []model.Note
	if err := c.do(ctx, http.MethodGet, path, nil, &notes); err != nil {
		return nil, err
	}
	return notes, nil
}

// GetNote returns the note with the given id.
func (c *Client) GetNote(ctx context.Context, id string) (model.Note, error) {
	var n model.Note
	err := c.do(ctx, http.MethodGet, notePath(id), nil, &n)
	return n, err
}

// CreateNote creates a note; the server assigns its ID and creation time.
func (c *Client) CreateNote(ctx context.Context, in model.NoteInput) (model.Note, error) {
	var n model.Note
	err := c.do(ctx, http.MethodPost, "/notes", in, &n)
	return n, err
}

// UpdateNote replaces the title, URL and tags of an existing note.
func (c *Client) UpdateNote(ctx context.Context, id string, in model.NoteInput) (model.Note, error) {
	var n model.Note
	err := c.do(ctx, http.MethodPut, notePath(id), in, &n)
	return n, err
}

// DeleteNote removes a note.
func (c *Client) DeleteNote(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, notePath(id), nil, nil)
}

// notePath escapes the id so odd input can't alter the URL's shape.
func notePath(id string) string {
	return "/notes/" + url.PathEscape(id)
}

// do sends a request, JSON-encoding body (if non-nil) and decoding the
// response into out (if non-nil).
func (c *Client) do(ctx context.Context, method, path string, body, out any) error {
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encode request: %w", err)
		}
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("%s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return parseAPIError(resp)
	}
	if out == nil {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

func parseAPIError(resp *http.Response) error {
	var payload struct {
		Error string `json:"error"`
	}
	// Best effort: if the body isn't our JSON error shape, fall back to the status text.
	_ = json.NewDecoder(io.LimitReader(resp.Body, 1<<16)).Decode(&payload)
	if payload.Error == "" {
		payload.Error = http.StatusText(resp.StatusCode)
	}
	return &APIError{StatusCode: resp.StatusCode, Message: payload.Error}
}
