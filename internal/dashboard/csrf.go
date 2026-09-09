package dashboard

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"net/http"
)

// newCSRFToken returns a cryptographically random synchronizer token for
// the dashboard's mutating forms. Harness is single-user (loopback), so
// one token per process is sufficient: the token is unpredictable to a
// cross-site attacker and embedded in every form, so a forged
// cross-origin POST cannot reproduce it.
func newCSRFToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand failure is unrecoverable — fail fast at startup.
		panic("dashboard: crypto/rand failed: " + err.Error())
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

// csrfProtect rejects state-changing requests whose csrf form value does
// not match the process token (constant-time compare). Safe methods
// (GET/HEAD/OPTIONS) pass through unchanged.
func csrfProtect(token string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isSafeMethod(r.Method) {
			next.ServeHTTP(w, r)
			return
		}
		got := r.FormValue("csrf")
		if subtle.ConstantTimeCompare([]byte(got), []byte(token)) != 1 {
			http.Error(w, "invalid CSRF token", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func isSafeMethod(m string) bool {
	switch m {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	default:
		return false
	}
}
