// Package config loads and validates the Harness runtime configuration.
// It mirrors the sibling repo's convention: a single JSON file
// (config.json) with strict decoding and defaults synthesized in
// Validate(), so a template operators copy for a fresh setup must pass
// the same loader the binary runs.
package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
)

// defaultListenAddr is the loopback bind for the dashboard. Harness is
// local-first: it stays on loopback until multi-user/team mode lands.
const defaultListenAddr = "127.0.0.1:8787"

// Config is the root runtime configuration for the Harness process.
type Config struct {
	Dashboard Dashboard           `json:"dashboard"`
	Store     Store               `json:"store"`
	LLM       LLM                 `json:"llm"`
	Providers map[string]Provider `json:"providers"`
}

// Dashboard configures the local HTTP dashboard.
type Dashboard struct {
	// ListenAddr is the host:port the dashboard binds to.
	ListenAddr string `json:"listen_addr"`
}

// Store configures the persistence backend (pluggable adapters).
type Store struct {
	// Driver selects the adapter: "sqlite" (the others land later).
	Driver string `json:"driver"`
	// DSN is the driver-specific data source name (a file path, or
	// ":memory:" for SQLite).
	DSN string `json:"dsn"`
}

// LLM configures the provider layer's reliability knobs.
type LLM struct {
	MaxRetries         int     `json:"max_retries"`
	RetryBaseMS        int     `json:"retry_base_ms"`
	BreakerFailures    int     `json:"breaker_consecutive_failures"`
	BreakerCooldownS   int     `json:"breaker_cooldown_s"`
	AllowRepair        bool    `json:"ladder_allow_repair"`
	AllowReask         bool    `json:"ladder_allow_reask"`
	DailyTokenBudget   int     `json:"daily_token_budget"`
	BudgetWarnFraction float64 `json:"budget_warn_fraction"`
}

// Provider configures one LLM backend (OpenAI-compatible).
type Provider struct {
	BaseURL   string `json:"base_url"`
	APIKeyEnv string `json:"api_key_env"`
}

// Default returns the zero-config defaults (localhost, SQLite, the
// common OpenAI-compatible providers, and permissive LLM knobs).
func Default() Config {
	return Config{
		Dashboard: Dashboard{ListenAddr: defaultListenAddr},
		Store:     Store{Driver: "sqlite", DSN: "./data/harness.db"},
		LLM: LLM{
			MaxRetries:         2,
			RetryBaseMS:        250,
			BreakerFailures:    5,
			BreakerCooldownS:   60,
			AllowRepair:        true,
			AllowReask:         true,
			DailyTokenBudget:   0, // 0 = unlimited
			BudgetWarnFraction: 0.8,
		},
		Providers: map[string]Provider{
			"deepseek": {BaseURL: "https://api.deepseek.com", APIKeyEnv: "DEEPSEEK_API_KEY"},
			"openai":   {BaseURL: "https://api.openai.com/v1", APIKeyEnv: "OPENAI_API_KEY"},
			"ollama":   {BaseURL: "http://localhost:11434/v1", APIKeyEnv: ""},
		},
	}
}

// Load reads path (if present) and applies defaults for absent fields.
// A missing file is not an error — a first run uses the defaults.
func Load(path string) (Config, error) {
	cfg := Default()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, fmt.Errorf("config: read %s: %w", path, err)
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&cfg); err != nil {
		return cfg, fmt.Errorf("config: parse %s: %w", path, err)
	}
	if err := cfg.Validate(); err != nil {
		return cfg, err
	}
	return cfg, nil
}

// Validate normalizes the config in place and rejects invalid values.
// It mutates the receiver on purpose (synthesized defaults), so callers
// must validate the SAME instance the runtime reads — a validated copy
// is a config the runtime never received.
func (c *Config) Validate() error {
	if c.Dashboard.ListenAddr == "" {
		c.Dashboard.ListenAddr = defaultListenAddr
	}
	if c.Store.Driver == "" {
		c.Store.Driver = "sqlite"
	}
	if c.Store.DSN == "" {
		c.Store.DSN = "./data/harness.db"
	}
	if c.LLM.MaxRetries == 0 {
		c.LLM.MaxRetries = 2
	}
	if c.LLM.RetryBaseMS == 0 {
		c.LLM.RetryBaseMS = 250
	}
	if c.LLM.BreakerFailures == 0 {
		c.LLM.BreakerFailures = 5
	}
	if c.LLM.BreakerCooldownS == 0 {
		c.LLM.BreakerCooldownS = 60
	}
	if c.LLM.BudgetWarnFraction == 0 {
		c.LLM.BudgetWarnFraction = 0.8
	}
	if c.Providers == nil {
		c.Providers = map[string]Provider{}
	}
	return nil
}
