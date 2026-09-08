# Dashboard — Template & Page Contract

> Scoped rules for everything under `internal/dashboard/`. The root
> `AGENTS.md` holds the always-on repo rules; the `go-htmx` skill (in
> `skills-test`) holds the htmx-specific rules. This file documents the
> decisions that are specific to THIS repo's dashboard.

## Folder-per-view

Each view lives in its own folder, keyed by route, with **full
co-location**: the template, the handler, and any tests live together.

```
pages/<name>/<name>.html    {{define "content"}} — REQUIRED
pages/<name>/<name>.go      Page() descriptor + handler (REQUIRED)
pages/<name>/<name>_test.go optional
```

- `/` → `pages/summary/`, `/app` → `pages/app/`, `*` → `pages/error/`
  (catch-all 404).
- **Arbitrary nesting depth.** A nested route maps to a nested folder with
  no depth limit: `/agents/new` → `pages/agents/new/{new.html,new.go}`.

### Why per-page embedding

Each page package embeds its own template (`//go:embed <name>.html`) and
hands the text to the engine. This is deliberate: a central
`//go:embed pages/*/*.html` glob **cannot express arbitrary depth**
(Go's embed `*` never crosses `/`), so per-page embedding is what makes
any-depth nesting work. Do not switch back to a central glob.

## Adding a page

1. Create `pages/<name>/` with `<name>.html` (a `{{define "content"}}`
   block) and `<name>.go`.
2. In `<name>.go`, return a `page.Page` with `Path` (explicit, never
   inferred from the folder name), `Name` (the template name), `Template`
   (the embedded text), and a `Handler` that builds its data and calls
   `d.Render(w, r, name, data)` (or `d.RenderStatus` for a non-200 page).
3. Add the page to the `pages` slice in `pages.go` (the catch-all 404
   page stays last).

## Hard rules

- **The render contract lives in exactly one place** (`render.go`). Pages
  call `d.Render` / `d.RenderStatus`; they never branch on `HX-Request`
  themselves.
- **The shared layout is not duplicated.** `layout/{layout.html,
  style.html}` (`base` + `mainwrap`) is parsed into every page's set.
- **Every page template MUST define `content`.** The engine fails fast at
  startup if a page is missing its `{{define "content"}}` block (a silent
  empty page otherwise).
- **`page` is a leaf package.** It imports only `net/http` and `log/slog`,
  never `dashboard` — that's what keeps `dashboard → pages/<name> → page`
  acyclic.
- **Page data embeds `page.Base`** (or supplies the `.Title`/`.Addr`
  fields the layout reads) and adds page-specific fields on top.
- **Components are unchanged** — the folder-per-component contract in
  `components/` is a sibling convention, not replaced by this one.
