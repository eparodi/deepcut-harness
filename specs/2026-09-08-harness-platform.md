# Harness — Platform (Draft)

**Feature slug:** `harness-platform`
**Status:** Draft
**Owner:** PM
**Created:** 2026-09-08

## Requirements

Harness is the **local-first AI engineering team for your repositories**.
A user defines an **Agent** (the brain — a persona/goal) and equips it
with **Skills** (the hands — a runtime + allowed commands). Agents run
on the host OS, inside the user's workspace, using the user's own LLM
API keys. Zero source-code egress: only the LLM prompt leaves the
machine.

### User stories

- As a developer, I can run `harness init` and get a working local
  setup (this first milestone: a localhost dashboard + CLI).
- As a developer, I can define an Agent (name + goal + model) and attach
  Skills to it.
- As a developer, I can run an Agent against a workspace and stream its
  "thinking" + terminal actions, approving destructive steps.
- As a developer, I can search past runs and the repo's persistent
  memory.

### Acceptance criteria (scaffold milestone — this change)

- `make build`, `make test`, `make vet` pass with zero external deps.
- `harness` (no args) serves the dashboard on `127.0.0.1:8787`.
- The dashboard renders via the go-htmx Rule A/B contract (full document
  vs swap-region partial), pinned by render tests.
- `harness version` and `harness <unknown>` behave per the dispatch
  table.

## Explicit Non-Goals (for now)

- No cloud/remote execution, multi-tenancy, or team RBAC.
- No auth layer (single-user, loopback-only).
- No Skill marketplace, vector memory, or Postgres — those land as
  follow-up features.
- No new third-party Go dependencies.

## Design

- **Single binary**, `main.go` → `internal/cli.Main` (dispatch table).
- **`internal/dashboard`** serves the go-htmx UI: embedded templates,
  folder-per-component registry, build-time-fetched + sha256-pinned
  htmx (`tools/fetchhtmx`).
- **`internal/config`** loads `config.json` strictly (unknown fields
  rejected), defaults synthesized in `Validate()`.
- Follow-up packages (not yet built): `internal/llm` (provider client,
  malformed-response ladder, retries/breaker, cost budget — `ai-engineer`
  skill), `internal/store` (Postgres via `db-analyst`), `internal/exec`
  (workspace-jail executor).

## Task Checklist

- [x] Git repo + remote + `main` branch
- [x] `go.mod`, `main.go`, CLI dispatch (`internal/cli`)
- [x] Config loader + strict decoding (`internal/config`)
- [x] go-htmx dashboard: server, render contract, templates, components,
      htmx fetch (`internal/dashboard`, `tools/fetchhtmx`)
- [x] Build/test/vet green
- [x] `AGENTS.md`, `HOW_WE_WORK.md`, `zed/profiles.json`, starter README
- [ ] Agent/Skill domain model (next feature)
- [ ] LLM provider layer (`internal/llm`)
- [ ] Workspace-jail executor (`internal/exec`)
- [ ] Postgres store + persistent memory (`internal/store`)

## Implementation Notes

- Skills are referenced from the shared [`deepcut-skills`](https://github.com/eparodi/deepcut-skills) repo (no per-repo
  copies yet).
