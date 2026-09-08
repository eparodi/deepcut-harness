// Package store is the storage seam for Harness: it exposes a combined
// Store interface (the port) and a driver-based Open that selects the
// adapter. Each backend (SQLite, later Postgres/Elasticsearch) is an
// adapter implementing the same interface.
package store

import (
	"fmt"

	"deepcut-harness/internal/agent"
	"deepcut-harness/internal/skill"
	"deepcut-harness/internal/store/sqlite"
)

// Store combines the agent and skill persistence contracts; one adapter
// implements both.
type Store interface {
	agent.AgentStore
	skill.SkillStore
	Close() error
}

// Open opens the store for the given driver and DSN. Only "sqlite" is
// implemented today; the switch is the seam for future adapters.
func Open(driver, dsn string) (Store, error) {
	switch driver {
	case "sqlite":
		return sqlite.Open(dsn)
	default:
		return nil, fmt.Errorf("store: unknown driver %q", driver)
	}
}
