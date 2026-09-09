package dashboard

import (
	"net/http"

	"deepcut-harness/internal/config"
	"deepcut-harness/internal/dashboard/page"
	"deepcut-harness/internal/dashboard/pages/agents"
	"deepcut-harness/internal/dashboard/pages/app"
	errpage "deepcut-harness/internal/dashboard/pages/error"
	"deepcut-harness/internal/dashboard/pages/settings"
	"deepcut-harness/internal/dashboard/pages/skills"
	"deepcut-harness/internal/dashboard/pages/summary"
	"deepcut-harness/internal/store"
)

// registerPages parses each page's template into the engine and registers
// its route. The catch-all 404 page is listed last so it only claims
// paths no other page handles.
func registerPages(mux *http.ServeMux, h *handler, st store.Store, rt *config.Runtime, editor *config.Editor) {
	deps := page.Deps{
		Addr:         h.addr,
		Log:          h.log,
		Store:        st,
		Runtime:      rt,
		Editor:       editor,
		Render:       h.renderPage,
		RenderStatus: h.renderPageStatus,
	}
	pages := []page.Page{
		summary.Page(),
		app.Page(),
		agents.Page(),
		skills.Page(),
		settings.Page(),
		errpage.Page(),
	}
	for _, p := range pages {
		if err := h.templates.add(p.Name, p.Template); err != nil {
			// Template errors are programmer errors: fail fast like template.Must.
			panic(err)
		}
		mux.HandleFunc(p.Path, p.Handler(deps))
	}
}
