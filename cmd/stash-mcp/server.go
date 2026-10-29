package main

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/myNameIsH4ppy/stash/pkg/client"
	"github.com/myNameIsH4ppy/stash/pkg/model"
)

// newServer builds the MCP server and registers its tools. Taking the client
// as a parameter (instead of creating it here) lets tests point it at an
// httptest server.
func newServer(c *client.Client) *mcp.Server {
	s := mcp.NewServer(&mcp.Implementation{Name: "stash", Version: "0.1.0"}, nil)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_notes",
		Description: "List saved notes, optionally only those with a given tag.",
	}, listNotes(c))

	return s
}

// The input struct doubles as the tool's schema: the SDK derives the JSON
// schema from the field types and the `jsonschema` description tags, and this
// is what Claude reads to decide how to call the tool.
type listNotesInput struct {
	Tag string `json:"tag,omitempty" jsonschema:"only return notes carrying this tag"`
}

// MCP structured output must be a JSON object, so the slice is wrapped.
type listNotesOutput struct {
	Notes []model.Note `json:"notes"`
}

func listNotes(c *client.Client) mcp.ToolHandlerFor[listNotesInput, listNotesOutput] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in listNotesInput) (*mcp.CallToolResult, listNotesOutput, error) {
		notes, err := c.ListNotes(ctx, in.Tag)
		if err != nil {
			// Returned errors become a tool error the model can read and react
			// to, rather than a protocol failure.
			return nil, listNotesOutput{}, err
		}
		if notes == nil {
			notes = []model.Note{} // encode as [] rather than null
		}
		return nil, listNotesOutput{Notes: notes}, nil
	}
}
