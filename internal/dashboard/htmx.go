package dashboard

import (
	"embed"
	"io/fs"
	"net/http"
)

// htmxFS embeds the build-time-fetched htmx.min.js (tools/fetchhtmx).
// The file is gitignored and REQUIRED: a plain `go build` on a fresh
// clone fails at this embed directive until `make htmx` (or any build
// target that depends on it) runs. That loud failure is the delivery
// contract — no CDN at runtime, no committed library copy.
//
//go:embed htmx/*
var htmxFS embed.FS

// htmxMinJS reads the embedded htmx library.
func htmxMinJS() ([]byte, error) {
	return fs.ReadFile(htmxFS, "htmx/htmx.min.js")
}

// handleHTMXJS serves the embedded htmx library (handleAppJS mirror:
// GET-only, explicit content type, no-store).
func (h *handler) handleHTMXJS(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	data, err := htmxMinJS()
	if err != nil {
		h.errLog("dashboard: htmx read failed", "err", err)
		http.Error(w, "htmx unavailable — run make htmx", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(data)
}
