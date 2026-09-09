// Package agents is the dashboard view at GET /agents.
package agents

import (
	_ "embed"
	"net/http"

	"deepcut-harness/internal/agent"
	"deepcut-harness/internal/dashboard/page"
)

//go:embed agents.html
var tmpl string

type data struct {
	page.Base
	Agents []agent.Agent
}

// Page returns the agents view descriptor.
func Page() page.Page {
	return page.Page{
		Path:     "/agents",
		Name:     "agents",
		Template: tmpl,
		Handler: func(d page.Deps) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				var list []agent.Agent
				if d.Store != nil {
					var err error
					list, err = d.Store.ListAgents()
					if err != nil {
						d.Log.Error("agents: list", "err", err)
						list = nil
					}
				}
				d.Render(w, r, "agents", data{
					Base:   page.Base{Title: "Agents", Addr: d.Addr},
					Agents: list,
				})
			}
		},
	}
}
