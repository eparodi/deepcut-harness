// Package errpage is the dashboard's catch-all 404 view.
//
// The folder is pages/error/ but the package is errpage to avoid
// shadowing the predeclared error type.
package errpage

import (
	_ "embed"
	"net/http"

	"deepcut-harness/internal/dashboard/page"
)

//go:embed error.html
var tmpl string

// Page returns the 404 error view descriptor. Its route is the
// catch-all "/", registered last so it only claims paths no other page
// handles.
func Page() page.Page {
	return page.Page{
		Path:     "/",
		Name:     "error",
		Template: tmpl,
		Handler: func(d page.Deps) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				d.RenderStatus(w, r, "error", page.Base{Title: "Not found", Addr: d.Addr}, http.StatusNotFound)
			}
		},
	}
}
