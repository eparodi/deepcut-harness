// Package app is the dashboard workspace view at GET /app.
package app

import (
	_ "embed"
	"net/http"

	"deepcut-harness/internal/dashboard/page"
)

//go:embed app.html
var tmpl string

// Page returns the app (workspace) view descriptor.
func Page() page.Page {
	return page.Page{
		Path:     "/app",
		Name:     "app",
		Template: tmpl,
		Handler: func(d page.Deps) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				d.Render(w, r, "app", page.Base{Title: "Workspace", Addr: d.Addr})
			}
		},
	}
}
