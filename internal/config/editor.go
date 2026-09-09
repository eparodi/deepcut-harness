package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Editor loads and saves the runtime config file and .env keys. It is the
// seam the /settings page uses to persist changes (atomically) and to
// write API keys into the gitignored .env (chmod 600).
type Editor struct {
	configPath string
	envPath    string
}

// NewEditor returns an Editor for the given config.json and .env paths.
func NewEditor(configPath, envPath string) *Editor {
	return &Editor{configPath: configPath, envPath: envPath}
}

// Load reads the config file (or defaults when it is absent).
func (e *Editor) Load() (Config, error) {
	return Load(e.configPath)
}

// Save validates and atomically writes cfg to config.json.
func (e *Editor) Save(cfg Config) error {
	if err := cfg.Validate(); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return atomicWrite(e.configPath, data)
}

// envNameRe matches a valid shell environment variable identifier.
var envNameRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// ValidEnvName reports whether name is a valid shell environment variable
// identifier (used before writing a key into .env).
func ValidEnvName(name string) bool {
	return envNameRe.MatchString(name)
}

// WriteEnvKey writes or updates key=value in the .env file, preserving the
// other lines, and chmods the file to 0600. key must be a valid env var
// name.
func (e *Editor) WriteEnvKey(key, value string) error {
	if !ValidEnvName(key) {
		return fmt.Errorf("config: invalid env var name %q", key)
	}
	var lines []string
	if data, err := os.ReadFile(e.envPath); err == nil {
		for _, line := range strings.Split(strings.TrimRight(string(data), "\n"), "\n") {
			if line != "" {
				lines = append(lines, line)
			}
		}
	}
	prefix := key + "="
	found := false
	for i, line := range lines {
		if strings.HasPrefix(line, prefix) {
			lines[i] = prefix + value
			found = true
			break
		}
	}
	if !found {
		lines = append(lines, prefix+value)
	}
	content := strings.Join(lines, "\n") + "\n"
	if err := os.MkdirAll(filepath.Dir(e.envPath), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(e.envPath, []byte(content), 0o600); err != nil {
		return err
	}
	return os.Chmod(e.envPath, 0o600)
}

// atomicWrite writes data to path via a temp file + rename, so a partial
// write can never corrupt the config.
func atomicWrite(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".cfg-*.tmp")
	if err != nil {
		return fmt.Errorf("config: create temp: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("config: write temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("config: close temp: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("config: rename: %w", err)
	}
	return nil
}
