# Project Technical Skills (Draft)

**Feature slug:** `project-skills`
**Status:** Approved
**Owner:** PM
**Created:** 2026-09-08

## Requirements

- Author the three project-specific technical skills that live in THIS
  repo (`.agents/skills/`): `go-harness` (stack), `harness-engineer`
  (role), `dashboard-engineer` (role).
- Wire them into `zed/profiles.json` and the `HOW_WE_WORK.md` /
  `README.md` role tables.
- Generic skills (`go-htmx`, `go-chi`, `ai-engineer`, `db-analyst`, and
  the process roles) continue to load from `deepcut-skills` — never
  copied into this repo.

## Explicit Non-Goals

- Do NOT copy generic skills into the repo.
- Do NOT author skills for packages that do not exist yet (`internal/llm`,
  `internal/store`, `internal/exec`) — the skills document only what is
  true today, with pointers to the `ai-engineer`/`db-analyst` skills for
  when those layers land.

## Design

### `go-harness` (stack skill)

Project-specific Go standards: single-binary layout, the `internal/cli`
dispatch table, the dashboard's folder-per-view + folder-per-component
contracts, the leaf `page` descriptor package, the strict config loader,
slog, and testing. Points to `go-htmx` (htmx rules) and `go-chi`
(generic layering/errors/DB) rather than duplicating them.

### `harness-engineer` (role skill)

Owns all Go implementation in `internal/`. Implements specs one task at a
time, test-first; never changes contracts unilaterally. Mirrors
`bot-engineer` (without the trading domain).

### `dashboard-engineer` (role skill)

Owns the dashboard frontend: `html/template` pages (folder-per-view),
the folder-per-component registry, and the vanilla-JS enhancement layer.
Mirrors `ui-engineer` (without the money/chart rules).

## Task Checklist

- [ ] `.agents/skills/go-harness/SKILL.md`
- [ ] `.agents/skills/harness-engineer/SKILL.md`
- [ ] `.agents/skills/dashboard-engineer/SKILL.md`
- [ ] Wire into `zed/profiles.json` (add `harness-engineer` + `dashboard-engineer`; replace `go-chi` with `go-harness` in the implementation profiles)
- [ ] Update `HOW_WE_WORK.md` role table
- [ ] Update `README.md` Skills section
- [ ] YAML-validate every `SKILL.md` frontmatter; JSON-validate `profiles.json`
