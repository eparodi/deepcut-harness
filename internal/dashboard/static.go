package dashboard

import (
	"net/http"
)

// handleAppJS serves the concatenated enhancement bundle (app.js),
// assembled once at startup by the component registry: one request, no
// build step, each component script is an IIFE.
func (h *handler) handleAppJS(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(h.templates.js)
}
