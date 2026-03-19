package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tupi-fintech/desafio-tecnico/internal/platform/config"
	"github.com/tupi-fintech/desafio-tecnico/internal/platform/observability"
	transport "github.com/tupi-fintech/desafio-tecnico/internal/transactions/interfaces/http"
)

func main() {
	cfg := config.Load()
	logger := observability.NewLogger(cfg.LogLevel)
	metrics := observability.NewMetrics()

	deps, err := transport.NewDependencies(cfg, logger, metrics)
	if err != nil {
		logger.Error("failed to initialize dependencies", slog.Any("error", err))
		os.Exit(1)
	}
	defer deps.Close()

	handler := transport.NewRouter(cfg, logger, metrics, deps.Commands, deps.Queries)
	server := &http.Server{
		Addr:              cfg.HTTPAddress,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		logger.Info("http server starting", slog.String("address", cfg.HTTPAddress))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server stopped unexpectedly", slog.Any("error", err))
			stop()
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", slog.Any("error", err))
		os.Exit(1)
	}

	logger.Info("server stopped")
}
