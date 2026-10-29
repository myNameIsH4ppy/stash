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

	mcp.AddTool(s, &mcp.Tool{
		Name:        "add_note",
		Description: "Save a new note or bookmark. Returns the created note, including its id.",
	}, addNote(c))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_note",
		Description: "Fetch one note by id.",
	}, getNote(c))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "delete_note",
		Description: "Permanently delete a note by id.",
	}, deleteNote(c))

	return s
}

type addNoteInput struct {
	Title string   `json:"title" jsonschema:"short title of the note"`
	URL   string   `json:"url,omitempty" jsonschema:"link the note points at, if any"`
	Tags  []string `json:"tags,omitempty" jsonschema:"labels used to group and filter notes"`
}

func addNote(c *client.Client) mcp.ToolHandlerFor[addNoteInput, model.Note] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in addNoteInput) (*mcp.CallToolResult, model.Note, error) {
		n, err := c.CreateNote(ctx, model.NoteInput{Title: in.Title, URL: in.URL, Tags: in.Tags})
		return nil, n, err
	}
}

type noteIDInput struct {
	ID string `json:"id" jsonschema:"the note's id, as returned by add_note or list_notes"`
}

func getNote(c *client.Client) mcp.ToolHandlerFor[noteIDInput, model.Note] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in noteIDInput) (*mcp.CallToolResult, model.Note, error) {
		n, err := c.GetNote(ctx, in.ID)
		return nil, n, err
	}
}

type deleteNoteOutput struct {
	Deleted string `json:"deleted" jsonschema:"id of the note that was deleted"`
}

func deleteNote(c *client.Client) mcp.ToolHandlerFor[noteIDInput, deleteNoteOutput] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in noteIDInput) (*mcp.CallToolResult, deleteNoteOutput, error) {
		if err := c.DeleteNote(ctx, in.ID); err != nil {
			return nil, deleteNoteOutput{}, err
		}
		return nil, deleteNoteOutput{Deleted: in.ID}, nil
	}
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
