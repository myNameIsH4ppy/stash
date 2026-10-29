# stash

A learning project: a set of experiments for understanding how things work, all built
against one small shared API. Not a real product. The domain is deliberately boring: a
notes/bookmarks tracker.

A **note** has: `id`, `title`, `url`, `tags[]`, `created_at`.

## Goals (learn by building)

1. A CLI in Go (Cobra)
2. A Terraform provider (Plugin Framework)
3. An MCP server
4. Claude skills and agents

The owner (Felice) is learning the stack as we go: explain the concept and the *why* before
code, lead with an analogy, build in small working pieces.

## Status

- Phase 1 (done): API, `pkg/client`, CLI, README.
- Later, only when asked: MCP server, Terraform provider, skills, agents. Do not start these
  unprompted.

## Layout

```
cmd/stash-api/              HTTP service (main only: flags, wiring, graceful shutdown)
cmd/stash/                  CLI (Cobra)
cmd/stash-mcp/              MCP server (later)
internal/api/               HTTP handlers (net/http, Go 1.22+ method+path routing)
internal/store/             Store interface + JSON-file implementation
pkg/model/                  Shared types (Note, NoteInput) used by server and client
pkg/client/                 Shared API client, used by CLI, MCP server and provider
terraform-provider-stash/   (later)
skills/                     SKILL.md files (later)
agents/                     agent definitions (later)
```

`pkg/` (not `internal/`) for client and model so they can be imported if the provider is
later split into its own repo. The server internals stay in `internal/`.

Module path: `github.com/myNameIsH4ppy/stash` (go.mod and every import).

## API

stdlib `net/http`, no framework. Storage is a JSON file behind the `store.Store` interface.

| Method | Path          | Notes                                  |
|--------|---------------|----------------------------------------|
| GET    | /notes        | optional `?tag=` filter                |
| POST   | /notes        | body: NoteInput, returns 201 + Note    |
| GET    | /notes/{id}   | 404 if missing                         |
| PUT    | /notes/{id}   | full replace of title/url/tags         |
| DELETE | /notes/{id}   | 204                                    |

Errors are JSON: `{"error": "message"}`.

## Conventions

- Prefer the standard library. No speculative abstractions, auth, or config systems.
- Idiomatic Go: small packages, wrap errors with `%w`, table-driven tests,
  `context.Context` as first arg on client calls.
- Config: env var `STASH_API_URL`, overridable by `--api-url`.
- Before calling work done: `go vet ./...` and `go test ./...` must pass.
- Conventional commit messages (`feat:`, `fix:`, `docs:`, `test:`, `chore:`), small commits per
  step. Felice runs the commits; suggest the commands rather than running them.
