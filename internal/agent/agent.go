// Package agent defines the Agent domain type — the "brain" of Harness:
// a persona + goal wired to an LLM provider/model, with attached skills
// (the "hands"). It also declares the AgentStore persistence contract.
package agent

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// Agent is the brain: a persona + goal, wired to an LLM provider/model.
type Agent struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Prompt      string    `json:"prompt"`
	Provider    string    `json:"provider"`
	Model       string    `json:"model"`
	Temperature float64   `json:"temperature"`
	SkillIDs    []string  `json:"skill_ids"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Validate checks the agent's fields against their allowed set.
func (a *Agent) Validate() error {
	if strings.TrimSpace(a.Name) == "" {
		return errors.New("agent: name is required")
	}
	if strings.TrimSpace(a.Prompt) == "" {
		return errors.New("agent: prompt is required")
	}
	if strings.TrimSpace(a.Provider) == "" {
		return errors.New("agent: provider is required")
	}
	if strings.TrimSpace(a.Model) == "" {
		return errors.New("agent: model is required")
	}
	if a.Temperature < 0 || a.Temperature > 1 {
		return fmt.Errorf("agent: temperature must be in [0,1], got %v", a.Temperature)
	}
	return nil
}

// AgentStore is the persistence contract for agents (the port). Each
// storage adapter (SQLite, Postgres, Elasticsearch, …) implements it.
// Methods are entity-prefixed so a combined Store can embed both this
// and skill.SkillStore without name collisions.
type AgentStore interface {
	CreateAgent(a Agent) (Agent, error)
	GetAgent(id string) (Agent, error)
	ListAgents() ([]Agent, error)
	UpdateAgent(a Agent) (Agent, error) // also the attach/detach path
	DeleteAgent(id string) error
	SearchAgents(query string) ([]Agent, error)
}
