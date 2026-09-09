# Config Admin UI (Draft)

**Feature slug:** `config-admin-ui`
**Status:** Approved
**Owner:** PM
**Created:** 2026-09-08

## Requirements

### User stories

- As a developer, I can view the current config (dashboard, store, llm,
  providers) on a `/settings` page.
- As a developer, I can add / remove / edit **providers** (name, base_url,
  api_key_env, and the key value).
- As a developer, I can edit the **LLM** reliability knobs (retries,
  breaker, budget).
- As a developer, I can edit **store** (driver/dsn) and **dashboard**
  (listen_addr).
- Changes persist to `config.json` and are validated by the strict loader.

### Acceptance criteria

- `/settings` renders each config section from the live config.
- Editing a section POSTs back, re-validates, and persists atomically.
- LLM provider changes hot-reload immediately; store/dashboard changes
  note that a restart is required.
- Invalid input re-renders with the error-summary component (no partial
  write, no crash).
- `make build` / `make vet` / `make test` green.

## Non-Goals

- No auth/RBAC (single-user, loopback — a future step).
- No key encryption-at-rest (macOS Keychain) yet.

## Design

### Page

`pages/settings/` (folder-per-view): a read view of the current config +
per-section forms (providers, llm, store, dashboard).

### Config editor seam

A `config` store that loads `config.json`, mutates the struct, runs
`Validate()`, and writes back atomically (temp file + rename). The
dashboard depends on this seam, not on the raw file.

### API keys (both)

The providers form edits BOTH `api_key_env` (the env-var name) and the
key VALUE. The key value is written to the gitignored `.env` (chmod 600)
under the named variable; it never lands in `config.json` or the
rendered page.

### Live reload

A mutex-guarded runtime config holder (mirror of the sibling repo's
`config.Runtime`) lets `/settings` swap the active config. On save, the
LLM registry and its knobs hot-reload; store/dashboard changes are
persisted but take effect on restart (noted in the UI).

## Task Checklist

- [ ] `pages/settings/` folder-per-view page (read view + forms)
- [ ] Config editor seam (load → mutate → validate → atomic write)
- [ ] `.env` key writer (write key under `api_key_env`, chmod 600)
- [ ] Providers form (add/remove/edit name, base_url, api_key_env, key)
- [ ] LLM form (retries, breaker, budget knobs)
- [ ] Store + dashboard form (driver, dsn, listen_addr)
- [ ] Runtime config holder + LLM hot-reload on save
- [ ] Error-summary validation UX
- [ ] Render + handler tests
- [ ] `make build` / `make vet` / `make test` green

## Decisions (resolved at the gate)

- **API keys:** both — edit `api_key_env` AND the key value (written to `.env`).
- **Live reload:** apply immediately (LLM registry hot-reloads; store/dashboard note restart).
