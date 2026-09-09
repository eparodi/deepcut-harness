package openai

import (
	"fmt"
	"os"
	"sort"
	"time"

	"deepcut-harness/internal/config"
	"deepcut-harness/internal/llm"
)

// Registry maps provider names to llm.Provider instances built from
// config. It is the seam the rest of the codebase uses to reach many
// providers through one interface.
type Registry struct {
	providers map[string]llm.Provider
}

// NewRegistry builds a provider for every entry in cfg.Providers.
func NewRegistry(cfg config.Config) *Registry {
	r := &Registry{providers: map[string]llm.Provider{}}
	for name, pc := range cfg.Providers {
		r.providers[name] = New(Config{
			BaseURL: pc.BaseURL,
			APIKey:  os.Getenv(pc.APIKeyEnv),
		}, Options{
			MaxRetries:         cfg.LLM.MaxRetries,
			RetryBase:          time.Duration(cfg.LLM.RetryBaseMS) * time.Millisecond,
			BreakerFailures:    cfg.LLM.BreakerFailures,
			BreakerCooldown:    time.Duration(cfg.LLM.BreakerCooldownS) * time.Second,
			AllowRepair:        cfg.LLM.AllowRepair,
			AllowReask:         cfg.LLM.AllowReask,
			DailyTokenBudget:   cfg.LLM.DailyTokenBudget,
			BudgetWarnFraction: cfg.LLM.BudgetWarnFraction,
		})
	}
	return r
}

// Provider returns the named provider, or an error if it is not
// configured.
func (r *Registry) Provider(name string) (llm.Provider, error) {
	p, ok := r.providers[name]
	if !ok {
		return nil, fmt.Errorf("openai: provider %q not configured", name)
	}
	return p, nil
}

// Names returns the configured provider names, sorted.
func (r *Registry) Names() []string {
	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
