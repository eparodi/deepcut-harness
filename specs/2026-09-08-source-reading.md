# Source Reading Layer (Draft)

**Feature slug:** `source-reading`
**Status:** Implemented
**Owner:** PM
**Created:** 2026-09-08

## Requirements

### User stories

- As the wizard (or any agent), I can read a local file by path,
  constrained to a workspace root (no `..` escape).
- As the operator, I choose the workspace root for a session (the
  chat/session feature); the reader resolves relative paths against it.
- As the wizard, I can fetch a URL over HTTP(S), bounded by a size cap and
  a timeout.
- Fetched/read content is treated as **untrusted data** injected into the
  LLM prompt (never instructions).

### Acceptance criteria

- `Read(path)` refuses `..` escapes and paths outside the workspace root.
- The reader resolves relative paths against a caller-supplied workspace
  root (session-scoped, default CWD).
- `Fetch(url)` caps the response size and enforces a timeout; non-HTTP(S)
  schemes are rejected.
- Content is returned as data, and the wizard's prompt marks it as such.
- Table-driven tests: happy path, traversal escape, non-http scheme,
  oversized response, timeout.
- `make build` / `make vet` / `make test` green.

## Explicit Non-Goals

- No command execution (that's the full `internal/exec` workspace-jail
  executor, later).
- No write operations.
- No `git clone` of remote repos — "repos" means reading files in a local
  workspace directory this step.

## Design

### Package

`internal/source`:

```go
// Reader is the read-only source seam the wizard uses.
type Reader interface {
	Read(ctx context.Context, path string) (string, error)
	Fetch(ctx context.Context, url string) (string, error)
}
```

### Guards

- **Path traversal** — resolve against a workspace root; reject `..`,
  absolute paths, and any path that escapes the root.
- **Fetch** — `http.Client` with a timeout, a max-body cap (read max+1 and
  error on overflow), and a scheme allow-list (`http`/`https`).
- **Prompt injection** — the caller marks returned content as data; the
  source layer never interprets it.

## Task Checklist

- [x] `internal/source` — `Reader` interface + implementation
- [x] Path-traversal guard (workspace root) + symlink guard
- [x] Fetch with timeout + size cap + scheme allow-list + SSRF guards
- [x] Config `source` block (`block_private_hosts`, `max_bytes`, `timeout_ms`)
- [x] Table-driven tests (happy, traversal, absolute, symlink, non-http, metadata, private, oversized, timeout)
- [x] `make build` / `make vet` / `make test` green

## Decisions (resolved at the gate)

- **Workspace root is session-scoped, not a setting.** The workspace root
  is chosen per chat/session (like other harnesses: the workspace is part
  of the conversation, not a global toggle). It is set on `source.Source.Root`
  by the chat/session feature; this step does not expose it in `/settings`
  or as a config field. The fetch-safety defaults (`block_private_hosts`,
  `max_bytes`, `timeout_ms`) stay in config.
- **SSRF (business posture).** Businesses read *internal* systems, so
  private/loopback/link-local hosts are **allowed by default** — but the
  cloud-metadata endpoints (`169.254.169.254`, `100.100.100.200`, and their
  hostnames) are **always blocked**, and a config flag
  (`source.block_private_hosts`) lets an enterprise lock down further.
  Rationale: the main SSRF risk in an agent tool is prompt-injection
  driving a fetch to the metadata service, not the operator reading their
  own intranet.
- **Business-level sources are a roadmap, not this step.** The `source`
  layer is a pluggable connector seam; later connectors (databases, code
  hosts, SaaS, docs, search — see the MCP server taxonomy below) attach
  behind it. This step ships only `Read` (local file) + `Fetch` (HTTP(S)).
- **OpenAPI/pipeline endpoints are a future non-goal.** Reading an
  OpenAPI-described endpoint ("hit an API and get pipeline data") is
  explicitly out of scope now; the URL/scheme dispatch is designed so a
  connector (`openapi://`, MCP) can be added without reworking the seam.

## Business-level sources (roadmap)

What "business sources" means, grounded in the [MCP reference servers](https://github.com/modelcontextprotocol/servers)
(2026-09-09): the ecosystem that defines how AI agents connect to
enterprise tools. The canonical read-only business sources:

- **Code hosts** — GitHub, GitLab (repo, PR, issues, search).
- **Databases** — PostgreSQL, SQLite, Redis (schema inspection + read-only queries).
- **Documents / knowledge** — Google Drive, Confluence/Notion (file access + search).
- **SaaS / collaboration** — Slack (channels/messages), Sentry (issues), Jira (tickets).
- **Search / web** — Brave Search, web fetch, browser (Puppeteer).
- **Cloud / storage** — AWS Knowledge Base (Bedrock), S3/GCS.
- **Memory** — knowledge-graph persistent memory.

**Design implication:** `source.Reader` is deliberately minimal; the
long-term shape is a **connector registry keyed by scheme/type** (`file`,
`http`, then `postgres://`, `github://`, `mcp://`, `openapi://`). MCP is the
obvious interop target (there is a Go MCP SDK), so business connectors can
be consumed as MCP servers rather than re-implemented in Harness.
