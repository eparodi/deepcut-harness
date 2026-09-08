// Command fetchhtmx downloads the PINNED htmx release into the
// dashboard embed directory (go-htmx delivery contract: no CDN at
// runtime, no committed library copy). The version and the SHA-256 are
// constants; the download is checksum-verified before it is written; the
// destination file is gitignored and required by the dashboard's
// //go:embed htmx/* (a missing fetch fails the build loudly, by design).
//
//	go run ./tools/fetchhtmx
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

const (
	// version is pinned from the official release list
	// (github.com/bigskysoftware/htmx/releases) — bump deliberately, then
	// re-pin pinnedSHA256 below with the measured value from a dry run.
	// v4.0.0 is the current major; the docs website lags releases, so the
	// pin is taken from the GitHub tags, never the docs' CDN examples.
	version = "4.0.0"
	// pinnedSHA256 is the SHA-256 of htmx.min.js for the pinned release,
	// measured from the official v4.0.0 tag (dist/htmx.min.js).
	pinnedSHA256 = "e484d9171a9db30a39c8f16e3d709d4137f3211c659f8e6125816635033d593f"
)

const downloadURL = "https://raw.githubusercontent.com/bigskysoftware/htmx/v" + version + "/dist/htmx.min.js"

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "fetchhtmx:", err)
		os.Exit(1)
	}
}

func run() error {
	resp, err := http.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("download %s: %w", downloadURL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download %s: status %s", downloadURL, resp.Status)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}
	sum := sha256.Sum256(data)
	got := hex.EncodeToString(sum[:])
	if pinnedSHA256 == "" {
		// First-run bootstrap: print the measured checksum so it can be
		// pinned; refuse to write an unverified file.
		return fmt.Errorf("no checksum pinned — verify %s from the official release, then set pinnedSHA256 = %q in tools/fetchhtmx/main.go", downloadURL, got)
	}
	if got != pinnedSHA256 {
		return fmt.Errorf("checksum mismatch: got %s, want %s — do NOT proceed; the download may be tampered", got, pinnedSHA256)
	}
	outDir := filepath.Join("internal", "dashboard", "htmx")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", outDir, err)
	}
	out := filepath.Join(outDir, "htmx.min.js")
	if err := os.WriteFile(out, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", out, err)
	}
	fmt.Printf("wrote %s (%d bytes, sha256 %s) — htmx v%s\n", out, len(data), got, version)
	return nil
}
