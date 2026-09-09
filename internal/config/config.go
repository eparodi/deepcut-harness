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
	Dashboard Dashboard `json:"dashboard"`
	Store     Store     `json:"store"`
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

// Default returns the zero-config defaults (localhost, SQLite on disk).
func Default() Config {
	return Config{
		Dashboard: Dashboard{ListenAddr: defaultListenAddr},
		Store:     Store{Driver: "sqlite", DSN: "./data/harness.db"},
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
	return nil
}
