package dashboard

import (
	"log/slog"
	"net/http"
	"runtime"
	"time"
)

// middleware composes the cross-cutting HTTP behavior. Read-only
// routes, so the chain is short: panic recovery, request logging,
// security headers, no-cache.

// recoverPanics logs the panic with a stack trace and re-raises so
// net/http still terminates the connection.
func recoverPanics(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				const size = 64 << 10
				buf := make([]byte, size)
				buf = buf[:runtime.Stack(buf, false)]
				logger.Error("panic while serving request",
					"client", r.RemoteAddr, "url", r.URL.String(),
					"err", err, "stack", string(buf))
				panic(err)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// statusRecorder captures the response status for the access log.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// accessLog logs one line per request at debug level.
func accessLog(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		logger.Debug("http request",
			"method", r.Method, "path", r.URL.Path,
			"status", rec.status,
			"duration", time.Since(start).Round(time.Millisecond))
	})
}

// securityHeaders sets browser hardening headers on every response.
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(w, r)
	})
}

// noCache prevents browsers from caching dynamic pages and API
// responses.
func noCache(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Cache-Control", "no-store, max-age=0")
		h.Set("Pragma", "no-cache")
		next.ServeHTTP(w, r)
	})
}
