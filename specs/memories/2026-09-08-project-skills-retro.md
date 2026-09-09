# Retro — Project Technical Skills (2026-09-08)

> Cross-references `2026-09-08-session-log.md`.

## What was done

Authored the three project-specific technical skills and wired them in:

- `.agents/skills/go-harness/SKILL.md` — the project stack skill.
- `.agents/skills/harness-engineer/SKILL.md` — owns all Go in `internal/`.
- `.agents/skills/dashboard-engineer/SKILL.md` — owns the dashboard frontend.
- `zed/profiles.json`, `HOW_WE_WORK.md`, `README.md` updated.

## Correction / learning → rule

### Repo disallows merge commits

**What happened:** `gh pr merge 1 --merge` failed with "Merge commits are
not allowed on this repository." Squash merge succeeded.

**Missing rule:** the repo's merge strategy wasn't documented.

**Fix:** added to root `AGENTS.md` §6 — squash-merge PRs
(`gh pr merge <n> --squash --delete-branch`), or rebase; never `--merge`.

### (Re-affirmed) skills are project-specific technical only

The three new skills live in-repo; generic roles/stack skills continue to
load from `deepcut-skills` — already covered by root `AGENTS.md` §2.

## Rule updates made

- `AGENTS.md` §6 — merge-strategy rule (squash only).
