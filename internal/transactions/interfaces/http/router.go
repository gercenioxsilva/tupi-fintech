package http

import (
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/tupi-fintech/desafio-tecnico/internal/platform/config"
	"github.com/tupi-fintech/desafio-tecnico/internal/platform/httpx"
	"github.com/tupi-fintech/desafio-tecnico/internal/platform/observability"
	"github.com/tupi-fintech/desafio-tecnico/internal/transactions/application"
	"github.com/tupi-fintech/desafio-tecnico/internal/transactions/infrastructure"
)

type Dependencies struct {
	Service *application.Service
	repo    interface{ Close() error }
}

func NewDependencies(cfg config.Config, logger *slog.Logger, metrics *observability.Metrics) (*Dependencies, error) {
	repo, err := infrastructure.NewFileRepository(cfg.DatabasePath)
	if err != nil {
		return nil, err
	}
	service := application.NewService(
		infrastructure.TLVDecoder{},
		infrastructure.MockAuthorizer{},
		repo,
		application.SystemClock{},
		logger,
		metrics,
	)
	return &Dependencies{Service: service, repo: repo}, nil
}

func (d *Dependencies) Close() error { return d.repo.Close() }

func NewRouter(cfg config.Config, logger *slog.Logger, handlerMetrics *observability.Metrics, service *application.Service) http.Handler {
	mux := http.NewServeMux()
	handler := Handler{service: service, logger: logger}
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok", "env": cfg.Environment})
	})
	mux.HandleFunc("GET /metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		_, _ = w.Write([]byte(handlerMetrics.RenderPrometheus()))
	})
	mux.HandleFunc("POST /api/v1/emv/transactions", handler.ProcessTransaction)

	return loggingMiddleware(logger, mux)
}

func loggingMiddleware(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		ww := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(ww, r)
		logger.Info("http request",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", ww.status),
			slog.Duration("duration", time.Since(started)),
		)
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusWriter) Write(p []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.ResponseWriter.Write(p)
}

func bodyBytes(r *http.Request) ([]byte, error) {
	defer r.Body.Close()
	return io.ReadAll(io.LimitReader(r.Body, 1<<20))
}
