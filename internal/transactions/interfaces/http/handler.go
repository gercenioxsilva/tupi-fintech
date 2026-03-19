package http

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/tupi-fintech/desafio-tecnico/internal/platform/httpx"
	"github.com/tupi-fintech/desafio-tecnico/internal/transactions/application"
)

type Handler struct {
	service *application.Service
	logger  *slog.Logger
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

	processed, err := h.service.Process(r.Context(), command)
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
