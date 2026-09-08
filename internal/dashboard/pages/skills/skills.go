// Package skills is the dashboard view at GET /skills.
package skills

import (
	_ "embed"
	"net/http"

	"deepcut-harness/internal/dashboard/page"
	"deepcut-harness/internal/skill"
)

//go:embed skills.html
var tmpl string

type data struct {
	page.Base
	Skills []skill.Skill
}

// Page returns the skills view descriptor.
func Page() page.Page {
	return page.Page{
		Path:     "/skills",
		Name:     "skills",
		Template: tmpl,
		Handler: func(d page.Deps) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				var list []skill.Skill
				if d.Store != nil {
					var err error
					list, err = d.Store.ListSkills()
					if err != nil {
						d.Log.Error("skills: list", "err", err)
						list = nil
					}
				}
				d.Render(w, r, "skills", data{
					Base:   page.Base{Title: "Skills", Addr: d.Addr},
					Skills: list,
				})
			}
		},
	}
}
