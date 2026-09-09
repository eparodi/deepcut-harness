package cli

import (
	"deepcut-harness/internal/config"
	"deepcut-harness/internal/store"
)

// openStore loads the config and opens the configured store. The caller
// is responsible for closing the returned store.
func openStore() (store.Store, error) {
	cfg, err := config.Load("config.json")
	if err != nil {
		return nil, err
	}
	return store.Open(cfg.Store.Driver, cfg.Store.DSN)
}
