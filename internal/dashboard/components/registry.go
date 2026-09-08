// Package components is the folder-per-component frontend registry:
// every component owns a folder with its HTML define, its stylesheet,
// and its enhancement script. The registry concatenates the pieces into
// single bundles at startup — no build step, no npm, go:embed only.
//
// Contract:
//
//	internal/dashboard/components/
//	  foundation.css  tokens + reset + page chrome — NOT a component
//	  <name>/
//	    <name>.html  template define(s) — REQUIRED
//	    <name>.css   component styles — optional
//	    <name>.js    enhancement controller (IIFE) — optional
//
// Bundling order: foundation.css first, then every component stylesheet
// in sorted path order; scripts concatenate the same way. The order is
// pinned by sorting — deterministic for every build. Programmer errors
// (a missing foundation.css or a component folder without its HTML)
// panic at startup, matching the template.Must pattern.
package components

import (
	"embed"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"
)

//go:embed foundation.css */*.css */*.js */*.html
var files embed.FS

// FS exposes the embedded files for template parsing.
func FS() fs.FS { return files }

// HTMLPaths returns every component template file, sorted — the pinned
// order the template engine parses them in. A component directory
// without its own <name>.html, or an HTML file that does not match its
// folder name, is a programmer error: panic at startup.
func HTMLPaths() []string {
	paths := globSorted("*/*.html")
	for _, p := range paths {
		want := path.Join(path.Dir(p), path.Dir(p)+".html")
		if p != want {
			panic(fmt.Sprintf("components: %s must be named %s (contract: <name>/<name>.html)", p, want))
		}
	}
	for _, dir := range componentDirs() {
		want := dir + "/" + dir + ".html"
		if !contains(paths, want) {
			panic(fmt.Sprintf("components: %s/ is missing %s (component html is REQUIRED)", dir, want))
		}
	}
	return paths
}

// CSSBundle concatenates foundation.css followed by every component
// stylesheet in sorted (pinned) order into ONE stylesheet string.
func CSSBundle() string {
	base, err := fs.ReadFile(files, "foundation.css")
	if err != nil {
		panic(fmt.Sprintf("components: read foundation.css: %v", err))
	}
	var b strings.Builder
	b.Write(base)
	for _, p := range globSorted("*/*.css") {
		writePart(&b, p)
	}
	return b.String()
}

// JSBundle concatenates every component script into ONE app.js bundle.
// Each file is already an IIFE — no name collisions, no shared mutable
// state. The bundle opens with the htmx lifecycle dispatcher: components
// that must re-apply to swapped content register init callables via
// window.__dcInit instead of binding DOMContentLoaded (which fires ONCE
// per document). The dispatcher runs every registered init on initial
// load and re-runs them via htmx.onLoad for swapped content.
func JSBundle() []byte {
	var b strings.Builder
	b.WriteString(`/* --- component: htmx-lifecycle (dispatcher) --- */
(function () {
  "use strict";
  var inits = [];
  window.__dcInit = function (fn) { inits.push(fn); };
  function run(root) {
    for (var i = 0; i < inits.length; i++) { inits[i](root); }
  }
  document.addEventListener("DOMContentLoaded", function () { run(document); });
  if (window.htmx) {
    htmx.onLoad(function (content) {
      if (content !== document && content !== document.body) { run(content); }
    });
  }
})();
`)
	for _, p := range globSorted("*/*.js") {
		writePart(&b, p)
	}
	return []byte(b.String())
}

// componentDirs lists the component folders, sorted.
func componentDirs() []string {
	entries, err := fs.ReadDir(files, ".")
	if err != nil {
		panic(fmt.Sprintf("components: read root: %v", err))
	}
	var dirs []string
	for _, e := range entries {
		if e.IsDir() {
			dirs = append(dirs, e.Name())
		}
	}
	sort.Strings(dirs)
	return dirs
}

// globSorted returns embed matches for pattern in sorted order.
func globSorted(pattern string) []string {
	matches, err := fs.Glob(files, pattern)
	if err != nil {
		panic(fmt.Sprintf("components: glob %s: %v", pattern, err))
	}
	sort.Strings(matches)
	return matches
}

// writePart appends one file to the bundle with a marker comment so the
// component boundary is visible in the served bundle.
func writePart(b *strings.Builder, p string) {
	data, err := fs.ReadFile(files, p)
	if err != nil {
		panic(fmt.Sprintf("components: read %s: %v", p, err))
	}
	b.WriteString("\n/* --- component: " + path.Dir(p) + " --- */\n")
	b.Write(data)
	b.WriteString("\n")
}

func contains(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}
