package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/n1ckerr0r/avito-labs/api"
	"github.com/n1ckerr0r/avito-labs/internal/config"
	"github.com/n1ckerr0r/avito-labs/internal/database"
	"github.com/n1ckerr0r/avito-labs/internal/httpapi"
	"github.com/n1ckerr0r/avito-labs/internal/repository"
	"github.com/n1ckerr0r/avito-labs/internal/service"
)

func main() {
	if err := run(); err != nil {
		slog.Error("trip service startup failed", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}

	logLevel := new(slog.LevelVar)
	switch cfg.LogLevel {
	case "debug":
		logLevel.Set(slog.LevelDebug)
	case "warn":
		logLevel.Set(slog.LevelWarn)
	case "error":
		logLevel.Set(slog.LevelError)
	default:
		logLevel.Set(slog.LevelInfo)
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel})))

	pool, err := database.OpenPool(context.Background(), cfg)
	if err != nil {
		return fmt.Errorf("connect to PostgreSQL: %w", err)
	}
	defer pool.Close()

	repo := repository.NewTrips(pool, cfg.DatabaseQueryTimeout)
	trips := service.NewTrips(repo, database.NewTxManager(pool))
	handler := httpapi.New(trips, pool, cfg.DatabaseQueryTimeout)
	mux := api.HandlerWithOptions(handler, api.ChiServerOptions{ErrorHandlerFunc: httpapi.ErrorHandler})
	server := &http.Server{Addr: cfg.HTTPAddr, Handler: mux, ReadTimeout: cfg.HTTPReadTimeout, ReadHeaderTimeout: cfg.HTTPReadHeaderTimeout, WriteTimeout: cfg.HTTPWriteTimeout, IdleTimeout: cfg.HTTPIdleTimeout}

	errCh := make(chan error, 1)
	go func() { errCh <- server.ListenAndServe() }()
	slog.Info("trip service started", "address", cfg.HTTPAddr)
	signalCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	select {
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("serve HTTP: %w", err)
		}
		return nil
	case <-signalCtx.Done():
		slog.Info("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown HTTP server: %w", err)
		}
		return nil
	}
}
