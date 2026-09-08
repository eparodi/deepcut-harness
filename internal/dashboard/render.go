package dashboard

import (
	"log/slog"
	"net/http"
)

// handler serves the shared dashboard surface: the render contract and
// the embedded static assets. Page handlers live in pages/<name>/ as
// free functions over page.Deps — they never branch on HX-Request.
type handler struct {
	templates *templateEngine
	log       *slog.Logger
	addr      string
}

func (h *handler) errLog(msg string, args ...any) {
	h.log.Error(msg, args...)
}

// renderPage is the single render choke point for status 200.
func (h *handler) renderPage(w http.ResponseWriter, r *http.Request, name string, data any) {
	h.renderPageStatus(w, r, name, data, http.StatusOK)
}

// renderPageStatus is the ONE place the HX-Request branch lives (Rule A/B):
// HX-Request: true renders only the swap region; anything else renders the
// full document. A second branching site would drift.
func (h *handler) renderPageStatus(w http.ResponseWriter, r *http.Request, name string, data any, status int) {
	if r.Header.Get("HX-Request") == "true" {
		h.renderPartial(w, name, data, status)
		return
	}
	h.renderFull(w, name, data, status)
}

func (h *handler) renderFull(w http.ResponseWriter, name string, data any, status int) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := h.templates.render(w, name, data); err != nil {
		h.errLog("render failed", "page", name, "err", err)
	}
}

func (h *handler) renderPartial(w http.ResponseWriter, name string, data any, status int) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := h.templates.renderPartial(w, name, data); err != nil {
		if err == errNoMainWrap {
			// A page set without a mainwrap region renders its own full
			// document (login-style) — fall back silently.
			if werr := h.templates.render(w, name, data); werr != nil {
				h.errLog("render failed", "page", name, "err", werr)
			}
			return
		}
		h.errLog("partial render failed", "page", name, "err", err)
	}
}
