package dashboard

import "net/http"

// faviconSVG is a minimal inline brand mark served at /favicon.svg so
// browsers never 404 the icon on a local-first install.
const faviconSVG = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 32 32">
<rect width="32" height="32" rx="7" fill="#1a2029"/>
<text x="16" y="22" font-family="system-ui,sans-serif" font-size="16" font-weight="700" fill="#ffffff" text-anchor="middle">H</text>
</svg>`

// handleFavicon serves the embedded SVG favicon.
func (h *handler) handleFavicon(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "image/svg+xml")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write([]byte(faviconSVG))
}
