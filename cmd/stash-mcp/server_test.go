package main

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/myNameIsH4ppy/stash/internal/api"
	"github.com/myNameIsH4ppy/stash/internal/store"
	"github.com/myNameIsH4ppy/stash/pkg/client"
)

// newSession wires a real MCP client to our server over an in-memory pipe,
// with the server backed by the real API handler. Tests then call tools the
// same way Claude would, with no subprocess and no mocks.
func newSession(t *testing.T) *mcp.ClientSession {
	t.Helper()
	ctx := context.Background()

	srv := httptest.NewServer(api.NewHandler(store.NewFileStore(filepath.Join(t.TempDir(), "notes.json"))))
	t.Cleanup(srv.Close)

	serverT, clientT := mcp.NewInMemoryTransports()
	if _, err := newServer(client.New(srv.URL)).Connect(ctx, serverT, nil); err != nil {
		t.Fatal(err)
	}
	cs, err := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "0"}, nil).Connect(ctx, clientT, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cs.Close() })
	return cs
}

// call invokes a tool and returns the result's structured content as a map.
func call(t *testing.T, cs *mcp.ClientSession, name string, args any) (map[string]any, *mcp.CallToolResult) {
	t.Helper()
	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("%s: protocol error: %v", name, err)
	}
	var out map[string]any
	if res.StructuredContent != nil {
		b, _ := json.Marshal(res.StructuredContent)
		if err := json.Unmarshal(b, &out); err != nil {
			t.Fatal(err)
		}
	}
	return out, res
}

func TestToolLifecycle(t *testing.T) {
	cs := newSession(t)

	added, res := call(t, cs, "add_note", map[string]any{"title": "Go tour", "url": "https://go.dev/tour", "tags": []string{"go"}})
	if res.IsError {
		t.Fatalf("add_note failed: %+v", res.Content)
	}
	id, _ := added["id"].(string)
	if id == "" || added["title"] != "Go tour" {
		t.Fatalf("add_note = %v", added)
	}

	got, _ := call(t, cs, "get_note", map[string]any{"id": id})
	if got["id"] != id {
		t.Fatalf("get_note = %v", got)
	}

	tests := []struct {
		args map[string]any
		want int
	}{
		{map[string]any{}, 1},
		{map[string]any{"tag": "go"}, 1},
		{map[string]any{"tag": "nope"}, 0},
	}
	for _, tt := range tests {
		listed, _ := call(t, cs, "list_notes", tt.args)
		if n := len(listed["notes"].([]any)); n != tt.want {
			t.Errorf("list_notes(%v) returned %d notes, want %d", tt.args, n, tt.want)
		}
	}

	deleted, _ := call(t, cs, "delete_note", map[string]any{"id": id})
	if deleted["deleted"] != id {
		t.Fatalf("delete_note = %v", deleted)
	}
	if _, res := call(t, cs, "get_note", map[string]any{"id": id}); !res.IsError {
		t.Error("get_note after delete should be a tool error")
	}
}

// API failures must come back as tool errors (IsError) carrying the server's
// message, so the model can read what went wrong instead of the call dying.
func TestToolErrors(t *testing.T) {
	cs := newSession(t)

	tests := []struct {
		name string
		tool string
		args map[string]any
		want string
	}{
		{"missing note", "get_note", map[string]any{"id": "nope"}, "404"},
		{"delete missing note", "delete_note", map[string]any{"id": "nope"}, "404"},
		{"empty title rejected by API", "add_note", map[string]any{"title": ""}, "400"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, res := call(t, cs, tt.tool, tt.args)
			if !res.IsError {
				t.Fatal("want tool error, got success")
			}
			text := res.Content[0].(*mcp.TextContent).Text
			if !strings.Contains(text, tt.want) {
				t.Errorf("error text %q does not contain %q", text, tt.want)
			}
		})
	}
}
