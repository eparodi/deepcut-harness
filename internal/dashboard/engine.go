package dashboard

import (
	"embed"
	"fmt"
	"html/template"
	"io"

	"deepcut-harness/internal/dashboard/components"
)

//go:embed layout/*.html
var layoutFS embed.FS

// templateEngine parses each page's template set once at startup and
// renders through the shared base layout. It also carries the
// startup-built component bundles (one stylesheet, one app.js).
type templateEngine struct {
	funcs     template.FuncMap
	compPaths []string
	css       template.CSS
	js        []byte
	templates map[string]*template.Template
}

func newTemplateEngine() (*templateEngine, error) {
	cssBundle := template.CSS(components.CSSBundle())
	return &templateEngine{
		funcs:     template.FuncMap{"cssBundle": func() template.CSS { return cssBundle }},
		compPaths: components.HTMLPaths(),
		css:       cssBundle,
		js:        components.JSBundle(),
		templates: map[string]*template.Template{},
	}, nil
}

// add parses one page into its own template set: the shared layout, the
// component defines, then the page's own content define. It fails if the
// page template is missing its {{define "content"}} block — otherwise the
// page would render silently empty.
func (e *templateEngine) add(name, pageTmpl string) error {
	t := template.New(name).Funcs(e.funcs)
	for _, dep := range []string{"layout/style.html", "layout/layout.html"} {
		if _, err := t.ParseFS(layoutFS, dep); err != nil {
			return fmt.Errorf("dashboard: parse %s: %w", dep, err)
		}
	}
	for _, cp := range e.compPaths {
		if _, err := t.ParseFS(components.FS(), cp); err != nil {
			return fmt.Errorf("dashboard: parse component %s: %w", cp, err)
		}
	}
	if _, err := t.Parse(pageTmpl); err != nil {
		return fmt.Errorf("dashboard: parse page %q: %w", name, err)
	}
	if t.Lookup("content") == nil {
		return fmt.Errorf("dashboard: page %q is missing its {{define \"content\"}} block", name)
	}
	e.templates[name] = t
	return nil
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
