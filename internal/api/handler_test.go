package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/myNameIsH4ppy/stash/internal/store"
	"github.com/myNameIsH4ppy/stash/pkg/model"
)

func newTestHandler(t *testing.T) http.Handler {
	t.Helper()
	return NewHandler(store.NewFileStore(filepath.Join(t.TempDir(), "notes.json")))
}

func do(h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestValidationAndNotFound(t *testing.T) {
	h := newTestHandler(t)
	tests := []struct {
		name         string
		method, path string
		body         string
		want         int
	}{
		{"create ok", "POST", "/notes", `{"title":"a"}`, http.StatusCreated},
		{"create missing title", "POST", "/notes", `{"url":"x"}`, http.StatusBadRequest},
		{"create blank title", "POST", "/notes", `{"title":"  "}`, http.StatusBadRequest},
		{"create bad json", "POST", "/notes", `{`, http.StatusBadRequest},
		{"create unknown field", "POST", "/notes", `{"title":"a","titel":"b"}`, http.StatusBadRequest},
		{"get missing", "GET", "/notes/nope", "", http.StatusNotFound},
		{"update missing", "PUT", "/notes/nope", `{"title":"a"}`, http.StatusNotFound},
		{"update invalid", "PUT", "/notes/nope", `{}`, http.StatusBadRequest},
		{"delete missing", "DELETE", "/notes/nope", "", http.StatusNotFound},
		{"wrong method", "PATCH", "/notes/nope", "", http.StatusMethodNotAllowed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := do(h, tt.method, tt.path, tt.body)
			if rec.Code != tt.want {
				t.Errorf("got %d, want %d (body %s)", rec.Code, tt.want, rec.Body)
			}
		})
	}
}

func TestLifecycleAndTagFilter(t *testing.T) {
	h := newTestHandler(t)

	var a, b model.Note
	rec := do(h, "POST", "/notes", `{"title":"A","url":"https://a","tags":["go","web"]}`)
	json.Unmarshal(rec.Body.Bytes(), &a)
	rec = do(h, "POST", "/notes", `{"title":"B","tags":["web"]}`)
	json.Unmarshal(rec.Body.Bytes(), &b)
	if a.ID == "" || a.ID == b.ID {
		t.Fatalf("bad ids: %q %q", a.ID, b.ID)
	}

	tests := []struct {
		query string
		want  int
	}{
		{"", 2},
		{"?tag=web", 2},
		{"?tag=go", 1},
		{"?tag=none", 0},
	}
	for _, tt := range tests {
		t.Run("list"+tt.query, func(t *testing.T) {
			rec := do(h, "GET", "/notes"+tt.query, "")
			var got []model.Note
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatal(err)
			}
			if len(got) != tt.want {
				t.Errorf("got %d notes, want %d", len(got), tt.want)
			}
		})
	}

	rec = do(h, "PUT", "/notes/"+a.ID, `{"title":"A2"}`)
	var updated model.Note
	json.Unmarshal(rec.Body.Bytes(), &updated)
	if rec.Code != http.StatusOK || updated.Title != "A2" || updated.ID != a.ID {
		t.Errorf("update: %d %+v", rec.Code, updated)
	}

	if rec := do(h, "DELETE", "/notes/"+a.ID, ""); rec.Code != http.StatusNoContent {
		t.Errorf("delete: %d", rec.Code)
	}
	if rec := do(h, "GET", "/notes/"+a.ID, ""); rec.Code != http.StatusNotFound {
		t.Errorf("get after delete: %d", rec.Code)
	}
}
