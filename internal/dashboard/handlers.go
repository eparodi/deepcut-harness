package dashboard

import (
	"log/slog"
	"net/http"

	"deepcut-harness/internal/config"
)

// handler serves pages. Read-only for now: every route is GET and none
// mutate state.
type handler struct {
	cfg       config.Config
	templates *templateEngine
	log       *slog.Logger
	addr      string
}

func (h *handler) errLog(msg string, args ...any) {
	h.log.Error(msg, args...)
}

// basePage is the shared page data for every layout render.
type basePage struct {
	Title string
	Addr  string
}

// ---- pages ----

// handleSummary renders the dashboard landing page.
func (h *handler) handleSummary(w http.ResponseWriter, r *http.Request) {
	h.renderPage(w, r, "summary", basePage{Title: "Harness", Addr: h.addr})
}

// handleNotFound renders the layout 404 for unknown HTML paths.
func (h *handler) handleNotFound(w http.ResponseWriter, r *http.Request) {
	h.renderPageStatus(w, r, "error", basePage{Title: "Not found", Addr: h.addr}, http.StatusNotFound)
}

// ---- render contract (Rule A/B) ----

// renderPage is the single render choke point. HX-Request: true renders
// only the swap region; anything else renders the full document. This is
// the ONE place the branch lives — a second branching site would drift.
func (h *handler) renderPage(w http.ResponseWriter, r *http.Request, name string, data any) {
	h.renderPageStatus(w, r, name, data, http.StatusOK)
}

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
