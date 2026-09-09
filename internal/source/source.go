// Package source is the read-only source seam the wizard (and future
// agents) use to read local files and fetch URLs. It is deliberately
// minimal and pluggable: later connectors (databases, code hosts, MCP,
// OpenAPI) attach behind the same Reader interface.
package source

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// DefaultMaxBytes is the size cap applied when Source.MaxBytes is 0.
const DefaultMaxBytes int64 = 1 << 20 // 1 MiB

// DefaultTimeout is the Fetch timeout applied when Source.Timeout is 0.
const DefaultTimeout = 10 * time.Second

// ErrTooLarge is returned when a read source exceeds its size cap.
var ErrTooLarge = errors.New("source: content exceeds max size")

// Reader is the read-only source seam. Implementations read a local file
// (Read) and fetch a URL (Fetch).
type Reader interface {
	Read(ctx context.Context, path string) (string, error)
	Fetch(ctx context.Context, url string) (string, error)
}

// Source is the default Reader: it reads files constrained to a workspace
// root and fetches URLs over HTTP(S) with size, timeout, and SSRF guards.
type Source struct {
	// Root is the workspace root for Read; empty means the process CWD.
	Root string
	// MaxBytes caps a single Read/Fetch result (0 = DefaultMaxBytes).
	MaxBytes int64
	// Timeout bounds a single Fetch (0 = DefaultTimeout).
	Timeout time.Duration
	// BlockPrivateHosts rejects private/loopback/link-local hosts on Fetch.
	BlockPrivateHosts bool
	// Client is the HTTP client for Fetch; nil uses a default with Timeout.
	Client *http.Client
}

// Read returns the contents of path, resolved against Root with no `..`
// escape, no absolute path, no symlink escape, and a size cap.
func (s Source) Read(ctx context.Context, path string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	root := s.Root
	if root == "" {
		var err error
		if root, err = os.Getwd(); err != nil {
			return "", fmt.Errorf("source: get cwd: %w", err)
		}
	}
	real, err := resolveWithin(root, path)
	if err != nil {
		return "", err
	}
	data, err := readFileCapped(real, s.maxBytes())
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// Load reads a reference by local path or http(s) URL, dispatching on the
// scheme. Empty input yields an empty result (no source). It is the shared
// helper the wizard uses to read one --source value.
func (s Source) Load(ctx context.Context, ref string) (string, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", nil
	}
	if strings.HasPrefix(ref, "http://") || strings.HasPrefix(ref, "https://") {
		return s.Fetch(ctx, ref)
	}
	return s.Read(ctx, ref)
}

// maxBytes returns the effective size cap.
func (s Source) maxBytes() int64 {
	if s.MaxBytes > 0 {
		return s.MaxBytes
	}
	return DefaultMaxBytes
}

// resolveWithin resolves path against root and verifies it stays inside
// root. It rejects absolute paths and `..`/symlink escapes.
func resolveWithin(root, path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", errors.New("source: empty path")
	}
	if filepath.IsAbs(path) {
		return "", fmt.Errorf("source: absolute path %q not allowed", path)
	}
	joined := filepath.Join(root, path)
	real, err := filepath.EvalSymlinks(joined)
	if err != nil {
		return "", fmt.Errorf("source: resolve %q: %w", path, err)
	}
	rootReal, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", fmt.Errorf("source: resolve root %q: %w", root, err)
	}
	rel, err := filepath.Rel(rootReal, real)
	if err != nil {
		return "", fmt.Errorf("source: rel: %w", err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("source: path %q escapes workspace root", path)
	}
	return real, nil
}

// readFileCapped reads at most max+1 bytes and errors if the file exceeds
// max (loud, not silent truncation).
func readFileCapped(path string, max int64) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("source: open %s: %w", path, err)
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, max+1))
	if err != nil {
		return nil, fmt.Errorf("source: read %s: %w", path, err)
	}
	if int64(len(data)) > max {
		return nil, ErrTooLarge
	}
	return data, nil
}
