// Package sqlite is the SQLite + FTS5 storage adapter. It implements the
// agent.AgentStore and skill.SkillStore ports on top of a single SQLite
// database, with two FTS5 virtual tables (trigram tokenizer) for
// full-text search.
package sqlite

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"deepcut-harness/internal/agent"
	"deepcut-harness/internal/id"
	"deepcut-harness/internal/skill"
)

// Store is the SQLite adapter implementing both store ports.
type Store struct {
	db *sql.DB
}

// Open opens (and migrates) a SQLite database at dsn (a file path or
// ":memory:").
func Open(dsn string) (*Store, error) {
	// Ensure the parent directory exists for file-backed DSNs (a first
	// run must not fail because ./data is missing). ":memory:" has no file.
	if dsn != "" && dsn != ":memory:" {
		if err := os.MkdirAll(filepath.Dir(dsn), 0o755); err != nil {
			return nil, fmt.Errorf("sqlite: mkdir: %w", err)
		}
	}
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("sqlite: open: %w", err)
	}
	// A single connection serializes writes: SQLite allows one writer at
	// a time, and this is a single-user tool.
	db.SetMaxOpenConns(1)
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("sqlite: migrate: %w", err)
	}
	return s, nil
}

// Close closes the underlying database.
func (s *Store) Close() error { return s.db.Close() }

func (s *Store) migrate() error {
	for _, stmt := range schema {
		if _, err := s.db.Exec(stmt); err != nil {
			return fmt.Errorf("schema: %w", err)
		}
	}
	return nil
}

// schema is the DDL + pragmas, run idempotently at startup. FTS5 uses the
// trigram tokenizer for substring search; external-content FTS5 tables
// read from the primary tables, and the triggers keep the FTS index in
// sync.
var schema = []string{
	"PRAGMA journal_mode=WAL",
	"PRAGMA foreign_keys=ON",
	"PRAGMA busy_timeout=5000",

	`CREATE TABLE IF NOT EXISTS agents (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		uuid        TEXT NOT NULL UNIQUE,
		name        TEXT NOT NULL UNIQUE,
		prompt      TEXT NOT NULL,
		provider    TEXT NOT NULL,
		model       TEXT NOT NULL,
		temperature REAL NOT NULL,
		skill_ids   TEXT NOT NULL DEFAULT '[]',
		created_at  INTEGER NOT NULL,
		updated_at  INTEGER NOT NULL
	)`,

	`CREATE TABLE IF NOT EXISTS skills (
		id               INTEGER PRIMARY KEY AUTOINCREMENT,
		uuid             TEXT NOT NULL UNIQUE,
		name             TEXT NOT NULL UNIQUE,
		description      TEXT NOT NULL,
		category         TEXT NOT NULL DEFAULT '',
		runtime          TEXT NOT NULL,
		allowed_commands TEXT NOT NULL DEFAULT '[]',
		created_at       INTEGER NOT NULL
	)`,

	`CREATE VIRTUAL TABLE IF NOT EXISTS agents_fts USING fts5(
		name, prompt, provider, model,
		content='agents', content_rowid='id', tokenize='trigram'
	)`,

	`CREATE VIRTUAL TABLE IF NOT EXISTS skills_fts USING fts5(
		name, description, runtime,
		content='skills', content_rowid='id', tokenize='trigram'
	)`,

	`CREATE TRIGGER IF NOT EXISTS agents_ai AFTER INSERT ON agents BEGIN
		INSERT INTO agents_fts(rowid, name, prompt, provider, model)
		VALUES (new.id, new.name, new.prompt, new.provider, new.model);
	END`,

	`CREATE TRIGGER IF NOT EXISTS agents_ad AFTER DELETE ON agents BEGIN
		INSERT INTO agents_fts(agents_fts, rowid, name, prompt, provider, model)
		VALUES ('delete', old.id, old.name, old.prompt, old.provider, old.model);
	END`,

	`CREATE TRIGGER IF NOT EXISTS agents_au AFTER UPDATE ON agents BEGIN
		INSERT INTO agents_fts(agents_fts, rowid, name, prompt, provider, model)
		VALUES ('delete', old.id, old.name, old.prompt, old.provider, old.model);
		INSERT INTO agents_fts(rowid, name, prompt, provider, model)
		VALUES (new.id, new.name, new.prompt, new.provider, new.model);
	END`,

	`CREATE TRIGGER IF NOT EXISTS skills_ai AFTER INSERT ON skills BEGIN
		INSERT INTO skills_fts(rowid, name, description, runtime)
		VALUES (new.id, new.name, new.description, new.runtime);
	END`,

	`CREATE TRIGGER IF NOT EXISTS skills_ad AFTER DELETE ON skills BEGIN
		INSERT INTO skills_fts(skills_fts, rowid, name, description, runtime)
		VALUES ('delete', old.id, old.name, old.description, old.runtime);
	END`,

	`CREATE TRIGGER IF NOT EXISTS skills_au AFTER UPDATE ON skills BEGIN
		INSERT INTO skills_fts(skills_fts, rowid, name, description, runtime)
		VALUES ('delete', old.id, old.name, old.description, old.runtime);
		INSERT INTO skills_fts(rowid, name, description, runtime)
		VALUES (new.id, new.name, new.description, new.runtime);
	END`,
}

