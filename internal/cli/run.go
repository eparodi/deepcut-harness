package cli

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"deepcut-harness/internal/config"
	"deepcut-harness/internal/dashboard"
	"deepcut-harness/internal/store"
)

// run starts the Harness dashboard and blocks until SIGINT/SIGTERM.
func run() error {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	editor := config.NewEditor("config.json", ".env")
	cfg, err := editor.Load()
	if err != nil {
		return err
	}
	rt := config.NewRuntime(cfg)

	st, err := store.Open(cfg.Store.Driver, cfg.Store.DSN)
	if err != nil {
		return err
	}
	defer st.Close()

	srv := dashboard.New(cfg, logger, st, rt, editor)
	if err := srv.Start(); err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()
	logger.Info("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}
