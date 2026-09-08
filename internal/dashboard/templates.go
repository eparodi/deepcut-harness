package dashboard

import (
	"embed"
	"fmt"
	"html/template"
	"io"
	"io/fs"

	"deepcut-harness/internal/dashboard/components"
)

//go:embed templates/*.html
var templateFS embed.FS

// pageTemplates declares every embedded page template and the layout
// files it depends on. Each page is parsed into its own template set so
// every page can override the shared "content" block without colliding.
var pageTemplates = map[string][]string{
	"summary": {"style.html", "layout.html"},
	"error":   {"style.html", "layout.html"},
}

// templateEngine parses the embedded templates once at startup and
// renders pages through the shared base layout. It also carries the
// startup-built component bundles: one stylesheet and one app.js,
// concatenated by the component registry.
type templateEngine struct {
	templates map[string]*template.Template
	css       template.CSS // concatenated component stylesheet
	js        []byte       // concatenated enhancement bundle (app.js)
}

// newTemplateEngine parses every page with its dependencies and fails
// fast (like template.Must) if any template is broken or if a template
// file exists that was not declared in pageTemplates — a startup sanity
// check so a typo never ships silently.
func newTemplateEngine() (*templateEngine, error) {
	cssBundle := template.CSS(components.CSSBundle())
	funcs := template.FuncMap{
		"cssBundle": func() template.CSS { return cssBundle },
	}

	e := &templateEngine{
		templates: map[string]*template.Template{},
		css:       cssBundle,
		js:        components.JSBundle(),
	}
	// Every component define is parsed into EVERY page's template set
	// (defines are inert until invoked).
	compPaths := components.HTMLPaths()

	for name, deps := range pageTemplates {
		t := template.New(name).Funcs(funcs)
		for _, dep := range deps {
			if _, err := t.ParseFS(templateFS, "templates/"+dep); err != nil {
				return nil, fmt.Errorf("dashboard: parse %s: %w", dep, err)
			}
		}
		for _, cp := range compPaths {
			if _, err := t.ParseFS(components.FS(), cp); err != nil {
				return nil, fmt.Errorf("dashboard: parse component %s: %w", cp, err)
			}
		}
		if _, err := t.ParseFS(templateFS, "templates/"+name+".html"); err != nil {
			return nil, fmt.Errorf("dashboard: parse %s.html: %w", name, err)
		}
		e.templates[name] = t
	}

	// Sanity check: every embedded template file must be declared as a
	// page or as a dependency of some page.
	declared := map[string]bool{}
	for name, deps := range pageTemplates {
		declared[name+".html"] = true
		for _, dep := range deps {
			declared[dep] = true
		}
	}
	entries, err := fs.ReadDir(templateFS, "templates")
	if err != nil {
		return nil, fmt.Errorf("dashboard: read templates dir: %w", err)
	}
	for _, entry := range entries {
		if !declared[entry.Name()] {
			return nil, fmt.Errorf("dashboard: template %s is not declared in pageTemplates", entry.Name())
		}
	}
	return e, nil
}

// render executes the page's base layout with the page data.
func (e *templateEngine) render(w io.Writer, name string, data any) error {
	t, ok := e.templates[name]
	if !ok {
		return fmt.Errorf("dashboard: unknown template %q", name)
	}
	return t.ExecuteTemplate(w, "base", data)
}

// errNoMainWrap marks a page set that renders its own base without the
// shared "mainwrap" region: partial rendering falls back to the full
// document silently — a design property, not an error.
var errNoMainWrap = fmt.Errorf("dashboard: template has no mainwrap region")

// renderPartial executes ONLY the swap region (the "mainwrap" block) —
// go-htmx Rule B. The partial must equal the ENTIRE region the swap
// target replaces; the region covers everything inside .main-wrap.
func (e *templateEngine) renderPartial(w io.Writer, name string, data any) error {
	t, ok := e.templates[name]
	if !ok {
		return fmt.Errorf("dashboard: unknown template %q", name)
	}
	if t.Lookup("mainwrap") == nil {
		return errNoMainWrap
	}
	return t.ExecuteTemplate(w, "mainwrap", data)
}
