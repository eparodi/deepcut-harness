// Package dashboard is the embedded Harness HTTP dashboard. It renders
// html/template pages with the go-htmx partial-swap contract (Rule A/B):
// a direct load returns the full document, an HX-Request returns only
// the swap region. Views are folder-per-view (see AGENTS.md); htmx.min.js
// is fetched at build time (tools/fetchhtmx) and embedded — no CDN at
// runtime.
package dashboard

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"deepcut-harness/internal/config"
)

// Server is the embedded dashboard HTTP server. Start binds the
// configured listen address (returning bind errors synchronously) and
// serves in a goroutine; Shutdown drains connections.
type Server struct {
	logger *slog.Logger
	addr   string
	srv    *http.Server
}

// New wires the dashboard: parses the embedded templates (panicking on
// programmer error, like template.Must), registers the page routes and
// the static assets, and assembles the middleware chain.
func New(cfg config.Config, logger *slog.Logger) *Server {
	if logger == nil {
		logger = slog.Default()
	}
	s := &Server{logger: logger, addr: cfg.Dashboard.ListenAddr}

	engine, err := newTemplateEngine()
	if err != nil {
		// Template errors are programmer errors, not runtime errors:
		// fail loudly at startup like template.Must.
		panic(err)
	}
	h := &handler{templates: engine, log: logger, addr: cfg.Dashboard.ListenAddr}

	mux := http.NewServeMux()
	registerPages(mux, h)
	mux.HandleFunc("/static/htmx.min.js", h.handleHTMXJS)
	mux.HandleFunc("/static/app.js", h.handleAppJS)
	mux.HandleFunc("/favicon.svg", h.handleFavicon)

	// Middleware chain: recovery outermost, logging, headers, no-cache.
	var root http.Handler = mux
	root = noCache(root)
	root = securityHeaders(root)
	root = accessLog(logger, root)
	root = recoverPanics(logger, root)

	s.srv = &http.Server{
		Handler:           root,
		ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelError),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      20 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 16,
	}
	return s
}

// Start binds the listen address and serves in a goroutine. Bind errors
// (port in use, bad address) return synchronously so the process can
// fail fast at startup.
func (s *Server) Start() error {
	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("dashboard: listen %s: %w", s.addr, err)
	}
	s.addr = ln.Addr().String() // record the actual address (ephemeral ports in tests)
	s.logger.Info("dashboard listening", "addr", s.addr)
	go func() {
		if err := s.srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			s.logger.Warn("dashboard server stopped unexpectedly", "err", err)
		}
	}()
	return nil
}

// Addr returns the bound listen address (valid after Start).
func (s *Server) Addr() string {
	return s.addr
}

// Shutdown gracefully stops the server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.srv.Shutdown(ctx)
}