const agentCols = "uuid, name, prompt, provider, model, temperature, skill_ids, created_at, updated_at"
const skillCols = "uuid, name, description, category, runtime, allowed_commands, created_at"

// ---- agents ----

func (s *Store) CreateAgent(a agent.Agent) (agent.Agent, error) {
	if err := a.Validate(); err != nil {
		return agent.Agent{}, err
	}
	now := time.Now().UTC()
	a.ID = id.New()
	a.CreatedAt = now
	a.UpdatedAt = now
	if a.SkillIDs == nil {
		a.SkillIDs = []string{}
	}
	ids, err := json.Marshal(a.SkillIDs)
	if err != nil {
		return agent.Agent{}, err
	}
	if _, err := s.db.Exec(
		`INSERT INTO agents (uuid, name, prompt, provider, model, temperature, skill_ids, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		a.ID, a.Name, a.Prompt, a.Provider, a.Model, a.Temperature, string(ids), now.Unix(), now.Unix(),
	); err != nil {
		if isUnique(err) {
			return agent.Agent{}, fmt.Errorf("agent: name %q already exists", a.Name)
		}
		return agent.Agent{}, err
	}
	return a, nil
}

func (s *Store) GetAgent(uuid string) (agent.Agent, error) {
	return scanAgent(s.db.QueryRow("SELECT "+agentCols+" FROM agents WHERE uuid = ?", uuid))
}

func (s *Store) ListAgents() ([]agent.Agent, error) {
	rows, err := s.db.Query("SELECT " + agentCols + " FROM agents ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []agent.Agent
	for rows.Next() {
		a, err := scanAgent(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *Store) UpdateAgent(a agent.Agent) (agent.Agent, error) {
	if err := a.Validate(); err != nil {
		return agent.Agent{}, err
	}
	a.UpdatedAt = time.Now().UTC()
	if a.SkillIDs == nil {
		a.SkillIDs = []string{}
	}
	ids, err := json.Marshal(a.SkillIDs)
	if err != nil {
		return agent.Agent{}, err
	}
	res, err := s.db.Exec(
		`UPDATE agents SET name=?, prompt=?, provider=?, model=?, temperature=?, skill_ids=?, updated_at=? WHERE uuid=?`,
		a.Name, a.Prompt, a.Provider, a.Model, a.Temperature, string(ids), a.UpdatedAt.Unix(), a.ID,
	)
	if err != nil {
		if isUnique(err) {
			return agent.Agent{}, fmt.Errorf("agent: name %q already exists", a.Name)
		}
		return agent.Agent{}, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return agent.Agent{}, sql.ErrNoRows
	}
	return a, nil
}

func (s *Store) DeleteAgent(uuid string) error {
	res, err := s.db.Exec("DELETE FROM agents WHERE uuid = ?", uuid)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *Store) SearchAgents(query string) ([]agent.Agent, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		return s.ListAgents()
	}
	if len(q) < 3 {
		like := "%" + q + "%"
		rows, err := s.db.Query("SELECT "+agentCols+" FROM agents WHERE name LIKE ? OR prompt LIKE ? OR provider LIKE ? OR model LIKE ? ORDER BY name", like, like, like, like)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		return collectAgents(rows)
	}
	match := `"` + strings.ReplaceAll(q, `"`, " ") + `"`
	rows, err := s.db.Query(
		`SELECT a.`+strings.ReplaceAll(agentCols, ", ", ", a.")+` FROM agents_fts f JOIN agents a ON a.id = f.rowid WHERE agents_fts MATCH ? ORDER BY bm25(agents_fts)`,
		match,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectAgents(rows)
}

// ---- skills ----

func (s *Store) CreateSkill(sk skill.Skill) (skill.Skill, error) {
	if err := sk.Validate(); err != nil {
		return skill.Skill{}, err
	}
	now := time.Now().UTC()
	sk.ID = id.New()
	sk.CreatedAt = now
	if sk.AllowedCommands == nil {
		sk.AllowedCommands = []string{}
	}
	cmds, err := json.Marshal(sk.AllowedCommands)
	if err != nil {
		return skill.Skill{}, err
	}
	if _, err := s.db.Exec(
		`INSERT INTO skills (uuid, name, description, category, runtime, allowed_commands, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		sk.ID, sk.Name, sk.Description, sk.Category, sk.Runtime, string(cmds), now.Unix(),
	); err != nil {
		if isUnique(err) {
			return skill.Skill{}, fmt.Errorf("skill: name %q already exists", sk.Name)
		}
		return skill.Skill{}, err
	}
	return sk, nil
}

func (s *Store) GetSkill(uuid string) (skill.Skill, error) {
	return scanSkill(s.db.QueryRow("SELECT "+skillCols+" FROM skills WHERE uuid = ?", uuid))
}

func (s *Store) ListSkills() ([]skill.Skill, error) {
	rows, err := s.db.Query("SELECT " + skillCols + " FROM skills ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []skill.Skill
	for rows.Next() {
		sk, err := scanSkill(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, sk)
	}
	return out, rows.Err()
}

func (s *Store) DeleteSkill(uuid string) error {
	res, err := s.db.Exec("DELETE FROM skills WHERE uuid = ?", uuid)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *Store) SearchSkills(query string) ([]skill.Skill, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		return s.ListSkills()
	}
	if len(q) < 3 {
		like := "%" + q + "%"
		rows, err := s.db.Query("SELECT "+skillCols+" FROM skills WHERE name LIKE ? OR description LIKE ? OR runtime LIKE ? ORDER BY name", like, like, like)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		return collectSkills(rows)
	}
	match := `"` + strings.ReplaceAll(q, `"`, " ") + `"`
	rows, err := s.db.Query(
		`SELECT s.`+strings.ReplaceAll(skillCols, ", ", ", s.")+` FROM skills_fts f JOIN skills s ON s.id = f.rowid WHERE skills_fts MATCH ? ORDER BY bm25(skills_fts)`,
		match,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectSkills(rows)
}

// ---- scanning ----

func scanAgent(row interface{ Scan(...any) error }) (agent.Agent, error) {
	var a agent.Agent
	var ids string
	var createdAt, updatedAt int64
	if err := row.Scan(&a.ID, &a.Name, &a.Prompt, &a.Provider, &a.Model, &a.Temperature, &ids, &createdAt, &updatedAt); err != nil {
		return agent.Agent{}, err
	}
	if err := json.Unmarshal([]byte(ids), &a.SkillIDs); err != nil {
		return agent.Agent{}, err
	}
	if a.SkillIDs == nil {
		a.SkillIDs = []string{}
	}
	a.CreatedAt = time.Unix(createdAt, 0).UTC()
	a.UpdatedAt = time.Unix(updatedAt, 0).UTC()
	return a, nil
}

func scanSkill(row interface{ Scan(...any) error }) (skill.Skill, error) {
	var sk skill.Skill
	var cmds string
	var createdAt int64
	if err := row.Scan(&sk.ID, &sk.Name, &sk.Description, &sk.Category, &sk.Runtime, &cmds, &createdAt); err != nil {
		return skill.Skill{}, err
	}
	if err := json.Unmarshal([]byte(cmds), &sk.AllowedCommands); err != nil {
		return skill.Skill{}, err
	}
	if sk.AllowedCommands == nil {
		sk.AllowedCommands = []string{}
	}
	sk.CreatedAt = time.Unix(createdAt, 0).UTC()
	return sk, nil
}

func collectAgents(rows *sql.Rows) ([]agent.Agent, error) {
	var out []agent.Agent
	for rows.Next() {
		a, err := scanAgent(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func collectSkills(rows *sql.Rows) ([]skill.Skill, error) {
	var out []skill.Skill
	for rows.Next() {
		sk, err := scanSkill(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, sk)
	}
	return out, rows.Err()
}

func isUnique(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed")
}
