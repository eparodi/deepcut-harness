# Agent & Skill Wizard (Draft)

**Feature slug:** `agent-skill-wizard`
**Status:** Draft
**Owner:** PM
**Created:** 2026-09-08

## Requirements

### User stories

- As a developer, I can create an **Agent** or a **Skill** through an
  LLM-driven wizard that asks me questions and fills in the fields.
- The wizard works in the **CLI** (interactive) and the **dashboard**
  (a chat form).
- The wizard can **read sources** I point at — local files/repos and
  URLs — to inform its questions and the resulting definition.
- The wizard produces a validated `Agent`/`Skill` and persists it via the
  existing store.

### Acceptance criteria

- `harness wizard agent` and `harness wizard skill` run an interactive
  Q&A (with optional source references) and create the record.
- A dashboard chat form drives the same wizard and creates the record.
- The wizard's LLM responses follow a strict JSON protocol (ask vs done),
  enforced by the malformed-response ladder.
- Source content is injected into the prompt as **data**, never as
  instructions.
- `make build` / `make vet` / `make test` green.

## Non-Goals

- No command execution (the source layer is read-only; the full
  workspace-jail executor is later).
- No `git clone` of remote repos — "repos" means reading files in a local
  workspace directory this step.
- No streaming, no function/tool calling.

## Design

### The wizard protocol

The wizard is a loop over `llm.Provider`. A system prompt describes the
target (`agent` or `skill`) and its fields, and the LLM replies with JSON:

```json
{ "action": "ask", "question": "What should this agent optimize for?" }
```

or

```json
{ "action": "done", "agent": { "name": "...", "prompt": "...", "provider": "...", "model": "...", "temperature": 0.2 } }
```

On `done`, the wizard validates the definition and calls
`CreateAgent`/`CreateSkill`.

### Source reading

The user can name a source (a local path or a URL); the wizard reads it
via `source.Reader` (`Read`/`Fetch`) and feeds the content into the prompt
as DATA. Fetched/read content is bounded, guarded against traversal, and
never treated as instructions (AGENTS.md §4).

### Packages

```
internal/wizard            // the ask/done loop + prompt construction
internal/source            // read local files + fetch URLs (source-reading spec)
internal/cli/wizard.go     // interactive CLI prompt loop
internal/dashboard/pages/agents (or /wizard)  // the chat form
```

### CLI

`harness wizard agent` / `harness wizard skill` reads user answers from
stdin (and optional `--source <path|url>`), sends them to the provider,
and stops when the LLM emits `done`.

### Dashboard

A chat form POSTs each answer and re-renders the conversation; on `done`
it creates the record and links to `/agents` or `/skills`.

## Task Checklist

- [ ] `internal/source` — `Reader` (read file + fetch URL) with guards
- [ ] `internal/wizard` — ask/done loop + prompt construction + JSON protocol
- [ ] Wizard marks source content as data in the prompt
- [ ] Wizard validates the emitted definition before persisting
- [ ] CLI: `wizard agent` / `wizard skill` (with `--source`)
- [ ] Dashboard: chat form (folder-per-view) + create on `done`
- [ ] Table-driven tests (fake provider: ask→done, malformed, budget veto)
- [ ] `make build` / `make vet` / `make test` green

## Decisions (resolved at the gate)

- **Shape:** adaptive `ask`/`done` (the LLM decides the next question).
- **Sources:** external files/repos + the internet, via the read-only
  source layer (`specs/2026-09-08-source-reading.md`).
