# Source Reading Layer (Draft)

**Feature slug:** `source-reading`
**Status:** Draft
**Owner:** PM
**Created:** 2026-09-08

## Requirements

### User stories

- As the wizard (or any agent), I can read a local file by path,
  constrained to a workspace root (no `..` escape).
- As the wizard, I can fetch a URL over HTTP(S), bounded by a size cap and
  a timeout.
- Fetched/read content is treated as **untrusted data** injected into the
  LLM prompt (never instructions).

### Acceptance criteria

- `Read(path)` refuses `..` escapes and paths outside the workspace root.
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

- [ ] `internal/source` — `Reader` interface + implementation
- [ ] Path-traversal guard (workspace root)
- [ ] Fetch with timeout + size cap + scheme allow-list
- [ ] Table-driven tests (happy, traversal, non-http, oversized, timeout)
- [ ] `make build` / `make vet` / `make test` green

## Decisions (resolved at the gate)

- **SSRF:** allow private/loopback IPs behind a config flag (default open — local-first).
