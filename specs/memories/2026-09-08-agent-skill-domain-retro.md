# Retro — Agent & Skill Domain Model (2026-09-08)

> Cross-references `2026-09-08-session-log.md`.

## What was done

Agent/Skill domain types + ports, a SQLite adapter with FTS5 search,
shortened base62 UUID ids, CLI CRUD/search, and `/agents` + `/skills`
dashboard pages (`feat/agent-skill-domain`, PR #3).

## Learnings → rules

### 1. A combined Store interface needs entity-prefixed methods

**What happened:** the initial `AgentStore`/`SkillStore` interfaces used
`Create`/`Get`/`List`/`Delete`/`Search`; embedding both into one `Store`
interface would have collided on those names.

**Rule:** when a combined interface embeds multiple ports, entity-prefix
the methods (`CreateAgent`/`CreateSkill`). Recorded in the spec's
Implementation Notes.

### 2. FTS5 trigram cannot match below 3 characters

**What happened:** trigram tokenizes into 3-grams, so a 1–2 char query
matches nothing. Search therefore falls back to `LIKE` for short queries.

**Rule:** trigram-based search needs a short-query fallback. Pinned by
`TestAgentSearchFTSAndLike` and documented in the spec.

## Rule updates made

- `specs/2026-09-08-agent-skill-domain.md` — Implementation Notes
  (method prefixing, integer-PK + uuid schema, trigram < 3 fallback).
