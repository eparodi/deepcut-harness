---
name: go-harness
description: "Go standards for the Harness repo — single-binary layout, the internal/cli dispatch table, the dashboard's folder-per-view + folder-per-component contracts, the leaf page descriptor package, the strict config loader, slog, and testing. Load when writing Go code in this repo."
---

# Go Harness Standards

The Harness repo is a single Go binary: a CLI dispatch table plus an
embedded `go-htmx` dashboard. This skill documents what is SPECIFIC to
this repo. For generic rules, load the shared skills from `deepcut-skills`
— `go-htmx` (htmx attributes/config/history), `go-chi` (layering, error
handling, testing, DB patterns — ignore the chi-router sections, Harness
uses stdlib `http.ServeMux`), `ai-engineer` (the LLM layer, when it
lands), `db-analyst` (Postgres, when it lands).

## Project Layout

```
deepcut-harness/
├── main.go                    # thin entrypoint → internal/cli.Main
├── internal/
│   ├── cli/                   # subcommand dispatch + default run loop
│   ├── config/                # strict config.json loader
│   └── dashboard/             # go-htmx HTTP dashboard
│       ├── dashboard.go       # Server + middleware chain
│       ├── render.go          # handler + render contract (Rule A/B)
│       ├── engine.go          # template engine (per-page parsing)
│       ├── pages.go           # page registration
│       ├── layout/            # shared layout (base + mainwrap)
│       ├── page/              # page descriptor (leaf package)
│       ├── pages/             # folder-per-view (summary/, app/, error/)
│       └── components/        # folder-per-component
├── tools/fetchhtmx/           # pinned htmx fetch (sha256-verified)
├── specs/                     # single source of truth
└── zed/profiles.json          # agent role profiles
```

Single binary; Go code lives under `internal/<pkg>/`. `main.go` is a thin
bridge: `os.Exit(cli.Main(os.Args[1:], os.Stdout, os.Stderr))`. New files
go where equivalent files already live.

## CLI Dispatch Table

Subcommands live in `internal/cli/` behind a dispatch table
(`cli.go`). A new subcommand is ONE file + ONE entry in the `commands`
slice; the usage text renders from the table so it can never drift.

- Handler signature is uniform: `func(args []string, stdout io.Writer) error`.
- Commands needing injected deps (a logger, stdin) adapt inside their
  table entry, keeping the table uniform.
- The no-args default runs the dashboard (`run()`).
- `Main` returns 0 on success, 1 on command/loop failure, 2 on an unknown
  subcommand (which prints usage, never falls through to the loop).

## Dashboard (folder-per-view + folder-per-component)

Two sibling conventions, both documented in `internal/dashboard/AGENTS.md`:

- **Folder-per-view**: `pages/<name>/{<name>.html, <name>.go}` co-locates
  a view's template and handler, keyed by route, to any nesting depth.
  Each page package embeds its own template (`//go:embed <name>.html`).
- **Folder-per-component**: `components/<name>/{<name>.html, <name>.css,
  <name>.js}`; the registry concatenates CSS/JS at startup.

Hard rules (full contract in `internal/dashboard/AGENTS.md`):

- The render contract lives in exactly ONE place (`render.go`). Pages call
  `page.Deps.Render`/`RenderStatus`; they never branch on `HX-Request`.
- The shared layout (`layout/`) is never duplicated.
- Every page template MUST define `content` (the engine fails fast at
  startup otherwise).
- `page` is a leaf package: it imports only `net/http` and `log/slog`,
  never `dashboard` — that keeps `dashboard → pages/<name> → page`
  acyclic. Never add a `dashboard` import to `page/` or `pages/*`.

## Config

`internal/config/config.go`:

- `Config` struct + `Default()` + `Load(path)` + `Validate()`.
- Strict decoding (`json.Decoder` + `DisallowUnknownFields`) so a typo'd
  key fails loudly, never silently.
- `Validate()` MUTATES the receiver to synthesize defaults (empty
  `listen_addr` → `127.0.0.1:8787`). Validate the SAME instance the
  runtime reads — a validated copy is a config the runtime never received.
- `config.json.example` must load under the strict loader (pinned by
  `TestLoadExample`).

## Logging & Error Handling

- `log/slog`. `run()` creates a `slog.NewTextHandler` at Info; handlers
  log through the injected logger (`h.log`, `h.errLog`).
- Wrap errors with `fmt.Errorf("context: %w", err)`; no `errors.Wrap`.

## Testing

- `make build`, `make test`, `make vet` — all run `tools/fetchhtmx` first
  (the `//go:embed htmx/*` directive requires the fetched file; a missing
  fetch fails the build loudly, by design — never commit the file).
- Table-driven tests: happy path, each error path, edge cases.
- Dashboard work pins the render contract: full document vs swap-region
  partial per page, byte-identity of the swap region, and the static
  asset routes (`render_test.go`).
- After inserting a parameter, grep all call sites — same-typed
  positional args are invisible to the compiler.

## Common Traps

- **`go:embed` `*` never crosses `/`.** A central `//go:embed
  pages/*/*.html` glob cannot express nested routes — that is why each
  page embeds its own template.
- **Import cycle via `page`.** Do not import `dashboard` from `pages/*`
  or `page`; the leaf `page` package is the only shared dependency.
- **Double `HX-Request` branch.** A second branch site drifts from
  `render.go`; page handlers must go through `Deps.Render`.
- **Validate-then-read config.** A validated COPY of a config leaves the
  runtime's instance without synthesized defaults.
- **`content` define is mandatory.** A page template missing its
  `{{define "content"}}` renders silently empty unless the engine's
  startup check catches it.
