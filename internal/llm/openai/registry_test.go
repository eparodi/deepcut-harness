package openai

import (
	"testing"

	"deepcut-harness/internal/config"
)

func TestRegistryFromConfig(t *testing.T) {
	r := NewRegistry(config.Default())

	if _, err := r.Provider("deepseek"); err != nil {
		t.Fatalf("deepseek: %v", err)
	}
	if _, err := r.Provider("openai"); err != nil {
		t.Fatalf("openai: %v", err)
	}
	if _, err := r.Provider("ollama"); err != nil {
		t.Fatalf("ollama: %v", err)
	}
	if _, err := r.Provider("not-configured"); err == nil {
		t.Fatal("expected error for unconfigured provider")
	}

	names := r.Names()
	if len(names) != 3 {
		t.Fatalf("names = %v, want 3", names)
	}
}
