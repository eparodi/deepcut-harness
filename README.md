# Harness

The **local-first AI engineering team for your repositories**.

Harness transforms a local development folder into an autonomous AI
workforce: define an **Agent** (the brain — e.g. a Senior DevOps or QA
Lead) and equip it with **Skills** (the hands — e.g. Bash, Python,
Terraform). Agents run directly on your machine, inside your actual
codebase, using your own API keys (DeepSeek/OpenAI). They read your
files, run your tests, fix your bugs, and provision your
infrastructure — without uploading your proprietary code to the cloud.

This is the **scaffold**: a single Go binary with an embedded
`go-htmx` dashboard, a CLI dispatch table, and the project conventions
in place. The Agent/Skill engine lands next.

## Running

```bash
make run            # starts the dashboard on 127.0.0.1:8787
# or
go run .            # same thing (default subcommand)
```

Open <http://127.0.0.1:8787>. `harness version` prints the build
version.

`make build` / `make test` / `make vet` fetch the pinned htmx release
first (sha256-verified, `tools/fetchhtmx`) — a bare `go build` on a
fresh clone fails loudly until `make htmx` runs, by design (no CDN at
runtime, no committed library copy).

## Project Structure

```
deepcut-harness/
├── main.go                          # thin entrypoint → internal/cli
├── internal/
│   ├── cli/                         # subcommand dispatch + default run loop
│   ├── config/                      # strict config.json loader
│   └── dashboard/                   # go-htmx HTTP dashboard
│       ├── server.go                # routes + middleware chain
│       ├── handlers.go              # render contract (Rule A/B)
│       ├── templates.go             # page registry + template engine
│       ├── templates/               # layout + pages
│       └── components/              # folder-per-component registry
├── tools/fetchhtmx/                 # pinned htmx fetch (sha256-verified)
├── specs/                           # single source of truth
├── zed/profiles.json                # agent role profiles
├── AGENTS.md / HOW_WE_WORK.md       # always-on rules + workflow
└── config.json.example              # operator config template
```

## Architecture (mirrors `deepcut-binance-bot`)

- **Single binary.** `main.go` bridges `os.Args` to `internal/cli.Main`.
- **CLI dispatch table.** Subcommands register once; usage text renders
  from the table so it can't drift. The no-args default runs the
  dashboard.
- **`go-htmx` partial-swap contract.** `handlers.go` is the one render
  choke point: `HX-Request: true` returns only the swap region, anything
  else returns the full document. htmx is embedded (build-time fetch).
- **Folder-per-component registry.** `components/<name>/{<name>.html,
  <name>.css, <name>.js}` concatenated into single bundles at startup.

## Skills

The shared skills live in the **`skills-test` repo** (the canonical hub)
— this repo references them rather than carrying per-repo copies yet.
Roles map to skills in `zed/profiles.json`. The relevant stack skills:

- `go-htmx` — htmx v4 usage for `html/template` apps (primary stack skill)
- `go-chi` — generic Go backend standards (layering, errors, testing,
  DB patterns); ignore the chi-router-specific sections — Harness uses
  stdlib `http.ServeMux`
- `ai-engineer` — the LLM layer (provider client, malformed-response
  ladder, retries/breaker, cost budget)
- `db-analyst` — Postgres migrations and the store/query layer

Generic roles (`pm`, `architect`, `ux-designer`, `backend-engineer`,
`reviewer`, `qa`, `security-engineer`, `orchestrator`, `spec-driven`)
load directly from `skills-test`.
