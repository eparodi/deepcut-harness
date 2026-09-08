# HOW WE WORK — Multi-Role AI Development (Harness)

## The Setup

You run up to six agent threads simultaneously in Zed, each with a
different role skill loaded from the **shared skills repo**
(`skills-test/.agents/skills/`). This repo does not carry per-repo skill
copies yet.

| Thread | Skill | Can Write? | Can Terminal? | Model (default) | Model (heavy) |
|--------|-------|-----------|--------------|-----------------|---------------|
| PM | `pm` | specs only | ❌ | Flash | Pro |
| Architect | `architect` | specs only | ❌ | Pro | — |
| Backend Engineer | `backend-engineer` + `go-htmx` + `go-chi` | all Go code | ✅ | Flash | Pro |
| AI Engineer | `ai-engineer` + `go-chi` | the LLM layer (`internal/llm/`) | ✅ (llm scope) | Flash | Pro |
| DB Analyst | `db-analyst` + `go-chi` | Postgres migrations + store | ✅ | Pro | — |
| UX Designer | `ux-designer` | design artifacts | ❌ | Flash | Pro |
| Reviewer | `reviewer` | ❌ | ✅ | Flash | Pro |
| QA | `qa` | ❌ | ✅ | Flash | — |
| Security Eng | `security-engineer` | ❌ | ✅ | Pro | — |

Flash = `deepseek-v4-flash`, Pro = `deepseek-v4-pro`. The canonical
routing policy (task-class table, escalation ladder, handoff template)
lives in `skills-test/HOW_WE_WORK.md` and skills-test AGENTS.md §10.20.

`go-chi` is loaded for generic Go backend standards (layering, errors,
testing, DB) — its chi-router sections don't apply: Harness uses stdlib
`http.ServeMux`. `go-htmx` is the dashboard's primary stack skill.

---

## Single-Thread Orchestrator Mode (Alternative)

When the spec is already approved or the task is a well-scoped bugfix:

```
@orchestrator I need to [goal]. Go until PLAN.md is fully checked off.
```

The `orchestrator` skill turns one agent into a simulated 4-role team
(PLANNER / CODER / REVIEWER / DEBUGGER) that loops until every
`PLAN.md` line is `[X]`.

---

## The Specs Directory

`specs/` is the single source of truth. New files are date-prefixed:
`specs/<YYYY-MM-DD>-<feature-slug>.md` (chronological ordering; retros
and session logs use their session date; `README.md` is the exception).

A spec contains, in order: Metadata, Requirements, Explicit Non-Goals,
Design, Task Checklist, Implementation Notes.

## Feature Lifecycle (Gated Phases)

1. **Requirements** (PM) → Review Gate
2. **Design** (Architect) → Review Gate
3. **Task Breakdown** (PM)
4. **Implementation** (Backend Engineer, one task at a time, test-first)
   → QA + Security audit in parallel at the end

## Handoff Protocol

- **PM → Architect**: "Requirements approved in `specs/<feature>.md`."
- **Architect → Engineers**: "Contracts stable — build against them."
- **Any role → PM** (ambiguity): add `[NEEDS CLARIFICATION: …]`, tag PM,
  do NOT guess.

## Model Selection

Flash for ~80% of reversible work; escalate to Pro for interface design,
risk/security, and when Flash hallucinates Go APIs.
