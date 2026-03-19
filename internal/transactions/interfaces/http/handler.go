package http

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/tupi-fintech/desafio-tecnico/internal/platform/httpx"
	"github.com/tupi-fintech/desafio-tecnico/internal/transactions/application"
)

type Handler struct {
	commands *application.CommandService
	queries  *application.QueryService
	logger   *slog.Logger
}

func (h Handler) ProcessTransaction(w http.ResponseWriter, r *http.Request) {
	payload, err := bodyBytes(r)
	if err != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "failed to read request body"})
		return
	}

	var command application.ProcessTransactionCommand
	if err := json.Unmarshal(payload, &command); err != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON payload"})
		return
	}

	processed, err := h.commands.Process(r.Context(), command)
	if err != nil {
		status := http.StatusUnprocessableEntity
		if errors.Is(err, application.ErrInvalidRequest) {
			status = http.StatusBadRequest
		}
		h.logger.Warn("failed to process transaction", slog.Any("error", err))
		httpx.WriteJSON(w, status, map[string]string{"error": err.Error()})
		return
	}

	httpx.WriteJSON(w, http.StatusOK, processed)
}

func (h Handler) ListTransactions(w http.ResponseWriter, r *http.Request) {
	limit, err := parseLimit(r.URL.Query().Get("limit"))
	if err != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	transactions, err := h.queries.List(r.Context(), limit)
	if err != nil {
		h.logger.Error("failed to list transactions", slog.Any("error", err))
		httpx.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to query transactions"})
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": transactions})
}

func (h Handler) GetTransaction(w http.ResponseWriter, r *http.Request) {
	correlationID := strings.TrimPrefix(r.URL.Path, "/api/v1/emv/transactions/")
	record, err := h.queries.GetByCorrelationID(r.Context(), correlationID)
	if err != nil {
		switch {
		case errors.Is(err, application.ErrInvalidRequest):
			httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "correlation id is required"})
		case errors.Is(err, application.ErrNotFound):
			httpx.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "transaction not found"})
		default:
			h.logger.Error("failed to get transaction", slog.Any("error", err), slog.String("correlation_id", correlationID))
			httpx.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to query transaction"})
		}
		return
	}
	httpx.WriteJSON(w, http.StatusOK, record)
}

func parseLimit(raw string) (int, error) {
	if raw == "" {
		return 50, nil
	}
	limit, err := strconv.Atoi(raw)
	if err != nil || limit <= 0 {
		return 0, errors.New("limit must be a positive integer")
	}
	return limit, nil
}
