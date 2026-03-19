package http

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tupi-fintech/desafio-tecnico/internal/platform/config"
	"github.com/tupi-fintech/desafio-tecnico/internal/platform/observability"
	"github.com/tupi-fintech/desafio-tecnico/internal/transactions/application"
)

func TestOpenAPISpecEndpoint(t *testing.T) {
	router := NewRouter(
		config.Config{HTTPAddress: ":8080", Environment: "test"},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		observability.NewMetrics(),
		&application.Service{},
	)

	req := httptest.NewRequest(http.MethodGet, "/openapi.json", nil)
	req.Host = "api.example.com"
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("expected application/json content type, got %q", got)
	}

	var spec map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &spec); err != nil {
		t.Fatalf("expected valid json, got error %v", err)
	}
	if spec["openapi"] != "3.0.3" {
		t.Fatalf("expected openapi version 3.0.3, got %v", spec["openapi"])
	}

	servers, ok := spec["servers"].([]any)
	if !ok || len(servers) == 0 {
		t.Fatalf("expected servers section in spec")
	}
	server, ok := servers[0].(map[string]any)
	if !ok {
		t.Fatalf("expected first server to be an object")
	}
	if server["url"] != "http://api.example.com" {
		t.Fatalf("expected server url http://api.example.com, got %v", server["url"])
	}
}

func TestSwaggerUIEndpoint(t *testing.T) {
	router := NewRouter(
		config.Config{HTTPAddress: ":8080", Environment: "test"},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		observability.NewMetrics(),
		&application.Service{},
	)

	req := httptest.NewRequest(http.MethodGet, "/swagger", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Fatalf("expected html content type, got %q", got)
	}
	if body := rec.Body.String(); body == "" || !strings.Contains(body, "SwaggerUIBundle") || !strings.Contains(body, "/openapi.json") {
		t.Fatalf("expected swagger ui page to reference SwaggerUIBundle and /openapi.json")
	}
}

func TestServerURLFallsBackToConfiguredAddress(t *testing.T) {
	if got := serverURL(nil, ":8080"); got != "http://localhost:8080" {
		t.Fatalf("expected localhost fallback, got %q", got)
	}
}
