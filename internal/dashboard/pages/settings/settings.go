// Package settings is the dashboard view at /settings: it renders the
// live config and lets the operator edit providers, LLM knobs, and
// (read-only) store/dashboard settings.
package settings

import (
	_ "embed"
	"fmt"
	"net/http"
	"net/url"
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
	// Runtime.Get returns a value copy, but Config.Providers is a map (a
	// reference type): the copy shares the live config's map. Deep-copy it
	// before mutating so a validation/save failure can never leak an
	// unvalidated change into the runtime, and so we never race a reader.
	cfg.Providers = cloneProviders(cfg.Providers)

	switch r.FormValue("action") {
	case "add-provider":
		name := strings.TrimSpace(r.FormValue("name"))
		if name == "" {
			return data{Base: base, Config: cfg, Error: "provider name is required"}
		}
		env := strings.TrimSpace(r.FormValue("api_key_env"))
		if env != "" && !config.ValidEnvName(env) {
			return data{Base: base, Config: cfg, Error: "api_key_env must be a valid env var name"}
		}
		baseURL := strings.TrimSpace(r.FormValue("base_url"))
		if baseURL != "" {
			if u, err := url.Parse(baseURL); err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
				return data{Base: base, Config: cfg, Error: "base_url must be a valid http(s) URL"}
			}
		}
		cfg.Providers[name] = config.Provider{BaseURL: baseURL, APIKeyEnv: env}
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
		var err error
		intFields := []struct {
			key string
			dst *int
		}{
			{"max_retries", &cfg.LLM.MaxRetries},
			{"retry_base_ms", &cfg.LLM.RetryBaseMS},
			{"breaker_failures", &cfg.LLM.BreakerFailures},
			{"breaker_cooldown_s", &cfg.LLM.BreakerCooldownS},
			{"daily_token_budget", &cfg.LLM.DailyTokenBudget},
		}
		for _, f := range intFields {
			if *f.dst, err = parseIntOpt(r.FormValue(f.key)); err != nil {
				return data{Base: base, Config: cfg, Error: f.key + ": " + err.Error()}
			}
		}
		if cfg.LLM.BudgetWarnFraction, err = parseFloatOpt(r.FormValue("budget_warn_fraction")); err != nil {
			return data{Base: base, Config: cfg, Error: "budget_warn_fraction: " + err.Error()}
		}
		cfg.LLM.AllowRepair = r.FormValue("ladder_allow_repair") == "true"
		cfg.LLM.AllowReask = r.FormValue("ladder_allow_reask") == "true"
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

// cloneProviders returns a deep copy of the providers map so callers can
// mutate it without touching the live config or racing readers.
func cloneProviders(src map[string]config.Provider) map[string]config.Provider {
	dst := make(map[string]config.Provider, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

// parseIntOpt parses a form value as an int, treating empty as 0 (the
// config default synthesis then fills the real default).
func parseIntOpt(s string) (int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("must be an integer")
	}
	return n, nil
}

// parseFloatOpt parses a form value as a float64, treating empty as 0.
func parseFloatOpt(s string) (float64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("must be a number")
	}
	return f, nil
}
