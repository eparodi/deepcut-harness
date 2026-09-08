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

func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}
