package id

import "testing"

func TestNewUnique(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 1000; i++ {
		got := New()
		if got == "" {
			t.Fatal("New returned empty id")
		}
		if seen[got] {
			t.Fatalf("duplicate id %q after %d iterations", got, i)
		}
		seen[got] = true
	}
}

func TestNewUsesAlphabet(t *testing.T) {
	for i := 0; i < 100; i++ {
		got := New()
		for _, c := range got {
			if !containsRune(alphabet, c) {
				t.Fatalf("id %q contains non-base62 rune %q", got, c)
			}
		}
	}
}

func containsRune(s string, r rune) bool {
	for _, c := range s {
		if c == r {
			return true
		}
	}
	return false
}
