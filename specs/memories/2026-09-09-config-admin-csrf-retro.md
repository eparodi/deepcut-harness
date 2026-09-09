# Retro — Config Admin UI + CSRF (2026-09-09)

> Cross-references `2026-09-09-session-log.md`.

## What was done

PR #5 delivered the `/settings` config-admin UI (live reload), the
Apple-style form restyle, and the end-to-end wiring of the remaining LLM
knobs (including making `budget_warn_fraction` actually consume the daily
budget). A review surfaced six findings; all were fixed:

- Deep-copy `Providers` before mutating (shallow-copy bug).
- CSRF protection on the mutating `/settings` forms.
- `base_url` and `api_key_env` validation.
- Loud (non-silent) numeric parsing.
- Accurate save banner.

## Correction / learning → rule

### 1. Struct value copies still share reference-type fields (generic)

`Runtime.Get()` returns `Config` by value, but `Config.Providers` is a
`map`, so the copy aliases the live config's map — `apply` mutated the
runtime before validation/save and could race readers.

→ **skills-test §10.68.**

### 2. The loopback dashboard needs CSRF (Harness-specific)

A malicious page can POST to `127.0.0.1:8787/settings` and mutate config /
write `.env` keys. Harness is single-user with no sessions, so the guard
is a per-process `crypto/rand` synchronizer token.

→ **Harness §10.1.**

### 3. `write_file`/`edit_file` mangle HTML templates (generic)

Writing `settings.html` through `write_file` injected stray closing tags
(`</h1>`, `</code>`, `</li>`) into the template — an HTML auto-close pass.
Fixed by rewriting via a quoted heredoc and re-reading.

→ **skills-test §10.67.**

### Re-affirmed (no new rule)

- **A config knob nobody reads silently does nothing** — `budget_warn_fraction`
  was defined + synthesized but never consumed (bidirectional-verification,
  already in AGENTS.md §2.1 / shared §10.24).
- **A displayed claim must match what actually applies** — the "LLM applies
  immediately" banner overstated a layer with no runtime consumer (shared
  §10.45 / §10.32).
- **Malformed input must fail loudly** — `atoi`/`atof` silent-0 replaced
  with loud parse errors (shared §10.46).

## Rule updates made

- `AGENTS.md` §10 — added Harness-specific §10.1 (CSRF).
- `skills-test` (shared hub) `AGENTS.md` §10 — added §10.67, §10.68.
