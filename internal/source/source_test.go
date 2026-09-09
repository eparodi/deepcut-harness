package source

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRead(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "root")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "hello.txt"), []byte("hello world"), 0o644); err != nil {
		t.Fatal(err)
	}
	// A secret OUTSIDE root, reachable via `..` and via a symlink.
	if err := os.WriteFile(filepath.Join(base, "secret.txt"), []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(base, "secret.txt"), filepath.Join(root, "link.txt")); err != nil {
		t.Fatal(err)
	}

	s := Source{Root: root}
	tests := []struct {
		name    string
		path    string
		want    string
		wantErr bool
	}{
		{"happy", "hello.txt", "hello world", false},
		{"traversal", "../secret.txt", "", true},
		{"absolute", "/etc/passwd", "", true},
		{"symlink escape", "link.txt", "", true},
		{"missing", "nope.txt", "", true},
		{"empty", "", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := s.Read(context.Background(), tt.path)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("Read(%q) = nil error, want error", tt.path)
				}
				return
			}
			if err != nil {
				t.Fatalf("Read(%q): %v", tt.path, err)
			}
			if got != tt.want {
				t.Fatalf("Read(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestReadTooLarge(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "big.txt"), make([]byte, 100), 0o644); err != nil {
		t.Fatal(err)
	}
	s := Source{Root: root, MaxBytes: 10}
	if _, err := s.Read(context.Background(), "big.txt"); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("err = %v, want ErrTooLarge", err)
	}
}

func TestFetch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "hello from http")
	}))
	defer srv.Close()

	got, err := (Source{}).Fetch(context.Background(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if got != "hello from http" {
		t.Fatalf("Fetch = %q", got)
	}
}

func TestFetchSchemeRejected(t *testing.T) {
	if _, err := (Source{}).Fetch(context.Background(), "file:///etc/passwd"); err == nil {
		t.Fatal("want error for non-http scheme")
	}
}

func TestFetchNon2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	if _, err := (Source{}).Fetch(context.Background(), srv.URL); err == nil {
		t.Fatal("want error for non-2xx status")
	}
}

func TestFetchMetadataBlocked(t *testing.T) {
	if _, err := (Source{}).Fetch(context.Background(), "http://169.254.169.254/latest/meta-data"); err == nil {
		t.Fatal("want error for metadata host")
	}
}

func TestFetchPrivateBlocked(t *testing.T) {
	s := Source{BlockPrivateHosts: true}
	if _, err := s.Fetch(context.Background(), "http://127.0.0.1:1/x"); err == nil {
		t.Fatal("want error for private host")
	}
}

func TestFetchTooLarge(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, string(make([]byte, 100)))
	}))
	defer srv.Close()

	if _, err := (Source{MaxBytes: 10}).Fetch(context.Background(), srv.URL); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("err = %v, want ErrTooLarge", err)
	}
}

func TestFetchTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		fmt.Fprint(w, "late")
	}))
	defer srv.Close()

	if _, err := (Source{Timeout: 20 * time.Millisecond}).Fetch(context.Background(), srv.URL); err == nil {
		t.Fatal("want timeout error")
	}
}

func TestLoadDispatches(t *testing.T) {
	// URL → Fetch.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "from http")
	}))
	defer srv.Close()
	if got, err := (Source{}).Load(context.Background(), srv.URL); err != nil || got != "from http" {
		t.Fatalf("Load(url) = %q, %v", got, err)
	}

	// Path → Read.
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "f.txt"), []byte("from file"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got, err := (Source{Root: root}).Load(context.Background(), "f.txt"); err != nil || got != "from file" {
		t.Fatalf("Load(path) = %q, %v", got, err)
	}

	// Empty → empty.
	if got, err := (Source{}).Load(context.Background(), "  "); err != nil || got != "" {
		t.Fatalf("Load(empty) = %q, %v", got, err)
	}
}
