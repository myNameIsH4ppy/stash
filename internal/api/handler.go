// Package api implements the stash HTTP API.
package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"strings"

	"github.com/YOUR-GITHUB-USER/stash/internal/store"
	"github.com/YOUR-GITHUB-USER/stash/pkg/model"
)

const maxBodyBytes = 1 << 20 // 1 MiB is plenty for a bookmark

type handler struct {
	store store.Store
}

// NewHandler returns the API's http.Handler. It takes the Store interface, so
// tests can pass a temp-file store and main passes the real one.
func NewHandler(s store.Store) http.Handler {
	h := &handler{store: s}
	mux := http.NewServeMux()
	// Go 1.22+: patterns can include the HTTP method and {wildcards}, which is
	// what lets us skip a router library.
	mux.HandleFunc("GET /notes", h.list)
	mux.HandleFunc("POST /notes", h.create)
	mux.HandleFunc("GET /notes/{id}", h.get)
	mux.HandleFunc("PUT /notes/{id}", h.update)
	mux.HandleFunc("DELETE /notes/{id}", h.delete)
	return mux
}

func (h *handler) list(w http.ResponseWriter, r *http.Request) {
	notes, err := h.store.List()
	if err != nil {
		writeStoreError(w, err)
		return
	}
	if tag := r.URL.Query().Get("tag"); tag != "" {
		notes = slices.DeleteFunc(notes, func(n model.Note) bool {
			return !slices.Contains(n.Tags, tag)
		})
	}
	writeJSON(w, http.StatusOK, notes)
}

func (h *handler) create(w http.ResponseWriter, r *http.Request) {
	in, ok := decodeInput(w, r)
	if !ok {
		return
	}
	n, err := h.store.Create(in)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, n)
}

func (h *handler) get(w http.ResponseWriter, r *http.Request) {
	n, err := h.store.Get(r.PathValue("id"))
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, n)
}

func (h *handler) update(w http.ResponseWriter, r *http.Request) {
	in, ok := decodeInput(w, r)
	if !ok {
		return
	}
	n, err := h.store.Update(r.PathValue("id"), in)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, n)
}

func (h *handler) delete(w http.ResponseWriter, r *http.Request) {
	if err := h.store.Delete(r.PathValue("id")); err != nil {
		writeStoreError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// decodeInput parses and validates a request body. On failure it has already
// written the error response and returns ok=false.
func decodeInput(w http.ResponseWriter, r *http.Request) (model.NoteInput, bool) {
	var in model.NoteInput
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	dec.DisallowUnknownFields() // catch typos like "titel" instead of silently ignoring them
	if err := dec.Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return in, false
	}
	if strings.TrimSpace(in.Title) == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return in, false
	}
	return in, true
}

func writeStoreError(w http.ResponseWriter, err error) {
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	// Don't leak internal details (file paths etc.) to callers.
	writeError(w, http.StatusInternalServerError, "internal error")
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
