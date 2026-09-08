package dashboard

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"deepcut-harness/internal/config"
)

func newTestHandler(t *testing.T) http.Handler {
	t.Helper()
	return New(config.Default(), nil).srv.Handler
}

// TestSummaryRenderContract pins the go-htmx Rule A/B contract: a direct
// load returns the full document; an HX-Request returns only the swap
// region (no doctype, no script tags, no title).
func TestSummaryRenderContract(t *testing.T) {
	handler := newTestHandler(t)
	tests := []struct {
		name    string
		hx      bool
		want    []string
		notWant []string
	}{
		{
			name: "full document",
			hx:   false,
			want: []string{
				"<!DOCTYPE html>",
				`<script src="/static/htmx.min.js" defer></script>`,
				`<script src="/static/app.js" defer></script>`,
				`<title>Harness</title>`,
				`<meta name="htmx-config" content='{"history":"reload"}'>`,
			},
		},
		{
			name: "partial swap region only",
			hx:   true,
			want: []string{"<main>", "status-badge", "running"},
			notWant: []string{
				"<!DOCTYPE html>",
				"<script",
				"<title>",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.hx {
				req.Header.Set("HX-Request", "true")
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", rec.Code)
			}
			body := rec.Body.String()
			for _, want := range tt.want {
				if !strings.Contains(body, want) {
					t.Errorf("body missing %q\n--- body ---\n%s", want, body)
				}
			}
			for _, not := range tt.notWant {
				if strings.Contains(body, not) {
					t.Errorf("body unexpectedly contains %q\n--- body ---\n%s", not, body)
				}
			}
		})
	}
}

// TestPartialMatchesFullRegion pins that the partial is byte-identical
// to the full document's swap region (extracted between the .main-wrap
// opening and the script tags that live outside it).
func TestPartialMatchesFullRegion(t *testing.T) {
	engine, err := newTemplateEngine()
	if err != nil {
		t.Fatal(err)
	}
	data := basePage{Title: "Harness", Addr: "127.0.0.1:8787"}

	var full, partial bytes.Buffer
	if err := engine.render(&full, "summary", data); err != nil {
		t.Fatal(err)
	}
	if err := engine.renderPartial(&partial, "summary", data); err != nil {
		t.Fatal(err)
	}

	fullStr := full.String()
	partStr := partial.String()

	const open = `<div class="main-wrap">`
	// The region ends at the </div> that closes .main-wrap, which sits
	// immediately before the script tags that live OUTSIDE the region.
	const closeAnchor = `</div>
<script src="/static/htmx.min.js" defer>`
	start := strings.Index(fullStr, open)
	end := strings.Index(fullStr, closeAnchor)
	if start < 0 || end < 0 || end < start {
		t.Fatalf("could not locate swap region in full render")
	}
	region := fullStr[start+len(open) : end]
	if strings.TrimSpace(region) != strings.TrimSpace(partStr) {
		t.Errorf("partial != full page region\n--- partial ---\n%s\n--- region ---\n%s", partStr, region)
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
	req := httptest.NewRequest(http.MethodGet, "/nope", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Not found") {
		t.Fatal("404 body missing the error page")
	}
}
