# Agent & Skill Domain Model (Approved)

**Feature slug:** `agent-skill-domain`
**Status:** Approved
**Owner:** PM
**Created:** 2026-09-08

## Requirements

### User stories

- As a developer, I can define an **Agent** (name + goal/prompt + model)
  and persist it locally.
- As a developer, I can define a **Skill** (name + description + runtime
  + allowed commands) and persist it locally.
- As a developer, I can **attach** a Skill to an Agent (and detach it).
- As a developer, I can list / get / remove agents and skills.
- As a developer, I can **search** agents and skills by text, ranked by
  relevance.
- As a developer, I can view and create agents and skills in the
  dashboard (`/agents`, `/skills`).

### Acceptance criteria

- `harness agent create|list|get|remove|attach|detach|search` work
  against a persistent local SQLite store.
- `harness skill create|list|get|remove|search` work against the same
  store.
- Search uses SQLite FTS5 (substring + BM25 relevance), not a naive scan.
- `/agents` and `/skills` render in the folder-per-view dashboard.
- Domain types + store interfaces are table-tested (happy path, each
  error path, edge cases), against an in-memory SQLite DB.
- IDs are UUIDs in a shortened (base62) form.

## Explicit Non-Goals

- No LLM execution — an Agent's prompt is not run yet (that's the
  `ai-engineer` layer, later).
- No command execution / workspace jail — a Skill's runtime and allowed
  commands are recorded, not enforced (that's the `exec` layer, later).
- No Postgres / Elasticsearch / JSON adapters this step (SQLite first;
  the interface is the seam that admits them later).
- No semantic/vector search (the product brief's "vector memory" is a
  separate concern, later).

## Design

### Data model (borrows the ecosystem's conventions)

```
Agent  { id, name, prompt, provider, model, temperature, skill_ids[],
         created_at, updated_at }
Skill  { id, name, description, category, runtime, allowed_commands[],
         created_at }
```

- `Agent.name` is a unique human-readable name; `Agent.prompt` is the
  system prompt (goal + tone); `provider`/`model` name the LLM backend
  (e.g. `deepseek` / `deepseek-v4-flash`); `temperature` is [0,1].
- `Skill.name` is a unique kebab-case slug; `description` is one line;
  `category` is `role` | `stack` | `process` (mirrors `AGENT_INDEX`);
  `runtime` is the execution env (e.g. `bash`); `allowed_commands` is
  the allow-list, enforced later.
- Attachment is `Agent.skill_ids` (ordered).

### ID scheme

UUID v4 entropy, shortened to a base62 string (~22 chars) via stdlib
`crypto/rand` — no third-party UUID dep.

### Interfaces (the extensibility contract = the port)

```go
type AgentStore interface {
	Create(Agent) (Agent, error)
	Get(id string) (Agent, error)
	List() ([]Agent, error)
	Update(Agent) (Agent, error)   // also the attach/detach path
	Delete(id string) error
	Search(query string) ([]Agent, error)
}

type SkillStore interface {
	Create(Skill) (Skill, error)
	Get(id string) (Skill, error)
	List() ([]Skill, error)
	Delete(id string) error
	Search(query string) ([]Skill, error)
}
```

### Storage (pluggable adapters)

The interfaces are the port; each backend is an adapter implementing the
same interface, selected by a `store.driver` config key (default
`sqlite`).

| Driver | Search | Deps / infra |
|---|---|---|
| `sqlite` (first) | FTS5 (trigram + BM25) | `modernc.org/sqlite` (pure Go) |
| `postgres` (later) | `tsvector` + `pg_trgm` | `pgx` + running Postgres |
| `elasticsearch` (later) | relevance | ES client + running ES |

### FTS5 search design

Two FTS5 virtual tables — `agents_fts` and `skills_fts` — with the
`trigram` tokenizer, covering name/description/prompt. `Search(query)`
runs `MATCH` ordered by `bm25()`, joins back to the primary tables, and
returns ranked results. Triggers keep the FTS tables in sync.

### Packages

```
internal/agent        // Agent type + AgentStore interface (port)
internal/skill        // Skill type + SkillStore interface (port)
internal/id           // shortened base62 UUID
internal/store        // adapter selection (store.driver) + Open/Close
internal/store/sqlite // SQLite + FTS5 adapter
internal/cli          // agent.go, skill.go — subcommands
internal/dashboard/pages/agents   // folder-per-view
internal/dashboard/pages/skills   // folder-per-view
```

## Task Checklist

- [ ] `internal/id` — shortened base62 UUID
- [ ] `internal/agent` — `Agent` type, validation, `AgentStore` interface
- [ ] `internal/skill` — `Skill` type, validation, `SkillStore` interface
- [ ] `internal/store/sqlite` — schema + FTS5 + migration, CRUD + `Search`
- [ ] `internal/config` — `store.driver` + `store.dsn`
- [ ] CLI: `agent create|list|get|remove|attach|detach|search`
- [ ] CLI: `skill create|list|get|remove|search`
- [ ] Dashboard: `/agents` + `/skills` (folder-per-view, read-only + create)
- [ ] Table-driven tests (in-memory SQLite) + render contract tests
- [ ] `make build` / `make vet` / `make test` green

## Decisions (resolved at the gate)

- **Storage:** SQLite first (`modernc.org/sqlite`, pure Go, FTS5).
- **IDs:** shortened base62 UUID.
- **Scope:** CLI + dashboard pages this step.
- **Dependency:** `modernc.org/sqlite` approved.

## Implementation Notes

- Interface methods are entity-prefixed (`CreateAgent`/`GetAgent`/… and
  `CreateSkill`/`GetSkill`/…) so the combined `Store` interface can embed
  both ports without name collisions (the sketch above used `Create`/`Get`).
- The SQLite schema uses an integer PRIMARY KEY + a UNIQUE `uuid` column
  (the public short-UUID id); external-content FTS5 tables join on the
  integer rowid and are kept in sync by triggers.
- Search uses FTS5 `MATCH` for queries ≥ 3 chars and a `LIKE` fallback for
  shorter queries (the trigram tokenizer cannot match below 3 chars).
