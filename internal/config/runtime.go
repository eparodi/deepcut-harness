package config

import "sync"

// Runtime is a mutex-guarded holder of the active config, so a /settings
// save can swap in a new config for live reload without tearing down the
// process.
type Runtime struct {
	mu  sync.RWMutex
	cfg Config
}

// NewRuntime wraps cfg in a Runtime.
func NewRuntime(cfg Config) *Runtime {
	return &Runtime{cfg: cfg}
}

// Get returns a snapshot of the active config.
func (r *Runtime) Get() Config {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.cfg
}

// Set swaps in a new active config.
func (r *Runtime) Set(cfg Config) {
	r.mu.Lock()
	r.cfg = cfg
	r.mu.Unlock()
}
