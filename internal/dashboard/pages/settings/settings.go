// Package settings is the dashboard view at /settings: it renders the
// live config and lets the operator edit providers, LLM knobs, and
// (read-only) store/dashboard settings.
package settings

import (
	_ "embed"
	"net/http"
	"strconv"
	"strings"

	"deepcut-harness/internal/config"
	"deepcut-harness/internal/dashboard/page"
)

//go:embed settings.html
var tmpl string

type data struct {
	page.Base
	Config config.Config
	Error  string
	Saved  bool
}

// Page returns the settings view descriptor. The single /settings route
// handles GET (render) and POST (apply a form action).
func Page() page.Page {
	return page.Page{
		Path:     "/settings",
		Name:     "settings",
		Template: tmpl,
		Handler: func(d page.Deps) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				base := page.Base{Title: "Settings", Addr: d.Addr}
				if r.Method == http.MethodPost {
					d.Render(w, r, "settings", apply(r, d, base))
					return
				}
				d.Render(w, r, "settings", data{Base: base, Config: d.Runtime.Get()})
			}
		},
	}
}

// apply runs one settings form action against the active config and
// persists + hot-reloads it.
func apply(r *http.Request, d page.Deps, base page.Base) data {
	if err := r.ParseForm(); err != nil {
		return data{Base: base, Config: d.Runtime.Get(), Error: "invalid form"}
	}
	cfg := d.Runtime.Get()
	switch r.FormValue("action") {
	case "add-provider":
		name := strings.TrimSpace(r.FormValue("name"))
		if name == "" {
			return data{Base: base, Config: cfg, Error: "provider name is required"}
		}
		env := strings.TrimSpace(r.FormValue("api_key_env"))
		cfg.Providers[name] = config.Provider{
			BaseURL:   strings.TrimSpace(r.FormValue("base_url")),
			APIKeyEnv: env,
		}
		if key := r.FormValue("key"); key != "" {
			if env == "" {
				return data{Base: base, Config: cfg, Error: "api_key_env is required to store the key"}
			}
			if err := d.Editor.WriteEnvKey(env, key); err != nil {
				return data{Base: base, Config: cfg, Error: "write .env: " + err.Error()}
			}
		}
	case "remove-provider":
		delete(cfg.Providers, r.FormValue("name"))
	case "update-llm":
		cfg.LLM.MaxRetries = atoi(r.FormValue("max_retries"))
		cfg.LLM.RetryBaseMS = atoi(r.FormValue("retry_base_ms"))
		cfg.LLM.BreakerFailures = atoi(r.FormValue("breaker_failures"))
		cfg.LLM.BreakerCooldownS = atoi(r.FormValue("breaker_cooldown_s"))
		cfg.LLM.DailyTokenBudget = atoi(r.FormValue("daily_token_budget"))
	default:
		return data{Base: base, Config: cfg, Error: "unknown action"}
	}
	if err := cfg.Validate(); err != nil {
		return data{Base: base, Config: cfg, Error: err.Error()}
	}
	if err := d.Editor.Save(cfg); err != nil {
		return data{Base: base, Config: cfg, Error: "save: " + err.Error()}
	}
	d.Runtime.Set(cfg)
	return data{Base: base, Config: cfg, Saved: true}
}

func atoi(s string) int {
	n, _ := strconv.Atoi(strings.TrimSpace(s))
	return n
}
