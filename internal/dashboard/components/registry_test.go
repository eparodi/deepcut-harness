package components

import (
	"strings"
	"testing"
)

func TestHTMLPathsPinned(t *testing.T) {
	got := HTMLPaths()
	want := []string{"status-badge/status-badge.html", "wizard/wizard.html"}
	if len(got) != len(want) {
		t.Fatalf("HTMLPaths = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("HTMLPaths[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestCSSBundleIncludesFoundationAndComponents(t *testing.T) {
	css := CSSBundle()
	if !strings.HasPrefix(css, "/* foundation.css") {
		t.Error("CSSBundle must start with foundation.css")
	}
	if !strings.Contains(css, "status-badge") {
		t.Error("CSSBundle missing status-badge component css")
	}
}

func TestJSBundleIncludesDispatcherAndComponents(t *testing.T) {
	js := string(JSBundle())
	if !strings.Contains(js, "__dcInit") {
		t.Error("JSBundle missing the htmx lifecycle dispatcher")
	}
	if !strings.Contains(js, "status-badge") {
		t.Error("JSBundle missing status-badge component js")
	}
}
