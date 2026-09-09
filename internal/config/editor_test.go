package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRuntimeGetSet(t *testing.T) {
	r := NewRuntime(Default())
	if got := r.Get().Dashboard.ListenAddr; got != "127.0.0.1:8787" {
		t.Fatalf("initial addr = %q", got)
	}
	cfg := Default()
	cfg.Dashboard.ListenAddr = "127.0.0.1:9999"
	r.Set(cfg)
	if got := r.Get().Dashboard.ListenAddr; got != "127.0.0.1:9999" {
		t.Fatalf("after Set addr = %q", got)
	}
}

func TestEditorSaveRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	e := NewEditor(path, filepath.Join(dir, ".env"))

	cfg := Default()
	cfg.Providers["openrouter"] = Provider{BaseURL: "https://openrouter.ai/api/v1", APIKeyEnv: "OPENROUTER_API_KEY"}
	if err := e.Save(cfg); err != nil {
		t.Fatal(err)
	}

	got, err := e.Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.Providers["openrouter"].BaseURL != "https://openrouter.ai/api/v1" {
		t.Fatalf("openrouter provider not round-tripped: %+v", got.Providers)
	}
}

func TestEditorWriteEnvKey(t *testing.T) {
	dir := t.TempDir()
	env := filepath.Join(dir, ".env")
	e := NewEditor(filepath.Join(dir, "config.json"), env)

	if err := e.WriteEnvKey("DEEPSEEK_API_KEY", "sk-test"); err != nil {
		t.Fatal(err)
	}
	if err := e.WriteEnvKey("OPENAI_API_KEY", "sk-other"); err != nil {
		t.Fatal(err)
	}
	// Update the first key without clobbering the second.
	if err := e.WriteEnvKey("DEEPSEEK_API_KEY", "sk-updated"); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(env)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, "DEEPSEEK_API_KEY=sk-updated\n") {
		t.Fatalf(".env missing updated key:\n%s", content)
	}
	if !strings.Contains(content, "OPENAI_API_KEY=sk-other\n") {
		t.Fatalf(".env missing preserved key:\n%s", content)
	}
	info, err := os.Stat(env)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf(".env mode = %o, want 600", info.Mode().Perm())
	}
}
