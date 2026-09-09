# Session Log — 2026-09-09

> Running record of corrections and user feedback for the config-admin
> UI + CSRF session (PR #5).

| # | What happened | Root cause | Fix / resolution |
|---|---------------|-----------|------------------|
| 1 | Review: `Runtime.Get()` returns a shallow copy; `apply` mutated the shared `Providers` map before validate/save (could leak + race) | Go value copy shares map fields | Deep-copy `Providers` before mutating; regression test. → generic §10.68. |
| 2 | Review: no CSRF on the mutating `/settings` forms | localhost-only dashboard had no CSRF guard | Per-process synchronizer token + middleware + `{{csrf}}` template func + `Server.CSRFToken()`. → Harness §10.1. |
| 3 | `write_file` mangled `settings.html` (auto-close injected stray `</h1>`/`</code>`/`</li>`) | HTML auto-close pass in the edit tool | Rewrote via quoted heredoc; re-read to verify. → generic §10.67. |
| 4 | `atoi`/`atof` silently coerced non-numeric input to 0 | silent parse fallback | `parseIntOpt`/`parseFloatOpt` reject non-numeric (empty still means default). |
| 5 | `base_url` and `api_key_env` accepted unvalidated | no input validation | `url.Parse` http(s)+host check; `ValidEnvName` regex (handler + `WriteEnvKey`). |
| 6 | "LLM changes apply immediately" overstated — LLM layer has no runtime consumer yet | banner overclaimed behavior | Accurate banner: "persisted; store/dashboard need restart". |

## Follow-ups / open questions

- [ ] Wire `openai.NewRegistry` into `run.go` — the LLM layer has no runtime consumer yet; closes when the wizard lands.
- [ ] Full SSRF allow-list guard (spec `source-reading`) — only URL-format hygiene added now.
