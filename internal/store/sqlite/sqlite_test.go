package sqlite

import (
	"database/sql"
	"errors"
	"testing"

	"deepcut-harness/internal/agent"
	"deepcut-harness/internal/skill"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestAgentCreateGetRoundTrip(t *testing.T) {
	s := newTestStore(t)
	created, err := s.CreateAgent(agent.Agent{
		Name: "Code Gardener", Prompt: "Refactor legacy code.", Provider: "deepseek", Model: "deepseek-v4-flash", Temperature: 0.2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.ID == "" {
		t.Fatal("CreateAgent did not assign an ID")
	}
	if created.CreatedAt.IsZero() || created.UpdatedAt.IsZero() {
		t.Fatal("CreateAgent did not set timestamps")
	}

	got, err := s.GetAgent(created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Code Gardener" || got.Provider != "deepseek" || got.Model != "deepseek-v4-flash" {
		t.Fatalf("round-trip mismatch: %+v", got)
	}
	if got.Temperature != 0.2 {
		t.Fatalf("temperature = %v", got.Temperature)
	}
}

func TestAgentDuplicateName(t *testing.T) {
	s := newTestStore(t)
	a := agent.Agent{Name: "dup", Prompt: "x", Provider: "deepseek", Model: "m"}
	if _, err := s.CreateAgent(a); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateAgent(a); err == nil {
		t.Fatal("expected duplicate-name error")
	}
}

func TestAgentNotFound(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.GetAgent("missing"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("GetAgent err = %v, want sql.ErrNoRows", err)
	}
	if err := s.DeleteAgent("missing"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("DeleteAgent err = %v, want sql.ErrNoRows", err)
	}
}

func TestAgentAttachDetach(t *testing.T) {
	s := newTestStore(t)
	sk, err := s.CreateSkill(skill.Skill{Name: "python", Runtime: "python3", Category: "stack"})
	if err != nil {
		t.Fatal(err)
	}
	a, err := s.CreateAgent(agent.Agent{Name: "gardener", Prompt: "x", Provider: "deepseek", Model: "m"})
	if err != nil {
		t.Fatal(err)
	}

	a.SkillIDs = append(a.SkillIDs, sk.ID)
	updated, err := s.UpdateAgent(a)
	if err != nil {
		t.Fatal(err)
	}
	if len(updated.SkillIDs) != 1 || updated.SkillIDs[0] != sk.ID {
		t.Fatalf("skill_ids = %v", updated.SkillIDs)
	}

	// detach
	updated.SkillIDs = nil
	if _, err := s.UpdateAgent(updated); err != nil {
		t.Fatal(err)
	}
	got, _ := s.GetAgent(a.ID)
	if len(got.SkillIDs) != 0 {
		t.Fatalf("expected empty skill_ids after detach, got %v", got.SkillIDs)
	}
}

func TestAgentSearchFTSAndLike(t *testing.T) {
	s := newTestStore(t)
	must := func(a agent.Agent) {
		if _, err := s.CreateAgent(a); err != nil {
			t.Fatal(err)
		}
	}
	must(agent.Agent{Name: "Code Gardener", Prompt: "Refactor legacy code.", Provider: "deepseek", Model: "m"})
	must(agent.Agent{Name: "QA Lead", Prompt: "Audit the test suite.", Provider: "deepseek", Model: "m"})
	must(agent.Agent{Name: "DevOps", Prompt: "Provision infrastructure.", Provider: "openai", Model: "gpt"})

	// FTS5 (>= 3 chars) — substring in prompt.
	got, err := s.SearchAgents("legacy")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Name != "Code Gardener" {
		t.Fatalf("SearchAgents(legacy) = %+v", got)
	}

	// LIKE fallback (< 3 chars) — "au" matches "Audit".
	got, err = s.SearchAgents("au")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Name != "QA Lead" {
		t.Fatalf("SearchAgents(au) = %+v", got)
	}

	// No match.
	got, err = s.SearchAgents("zzzzzz")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("SearchAgents(zzzzzz) = %+v, want empty", got)
	}
}

func TestSkillCRUDAndSearch(t *testing.T) {
	s := newTestStore(t)
	created, err := s.CreateSkill(skill.Skill{
		Name: "python", Description: "Run Python 3 scripts.", Category: "stack", Runtime: "python3", AllowedCommands: []string{"python3"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.ID == "" {
		t.Fatal("CreateSkill did not assign an ID")
	}

	got, err := s.GetSkill(created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "python" || got.Runtime != "python3" || got.Category != "stack" {
		t.Fatalf("round-trip mismatch: %+v", got)
	}
	if len(got.AllowedCommands) != 1 || got.AllowedCommands[0] != "python3" {
		t.Fatalf("allowed_commands = %v", got.AllowedCommands)
	}

	// FTS5 search on description.
	res, err := s.SearchSkills("python")
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 1 || res[0].Name != "python" {
		t.Fatalf("SearchSkills(python) = %+v", res)
	}
}

func TestSkillDuplicateName(t *testing.T) {
	s := newTestStore(t)
	sk := skill.Skill{Name: "bash", Runtime: "bash"}
	if _, err := s.CreateSkill(sk); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateSkill(sk); err == nil {
		t.Fatal("expected duplicate-name error")
	}
}
