package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/myNameIsH4ppy/stash/pkg/client"
	"github.com/myNameIsH4ppy/stash/pkg/model"
)

const defaultAPIURL = "http://localhost:8080"

// newRootCmd builds the command tree. It's a function (not package-level
// vars) so tests can build a fresh tree with their own output buffer and env.
func newRootCmd(out io.Writer) *cobra.Command {
	var apiURL string

	root := &cobra.Command{
		Use:           "stash",
		Short:         "Manage your stash of notes and bookmarks",
		SilenceUsage:  true, // don't dump usage on every runtime error
		SilenceErrors: true, // main prints errors itself
	}
	root.SetOut(out)

	// The flag's default is the env var (or the built-in fallback). Cobra only
	// overwrites it when --api-url is passed, so precedence is
	// flag > STASH_API_URL > default with no extra logic.
	root.PersistentFlags().StringVar(&apiURL, "api-url", envOr("STASH_API_URL", defaultAPIURL),
		"stash API base URL (env: STASH_API_URL)")

	// Commands read apiURL at run time, after flags are parsed.
	newClient := func() *client.Client { return client.New(apiURL) }

	root.AddCommand(
		newAddCmd(out, newClient),
		newListCmd(out, newClient),
		newGetCmd(out, newClient),
		newRmCmd(out, newClient),
	)
	return root
}

func newAddCmd(out io.Writer, newClient func() *client.Client) *cobra.Command {
	var url string
	var tags []string
	cmd := &cobra.Command{
		Use:   "add <title>",
		Short: "Add a note",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			n, err := newClient().CreateNote(cmd.Context(), model.NoteInput{Title: args[0], URL: url, Tags: tags})
			if err != nil {
				return err
			}
			fmt.Fprintf(out, "added %s\n", n.ID)
			return nil
		},
	}
	cmd.Flags().StringVarP(&url, "url", "u", "", "URL to bookmark")
	// StringSlice accepts both --tag a --tag b and --tag a,b.
	cmd.Flags().StringSliceVarP(&tags, "tag", "t", nil, "tag (repeatable or comma-separated)")
	return cmd
}

func newListCmd(out io.Writer, newClient func() *client.Client) *cobra.Command {
	var tag string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List notes",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			notes, err := newClient().ListNotes(cmd.Context(), tag)
			if err != nil {
				return err
			}
			if len(notes) == 0 {
				fmt.Fprintln(out, "no notes")
				return nil
			}
			// tabwriter lines columns up without hand-counting spaces.
			tw := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
			fmt.Fprintln(tw, "ID\tTITLE\tTAGS\tURL")
			for _, n := range notes {
				fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", n.ID, n.Title, strings.Join(n.Tags, ","), n.URL)
			}
			return tw.Flush()
		},
	}
	cmd.Flags().StringVarP(&tag, "tag", "t", "", "only show notes with this tag")
	return cmd
}

func newGetCmd(out io.Writer, newClient func() *client.Client) *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Show one note",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			n, err := newClient().GetNote(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			fmt.Fprintf(out, "ID:      %s\nTitle:   %s\nURL:     %s\nTags:    %s\nCreated: %s\n",
				n.ID, n.Title, n.URL, strings.Join(n.Tags, ", "), n.CreatedAt.Format("2006-01-02 15:04:05 MST"))
			return nil
		},
	}
}

func newRmCmd(out io.Writer, newClient func() *client.Client) *cobra.Command {
	return &cobra.Command{
		Use:   "rm <id>",
		Short: "Delete a note",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := newClient().DeleteNote(cmd.Context(), args[0]); err != nil {
				return err
			}
			fmt.Fprintf(out, "removed %s\n", args[0])
			return nil
		},
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// run is main's testable core: it returns an error instead of calling os.Exit.
func run(ctx context.Context, out io.Writer, args []string) error {
	root := newRootCmd(out)
	root.SetArgs(args)
	return root.ExecuteContext(ctx)
}
