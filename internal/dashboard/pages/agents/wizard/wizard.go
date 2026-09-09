// Package wizard is the dashboard view at /agents/wizard: the LLM-driven
// creation form for a new Agent.
package wizard

import (
	_ "embed"
	"net/http"

	"deepcut-harness/internal/dashboard/page"
	"deepcut-harness/internal/wizard"
)

//go:embed wizard.html
var tmpl string

type data struct {
	page.Base
	Wizard page.WizardState
}

// Page returns the /agents/wizard view descriptor.
func Page() page.Page {
	return page.Page{
		Path:     "/agents/wizard",
		Name:     "agents-wizard",
		Template: tmpl,
		Handler: func(d page.Deps) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				base := page.Base{Title: "New Agent", Addr: d.Addr}
				wstate := page.WizardState{Target: wizard.TargetAgent, Providers: d.WizardProviders(), Models: d.WizardModels()}
				if r.Method == http.MethodPost {
					wstate = d.WizardPost(r, wizard.TargetAgent)
				}
				d.Render(w, r, "agents-wizard", data{Base: base, Wizard: wstate})
			}
		},
	}
}
