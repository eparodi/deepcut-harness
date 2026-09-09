package dashboard

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"deepcut-harness/internal/config"
	"deepcut-harness/internal/llm/openai"
	"deepcut-harness/internal/source"
	"deepcut-harness/internal/store"
)

func newTestHandler(t *testing.T) http.Handler {
	t.Helper()
	st, err := store.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	dir := t.TempDir()
	rt := config.NewRuntime(config.Default())
	editor := config.NewEditor(filepath.Join(dir, "config.json"), filepath.Join(dir, ".env"))
	reg := openai.NewRegistry(config.Default())
	return New(config.Default(), nil, st, rt, editor, reg, source.Source{}).srv.Handler
}

// get issues a GET and returns the status code and body.
func get(t *testing.T, h http.Handler, path string, hx bool) (int, string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if hx {
		req.Header.Set("HX-Request", "true")
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec.Code, rec.Body.String()
}

// TestRenderContract pins the go-htmx Rule A/B contract for every page: a
// direct load returns the full document; an HX-Request returns only the
// swap region (no doctype, no script tags, no title).
func TestRenderContract(t *testing.T) {
	handler := newTestHandler(t)
	pages := []struct {
		name    string
		path    string
		title   string
		heading string
	}{
		{name: "summary", path: "/", title: "Harness", heading: "Harness"},
		{name: "app", path: "/app", title: "Workspace", heading: "Workspace"},
		{name: "agents", path: "/agents", title: "Agents", heading: "Agents"},
		{name: "skills", path: "/skills", title: "Skills", heading: "Skills"},
		{name: "agents-wizard", path: "/agents/wizard", title: "New Agent", heading: "New Agent"},
		{name: "skills-wizard", path: "/skills/wizard", title: "New Skill", heading: "New Skill"},
		{name: "settings", path: "/settings", title: "Settings", heading: "Settings"},
	}
	for _, p := range pages {
		t.Run(p.name+" full", func(t *testing.T) {
			code, body := get(t, handler, p.path, false)
			if code != http.StatusOK {
				t.Fatalf("status = %d, want 200", code)
			}
			for _, want := range []string{
				"<!DOCTYPE html>",
				`<script src="/static/htmx.min.js" defer></script>`,
				`<script src="/static/app.js" defer></script>`,
				`<title>` + p.title + `</title>`,
				`<meta name="htmx-config" content='{"history":"reload"}'>`,
				p.heading,
			} {
				if !strings.Contains(body, want) {
					t.Errorf("full body missing %q", want)
				}
			}
		})
		t.Run(p.name+" partial", func(t *testing.T) {
			code, body := get(t, handler, p.path, true)
			if code != http.StatusOK {
				t.Fatalf("status = %d, want 200", code)
			}
			for _, want := range []string{"<main>", p.heading} {
				if !strings.Contains(body, want) {
					t.Errorf("partial body missing %q", want)
				}
			}
			for _, not := range []string{"<!DOCTYPE html>", "<script", "<title>"} {
				if strings.Contains(body, not) {
					t.Errorf("partial body unexpectedly contains %q", not)
				}
			}
		})
	}
}

// TestPartialMatchesFullRegion pins that the partial is byte-identical to
// the full document's swap region (extracted between the .main-wrap
// opening and the script tags that live outside it).
func TestPartialMatchesFullRegion(t *testing.T) {
	handler := newTestHandler(t)
	_, full := get(t, handler, "/", false)
	_, partial := get(t, handler, "/", true)

	const open = `<div class="main-wrap">`
	// The region ends at the </div> that closes .main-wrap, immediately
	// before the script tags that live OUTSIDE the region.
	const closeAnchor = `</div>
<script src="/static/htmx.min.js" defer>`
	start := strings.Index(full, open)
	end := strings.Index(full, closeAnchor)
	if start < 0 || end < 0 || end < start {
		t.Fatalf("could not locate swap region in full render")
	}
	region := full[start+len(open) : end]
	if strings.TrimSpace(region) != strings.TrimSpace(partial) {
		t.Errorf("partial != full page region\n--- partial ---\n%s\n--- region ---\n%s", partial, region)
	}
}

// TestHTMXServed pins that the embedded htmx library is served with the
// right content type and content.
func TestHTMXServed(t *testing.T) {
	handler := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/static/htmx.min.js", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/javascript") {
		t.Fatalf("content-type = %q, want text/javascript", ct)
	}
	if !strings.Contains(rec.Body.String(), "htmx") {
		t.Fatal("htmx.min.js body missing htmx content")
	}
}

// TestNotFoundRendersErrorPage pins the 404 layout page.
func TestNotFoundRendersErrorPage(t *testing.T) {
	handler := newTestHandler(t)
	code, body := get(t, handler, "/nope", false)
	if code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", code)
	}
	if !strings.Contains(body, "Not found") {
		t.Fatal("404 body missing the error page")
	}
}

// TestWizardModelsEndpoint pins the /wizard-models fragment: it returns the
// provider's models and 404s for an unknown provider.
func TestWizardModelsEndpoint(t *testing.T) {
	handler := newTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/wizard-models?provider=deepseek", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if body := rec.Body.String(); !strings.Contains(body, "deepseek-v4-flash") || !strings.Contains(body, "deepseek-v4-pro") {
		t.Fatalf("body missing models: %s", body)
	}

	req = httptest.NewRequest(http.MethodGet, "/wizard-models?provider=nope", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("unknown provider status = %d, want 404", rec.Code)
	}
}
