// Package skill defines the Skill domain type — the "hands" of Harness:
// a runtime + allowed commands. It also declares the SkillStore
// persistence contract.
package skill

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// Skill is the hands: a runtime + allowed commands. Name is a kebab-case
// slug; category mirrors the agent-skill ecosystem (role | stack |
// process).
type Skill struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	Description     string    `json:"description"`
	Category        string    `json:"category"`
	Runtime         string    `json:"runtime"`
	AllowedCommands []string  `json:"allowed_commands"`
	CreatedAt       time.Time `json:"created_at"`
}

// Validate checks the skill's fields against their allowed set.
func (s *Skill) Validate() error {
	if strings.TrimSpace(s.Name) == "" {
		return errors.New("skill: name is required")
	}
	if strings.TrimSpace(s.Runtime) == "" {
		return errors.New("skill: runtime is required")
	}
	if s.Category != "" {
		switch s.Category {
		case "role", "stack", "process":
		default:
			return fmt.Errorf("skill: category must be role|stack|process, got %q", s.Category)
		}
	}
	return nil
}

// SkillStore is the persistence contract for skills (the port). Each
// storage adapter (SQLite, Postgres, Elasticsearch, …) implements it.
// Methods are entity-prefixed so a combined Store can embed both this
// and agent.AgentStore without name collisions.
type SkillStore interface {
	CreateSkill(s Skill) (Skill, error)
	GetSkill(id string) (Skill, error)
	ListSkills() ([]Skill, error)
	DeleteSkill(id string) error
	SearchSkills(query string) ([]Skill, error)
}
