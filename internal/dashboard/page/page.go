// Package page defines the folder-per-view contract for the dashboard.
// It is a leaf package — it imports only net/http, log/slog, and the
// store/domain packages, never dashboard — so page subpackages can
// import it without creating an import cycle
// (dashboard → pages/<name> → page).
package page

import (
	"log/slog"
	"net/http"

	"deepcut-harness/internal/config"
	"deepcut-harness/internal/llm"
	"deepcut-harness/internal/source"
	"deepcut-harness/internal/store"
	"deepcut-harness/internal/wizard"
)

// Base is the common page data every view's template relies on: the
// shared layout reads .Title and .Addr. Pages embed it and add their own
// fields.
type Base struct {
	Title string
	Addr  string
}

// Deps are the shared dependencies a page handler needs, injected by the
// dashboard at registration time so a page handler never reaches into
// dashboard internals — and never branches on HX-Request itself.
type Deps struct {
	Addr string
	Log  *slog.Logger
	// Store is the persistence backend (nil when pages are rendered in
	// tests without a store — read-only pages tolerate nil).
	Store store.Store
	// Runtime is the live config holder; the settings page reads and
	// swaps it on save.
	Runtime *config.Runtime
	// Editor persists config + .env changes (settings page).
	Editor *config.Editor
	// LLM returns an LLM JSON-mode completer for a provider name.
	LLM func(name string) (llm.JSONCompleter, error)
	// Source is the base source reader (fetch config); the workspace Root
	// is set per session by the chat feature.
	Source source.Source
	// Wizard manages in-memory wizard sessions.
	Wizard *wizard.Manager
	// Render executes the go-htmx render contract with status 200.
	Render func(w http.ResponseWriter, r *http.Request, name string, data any)
	// RenderStatus is Render with an explicit status (the 404 page).
	RenderStatus func(w http.ResponseWriter, r *http.Request, name string, data any, status int)
}

// Page is one view: its route path, its template name, its embedded
// template text, and a handler factory that receives Deps.
type Page struct {
	Path     string
	Name     string
	Template string
	Handler  func(Deps) http.HandlerFunc
}
