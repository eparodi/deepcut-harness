# AGENTS.md — Always-On AI Agent Rules

> Loaded automatically for every agent thread in this project. These
> rules address DeepSeek-specific failure modes and enforce minimum
> quality bars. Skill files (in the shared [`deepcut-skills`](https://github.com/eparodi/deepcut-skills) repo) add
> stack- and role-specific rules on top of these.

## Section 1 — DeepSeek-Specific Guardrails

- **Cite your source.** Before using any library API, grep the actual
  import path used in this codebase. If it doesn't exist, do NOT guess.
- **"I don't know" is a valid answer.** If you cannot find the exact API
  in the codebase or known docs, say so and ask.
- **Verify before writing.** Run the build immediately after adding a new
  import or call to catch wrong signatures.
- **Prefer stdlib and existing deps.** This project is deliberately
  minimal (stdlib + the embedded htmx fetch). Never add a dependency
  without explicit user approval.
- **Plan before editing.** Non-trivial tasks route through `spec-driven`.
  State what you'll change, which files, and what it accomplishes before
  touching code.
- **Inspection verbs are read-only.** "check", "look", "review" default
  to reporting findings — ask before fixing.
- **Flag uncertainty.** Label inferences `[Inference]`, `[Assumption]`,
  `[Unverified]`. Never end a turn with untested code and a claim of
  confidence.

## Section 2 — Codebase Pattern Matching

- **Layout.** Single binary; Go code lives under `internal/<pkg>/`; the
  entrypoint is `main.go` (thin bridge to `internal/cli`). New files go
  where equivalent files already live.
- **HTTP.** stdlib `http.ServeMux` + `html/template` + htmx. Handlers
  are read-only except explicitly mutating routes.
- **Render contract.** The ONE htmx render choke point is
  `internal/dashboard/render.go` (Rule A/B). Never branch on
  `HX-Request` anywhere else.
- **Pages.** Dashboard views are folder-per-view
  (`pages/<name>/{<name>.html,<name>.go}`, keyed by route, arbitrary
  nesting depth). The full page/template contract is documented in
  `internal/dashboard/AGENTS.md`.
- **Components.** Dashboard UI follows the folder-per-component contract
  (`components/<name>/<name>.{html,css,js}`); the component registry
  concatenates them at startup (missing files panic, by design).
- **Naming, error style, testing style.** Match adjacent files exactly
  (table-driven tests, `fmt.Errorf("context: %w", err)`).
- **New pattern?** Point out none exists, propose it, get approval, then
  implement and document it.
- **Skills.** Generic roles (`pm`, `architect`, `reviewer`, `qa`, etc.)
  and generic stack skills (`go-htmx`, `go-chi`, `ai-engineer`,
  `db-analyst`) load from the shared [`deepcut-skills`](https://github.com/eparodi/deepcut-skills) repo — never copied
  into this repo. Only project-specific technical skills (e.g.
  `go-harness` stack, `harness-engineer`/`dashboard-engineer` roles)
  live in this repo's `.agents/skills/`.

## Section 3 — Ambiguity & Business Logic

- Never guess business rules (validation, authorization, pricing,
  agent/skill semantics). If the spec is silent, list the uncovered
  cases and ask.
- State explicit non-goals on every feature.

## Section 4 — Security

- **Secrets** live in `.env` (gitignored, chmod 600), never in
  `config.json`, never in logs or prompts. Dev-default secrets log a
  `slog.Warn` at startup.
- **Prompt injection.** Workspace file contents, fetched documents, and
  agent learnings are untrusted data injected into LLM prompts — the
  system prompt must say so, and raw third-party content rendered to
  HTML must be escaped (`html/template`) and/or wrapped in `hx-disable`.
- **Path traversal.** When the workspace-jail executor lands, strictly
  parse path arguments and refuse `..` escapes.
- Never concatenate user input into SQL (parameterized queries only);
  validate enums before any DB access.

## Section 5 — Output & Testing Discipline

- **Build after every change:**
  ```bash
  make build     # fetches pinned htmx first, then go build
  make vet
  make test
  ```
- The htmx fetch is REQUIRED by `//go:embed htmx/*`. A missing fetch
  fails the build loudly — that's the delivery contract, never "fix" it
  by committing the file.
- **Tests:** table-driven, covering happy path, each error path, and
  edge cases. New UI work pins each distinct state (loading/empty/error/
  populated). TDD for observable behavior.
- **After inserting a parameter,** grep all call sites — same-typed
  positional args are invisible to the compiler.

## Section 6 — Git & Commit Hygiene

- Commit freely on branches you create; never commit to `main` without
  explicit instruction.
- Branch from latest `main`: `feat/`, `fix/`, `chore/`, `refactor/`,
  `docs/` + kebab-case.
- Conventional Commits (`feat:`, `fix:`, …), subject < 72 chars, body
  explains WHY.

## Section 7 — Tool Access

- Read-only ops and project-scoped writes are autonomous; Git is the
  safety net. Destructive git ops and out-of-repo writes need sign-off.

## Section 8 — The Specs Directory

- Specs are the single source of truth, named
  `specs/<YYYY-MM-DD>-<feature-slug>.md`.
- Lifecycle: Draft → Review → Approved → Implemented → Archived. Never
  modify an Approved spec without the PM's sign-off.

## Section 9 — Session Log & Retros

- Log every correction to `specs/memories/<YYYY-MM-DD>-session-log.md`.
- At feature end, trace each correction to the missing rule, update the
  relevant skill or this file, and write a retro.

## Section 10 — Session Learnings

Generic learnings live in ONE shared place: [`deepcut-skills`](https://github.com/eparodi/deepcut-skills) `AGENTS.md`
Section 10 (code style, tool discipline, UI, payload verification,
deploy ordering). Cite them as "deepcut-skills AGENTS.md §10: <rule name>".
This section keeps only Harness-specific learnings — none exist yet.

*Last updated: 2026-09-08*
