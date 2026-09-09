package config

import (
	"os"
	"testing"
)

func TestDefault(t *testing.T) {
	cfg := Default()
	if cfg.Dashboard.ListenAddr != "127.0.0.1:8787" {
		t.Fatalf("default listen addr = %q, want 127.0.0.1:8787", cfg.Dashboard.ListenAddr)
	}
	if cfg.Store.Driver != "sqlite" || cfg.Store.DSN != "./data/harness.db" {
		t.Fatalf("default store = %+v", cfg.Store)
	}
}

func TestLoadMissingFileUsesDefaults(t *testing.T) {
	cfg, err := Load("does-not-exist.json")
	if err != nil {
		t.Fatalf("Load returned error for missing file: %v", err)
	}
	if cfg.Dashboard.ListenAddr != "127.0.0.1:8787" {
		t.Fatalf("listen addr = %q, want default", cfg.Dashboard.ListenAddr)
	}
}

func TestLoadExample(t *testing.T) {
	cfg, err := Load("../../config.json.example")
	if err != nil {
		t.Fatalf("config.json.example must load under the strict loader: %v", err)
	}
	if cfg.Dashboard.ListenAddr != "127.0.0.1:8787" {
		t.Fatalf("example listen addr = %q, want 127.0.0.1:8787", cfg.Dashboard.ListenAddr)
	}
}

func TestLoadRejectsUnknownField(t *testing.T) {
	path := t.TempDir() + "/config.json"
	if err := writeFile(path, `{"dashboard":{"listen_addr":"127.0.0.1:8787"},"typo":true}`); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("expected strict loader to reject an unknown field")
	}
}

func TestValidateFillsEmptyListenAddr(t *testing.T) {
	cfg := Config{}
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
	if cfg.Dashboard.ListenAddr != "127.0.0.1:8787" {
		t.Fatalf("Validate did not fill default: %q", cfg.Dashboard.ListenAddr)
	}
}

func TestDefaultLLMAndProviders(t *testing.T) {
	cfg := Default()
	if cfg.LLM.MaxRetries != 2 || cfg.LLM.RetryBaseMS != 250 {
		t.Fatalf("llm defaults = %+v", cfg.LLM)
	}
	if cfg.Providers["deepseek"].BaseURL != "https://api.deepseek.com" || cfg.Providers["deepseek"].APIKeyEnv != "DEEPSEEK_API_KEY" {
		t.Fatalf("deepseek provider = %+v", cfg.Providers["deepseek"])
	}
	if cfg.Providers["openai"].APIKeyEnv != "OPENAI_API_KEY" {
		t.Fatalf("openai provider = %+v", cfg.Providers["openai"])
	}
	if cfg.Providers["ollama"].BaseURL != "http://localhost:11434/v1" {
		t.Fatalf("ollama provider = %+v", cfg.Providers["ollama"])
	}
}

func TestDefaultSource(t *testing.T) {
	cfg := Default()
	if cfg.Source.MaxBytes != 1<<20 {
		t.Fatalf("default max bytes = %d, want 1 MiB", cfg.Source.MaxBytes)
	}
	if cfg.Source.TimeoutMS != 10000 {
		t.Fatalf("default timeout ms = %d, want 10000", cfg.Source.TimeoutMS)
	}
	if cfg.Source.BlockPrivateHosts {
		t.Fatal("block_private_hosts should default false")
	}
}

func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}
