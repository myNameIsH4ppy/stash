# stash

A learning project: a tiny notes/bookmarks API plus several ways of talking to it, built to
understand how each piece works. Not a real product.

**So far:** HTTP API, shared Go client, CLI, MCP server.
Planned: Terraform provider, Claude skills and agents.

## Setup

Requires Go 1.25+ (see `go.mod`). The module path is `github.com/myNameIsH4ppy/stash`.

## Run the API

```sh
go run ./cmd/stash-api                 # listens on localhost:8080, data in ./stash.json
go run ./cmd/stash-api --addr localhost:9000 --data /tmp/notes.json
```

Try it with curl:

```sh
curl -X POST localhost:8080/notes -d '{"title":"Go tour","url":"https://go.dev/tour","tags":["go"]}'
curl localhost:8080/notes
curl 'localhost:8080/notes?tag=go'
curl localhost:8080/notes/<id>
curl -X PUT localhost:8080/notes/<id> -d '{"title":"New title"}'   # replaces title/url/tags
curl -X DELETE localhost:8080/notes/<id>
```

## Use the CLI

Build it once, or use `go run ./cmd/stash`:

```sh
go build -o bin/stash ./cmd/stash

bin/stash add "Go tour" --url https://go.dev/tour --tag go --tag learn
bin/stash list
bin/stash list --tag go
bin/stash get <id>
bin/stash rm <id>
```

The API location comes from `--api-url`, then the `STASH_API_URL` env var, then
`http://localhost:8080`:

```sh
export STASH_API_URL=http://localhost:9000
bin/stash --api-url http://other-host:8080 list   # flag wins over the env var
```

## Use it from Claude (MCP server)

MCP (Model Context Protocol) is a standard way for an AI assistant to discover and call tools.
`stash-mcp` exposes the API as four tools: `list_notes` (optional `tag`), `add_note`, `get_note`
and `delete_note`. Claude Code starts it as a child process and talks to it over stdin/stdout,
so there is no extra port. The server reads `STASH_API_URL` like the CLI does.

With the API running, build the server and register it with Claude Code:

```sh
go build -o bin/stash-mcp ./cmd/stash-mcp
claude mcp add stash --env STASH_API_URL=http://localhost:8080 -- "$PWD/bin/stash-mcp"
```

Start a new Claude Code session, run `/mcp` to check that `stash` is connected, then ask
something like "save a note titled Go tour with tag go".

Never print to stdout from this binary: stdout carries the protocol, so stray output corrupts it.
Errors from the API come back to Claude as tool errors it can read, not as crashes.

## Test

```sh
go vet ./...
go test ./...
```

## Layout

| Path                | What it is                                                     |
|---------------------|----------------------------------------------------------------|
| `cmd/stash-api/`    | API server entrypoint (flags, wiring, graceful shutdown)       |
| `cmd/stash/`        | Cobra CLI                                                      |
| `internal/api/`     | HTTP handlers                                                  |
| `internal/store/`   | `Store` interface + JSON-file implementation                   |
| `pkg/model/`        | `Note` / `NoteInput` types shared by server and client         |
| `cmd/stash-mcp/`    | MCP server (tools wrapping `pkg/client`)                       |
| `pkg/client/`       | API client (used by the CLI and MCP server; later the provider) |

`pkg/` rather than `internal/` for the client so it can be imported from another repo later.

## How a request flows

`stash add "Go tour" --tag go` →
`cmd/stash/root.go` (Cobra parses args, builds `NoteInput`) →
`pkg/client` `CreateNote` (JSON `POST /notes`) →
`internal/api` `create` (decode and validate) →
`internal/store` `FileStore.Create` (assign ID, write `stash.json` atomically) →
the new note comes back as JSON, and the CLI prints `added <id>`.
