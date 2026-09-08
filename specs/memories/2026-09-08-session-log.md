# Session Log — 2026-09-08

> Running record of corrections and user feedback for the Harness
> scaffold + folder-per-view session.

| # | What happened | Root cause | Fix / resolution |
|---|---------------|-----------|------------------|
| 1 | `copy_path` of skills into the repo was denied by the user | I defaulted to the per-repo-copy convention before the repo's skill policy was stated | User clarified: generic skills stay in `deepcut-skills`; only project-specific **technical** skills live in the repo. Documented in root `AGENTS.md` §2. |
| 2 | User wanted the dashboard views organized in folders ("the `/app` view lives in `app/`") | Initial scaffold used a flat `templates/` dir + monolithic `handlers.go` | Adopted folder-per-view with full co-location; spec + `internal/dashboard/AGENTS.md`. |
| 3 | "Full co-location" + "any depth" | Folder nesting must not be capped | Per-page `//go:embed <name>.html` (a central `pages/*/*.html` glob can't cross `/`); leaf `page` descriptor package breaks the import cycle. |

## Follow-ups / open questions

- [ ] Author the project-specific technical skills (`go-harness`, `harness-engineer`, `dashboard-engineer`) and wire them into `zed/profiles.json`.
- [ ] Landing page `/` is named `summary/` — confirm or rename to `home/`.
- [ ] `/app` is an exact-match route; decide whether `/app/` (trailing slash) should also resolve.
- [ ] Decide the review model: separate reviewer/qa/security threads (multi-role) vs. self-review + orchestrator.
