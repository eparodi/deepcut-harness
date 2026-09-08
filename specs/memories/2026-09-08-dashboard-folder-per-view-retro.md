# Retro — Dashboard Folder-Per-View (2026-09-08)

> Trace each session correction to the rule that was missing, and record
> what changed. Cross-references `2026-09-08-session-log.md`.

## 1. Skills live in the repo only when project-specific and technical

**Correction:** I tried to copy generic skills into the repo; the user
denied it and clarified the policy.

**Missing rule:** no explicit rule said where skills live for this repo.

**Fix:** added to root `AGENTS.md` §2 — generic roles/stack skills load
from [`deepcut-skills`](https://github.com/eparodi/deepcut-skills); only project-specific technical skills
(`go-harness`, `harness-engineer`, `dashboard-engineer`) live in this
repo.

## 2. Dashboard views are folder-per-view with full co-location

**Correction:** the scaffold's flat `templates/` + monolithic
`handlers.go` did not match the operator's mental model ("the `/app`
view lives in `app/`").

**Missing rule:** the dashboard's page organization was unspecified.

**Fix:** new `internal/dashboard/AGENTS.md` documents the page/template
contract: `pages/<name>/{<name>.html,<name>.go}`, route declared
explicitly, arbitrary nesting depth, render contract centralized in
`render.go`, `content` define required.

## 3. Arbitrary nesting depth requires per-page embedding + a leaf descriptor package

**Correction / design refinement:** "any depth" cannot be expressed by a
central `//go:embed pages/*/*.html` glob (`*` never crosses `/`), and
full co-location puts page handlers in subpackages (so `dashboard →
pages/<name>` needs a leaf `page` package to stay acyclic).

**Fix:** per-page `//go:embed <name>.html`; leaf `page` descriptor
package. Both documented in `internal/dashboard/AGENTS.md` and the spec.

## Rule updates made

- `AGENTS.md` §2 — added the skills-location rule.
- `internal/dashboard/AGENTS.md` — new (page/template contract).
- `specs/2026-09-08-dashboard-folder-per-view.md` — Approved.
