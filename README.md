# stash

A learning project: a tiny notes/bookmarks API plus several ways of talking to it, built to
understand how each piece works. Not a real product.

**Phase 1 (this state of the repo):** HTTP API, shared Go client, CLI.
Planned: MCP server, Terraform provider, Claude skills and agents.

## Setup

Requires Go 1.22+.

The module path is a placeholder, `github.com/YOUR-GITHUB-USER/stash`. Change it before
pushing to GitHub:

```sh
OLD=github.com/YOUR-GITHUB-USER/stash
NEW=github.com/<you>/stash
go mod edit -module $NEW
grep -rl "$OLD" --include='*.go' . | xargs sed -i '' "s#$OLD#$NEW#g"   # macOS sed; on Linux use: sed -i
go build ./...
```

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
| `pkg/client/`       | API client (used by the CLI; later MCP and the provider)       |

`pkg/` rather than `internal/` for the client so it can be imported from another repo later.

## How a request flows

`stash add "Go tour" --tag go` →
`cmd/stash/root.go` (Cobra parses args, builds `NoteInput`) →
`pkg/client` `CreateNote` (JSON `POST /notes`) →
`internal/api` `create` (decode and validate) →
`internal/store` `FileStore.Create` (assign ID, write `stash.json` atomically) →
the new note comes back as JSON, and the CLI prints `added <id>`.
