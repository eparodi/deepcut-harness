// Package summary is the dashboard landing view at GET /.
package summary

import (
	_ "embed"
	"net/http"

	"deepcut-harness/internal/dashboard/page"
)

//go:embed summary.html
var tmpl string

// Page returns the summary view descriptor.
func Page() page.Page {
	return page.Page{
		Path:     "/{$}",
		Name:     "summary",
		Template: tmpl,
		Handler: func(d page.Deps) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				d.Render(w, r, "summary", page.Base{Title: "Harness", Addr: d.Addr})
			}
		},
	}
}
