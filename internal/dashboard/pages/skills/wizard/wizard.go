// Package wizard is the dashboard view at /skills/wizard: the LLM-driven
// creation form for a new Skill.
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

// Page returns the /skills/wizard view descriptor.
func Page() page.Page {
	return page.Page{
		Path:     "/skills/wizard",
		Name:     "skills-wizard",
		Template: tmpl,
		Handler: func(d page.Deps) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				base := page.Base{Title: "New Skill", Addr: d.Addr}
				wstate := page.WizardState{Target: wizard.TargetSkill, Providers: d.WizardProviders(), Models: d.WizardModels()}
				if r.Method == http.MethodPost {
					wstate = d.WizardPost(r, wizard.TargetSkill)
				}
				d.Render(w, r, "skills-wizard", data{Base: base, Wizard: wstate})
			}
		},
	}
}
