# Dashboard Folder-Per-View (Approved)

**Feature slug:** `dashboard-folder-per-view`
**Status:** Approved
**Owner:** PM
**Created:** 2026-09-08

## Requirements

- Each dashboard view lives in its own folder, keyed by route: the files
  that render `/app` live in `app/`, `/agents` in `agents/`, and so on.
- **Full co-location**: a view's template, handler, and tests live in the
  same folder.
- **Arbitrary nesting depth**: a nested route (`/agents/new`) maps to a
  nested folder (`pages/agents/new/`) with no depth limit.
- The go-htmx render contract (Rule A/B) is preserved unchanged: one
  render choke point, full document vs swap-region partial.
- The shared layout (`base` + `mainwrap`) is not duplicated per page.
- The template contract is documented in `internal/dashboard/AGENTS.md`.

### Acceptance criteria

- `make build`, `make test`, `make vet` pass.
- The `/` summary view and the 404 error view are served from
  `pages/summary/` and `pages/error/` respectively.
- A new `pages/app/` view at `/app` renders and is covered by a render
  contract test (proves the convention scales).
- The render contract test still pins full-document vs partial behavior,
  and the partial-is-byte-identical-to-region check still holds.
- `internal/dashboard/AGENTS.md` documents the page/template contract.

## Explicit Non-Goals

- No filesystem auto-routing that infers routes from folder names alone —
  the route is declared explicitly in the page's Go file.
- No change to the component registry or the htmx fetch pipeline.
- No auth, no new dependencies.

## Design

### Target structure

```
internal/dashboard/
├── dashboard.go           # Server, New, Start, Shutdown, middleware chain
├── render.go              # handler + THE render choke point (Rule A/B)
├── engine.go              # template engine + per-page template parsing
├── pages.go               # page registration + sanity check
├── AGENTS.md              # template contract (this design, documented)
├── layout/
│   ├── layout.html        # base + mainwrap (shared)
│   └── style.html         # shared <style> injection
├── page/
│   └── page.go            # Page descriptor + Deps + Base (leaf package)
├── pages/
│   ├── summary/           # route: GET / (index page)
│   │   ├── summary.html
│   │   ├── summary.go
│   ├── app/               # route: GET /app
│   │   ├── app.html
│   │   ├── app.go
│   └── error/             # route: catch-all 404
│       ├── error.html
│       └── error.go
├── components/            # unchanged (folder-per-component)
├── static.go / htmx.go / favicon.go / middleware.go
└── render_test.go         # table-driven render contract over all pages
```

Nested routes follow the same convention to any depth: `/agents/new` →
`pages/agents/new/{new.html,new.go}`.

### Page contract

A page folder REQUIRES `<name>.html` (a `{{define "content"}}` block).
The page's Go file embeds its own template and declares its route and
template name explicitly:

```go
// pages/summary/summary.go
package summary

import (
	_ "embed"
	"net/http"

	"deepcut-harness/internal/dashboard/page"
)

//go:embed summary.html
var tmpl string

func Page() page.Page {
	return page.Page{
		Path:     "/{$}",
		Name:     "summary",
		Template: tmpl,
		Handler: func(d page.Deps) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				d.Render(w, r, "summary", page.Base{Title: "Harness", Addr: d.Addr})
			}
		},
	}
}
```

Per-page embedding (rather than a central `//go:embed pages/*/*.html`
glob) is what makes arbitrary nesting depth work — `*` in a Go embed
glob never crosses `/`, so a central glob cannot express `pages/agents/
new/`. Each page package embeds its own local `*.html` instead.

### The leaf `page` package (breaks the import cycle)

```go
package page

type Base struct{ Title, Addr string }

type Deps struct {
	Addr         string
	Log          *slog.Logger
	Render       func(w http.ResponseWriter, r *http.Request, name string, data any)
	RenderStatus func(w http.ResponseWriter, r *http.Request, name string, data any, status int)
}

type Page struct {
	Path     string
	Name     string
	Template string
	Handler  func(Deps) http.HandlerFunc
}
```

`page` imports only `net/http` and `log/slog` (never `dashboard`), so page
subpackages can import it without a cycle: `dashboard → pages/<name> →
page`.

### Registration

`pages.go` imports the page packages, injects the render closure into
`page.Deps`, parses each page's template into the engine, and registers
its route. The catch-all 404 page registers last. A startup sanity check
fails fast if a page template is missing its `{{define "content"}}` block.

## Task Checklist

- [ ] Add `page/` descriptor package (`Page`, `Base`, `Deps`)
- [ ] Move `layout.html`/`style.html` into `layout/`; rewrite the engine
- [ ] Move the summary view into `pages/summary/` (template + handler)
- [ ] Move the error view into `pages/error/` (template + handler)
- [ ] Add a minimal `pages/app/` view at `/app`
- [ ] Add `pages.go` route registration + the content-define sanity check
- [ ] Add `internal/dashboard/AGENTS.md` (page/template contract)
- [ ] Update `README.md` project structure
- [ ] `make build` / `make vet` / `make test` green

## Implementation Notes

- The `handler` struct + render choke point stay in `package dashboard`
  (single `HX-Request` branch, per the go-htmx skill).
- Page handlers are free functions closing over `page.Deps`, not methods on
  the dashboard `handler` — that is what makes the `pages/<name>/`
  subpackage co-location possible.
- `/` maps to `summary/` (the index page) by convention.
