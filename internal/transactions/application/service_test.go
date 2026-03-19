package application

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/tupi-fintech/desafio-tecnico/internal/platform/observability"
	"github.com/tupi-fintech/desafio-tecnico/internal/transactions/domain"
)

type stubDecoder struct {
	tlvs map[string]string
	err  error
}

func (s stubDecoder) Decode(string) (map[string]string, error) { return s.tlvs, s.err }

type stubAuthorizer struct {
	result domain.AuthorizationResult
	err    error
}

func (s stubAuthorizer) Authorize(context.Context, domain.Transaction) (domain.AuthorizationResult, error) {
	return s.result, s.err
}

type stubRepository struct {
	saved *ProcessedTransaction
	err   error
}

func (s *stubRepository) Save(_ context.Context, record ProcessedTransaction) error {
	s.saved = &record
	return s.err
}

type fixedClock struct{ now time.Time }

func (f fixedClock) Now() time.Time { return f.now }

func TestServiceProcessApproved(t *testing.T) {
	metrics := observability.NewMetrics()
	repo := &stubRepository{}
	service := NewService(
		stubDecoder{tlvs: map[string]string{"5A": "4111111111111111", "5F24": "301231", "9F34": "1E0300"}},
		stubAuthorizer{result: domain.AuthorizationResult{Approved: true, Code: "00", Message: "approved", CorrelationID: "abc"}},
		repo,
		fixedClock{now: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		metrics,
	)

	result, err := service.Process(context.Background(), ProcessTransactionCommand{TLVPayload: "ignored", Amount: 1000, Currency: "BRL"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Status != "approved" {
		t.Fatalf("expected approved, got %s", result.Status)
	}
	if repo.saved == nil {
		t.Fatal("expected record to be persisted")
	}
}

func TestServiceProcessValidationError(t *testing.T) {
	metrics := observability.NewMetrics()
	service := NewService(
		stubDecoder{tlvs: map[string]string{"5A": "123", "5F24": "301231", "9F34": "1E0300"}},
		stubAuthorizer{},
		&stubRepository{},
		fixedClock{now: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		metrics,
	)
	_, err := service.Process(context.Background(), ProcessTransactionCommand{TLVPayload: "ignored", Amount: 1000, Currency: "BRL"})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestServiceProcessAuthorizationError(t *testing.T) {
	metrics := observability.NewMetrics()
	service := NewService(
		stubDecoder{tlvs: map[string]string{"5A": "4111111111111111", "5F24": "301231", "9F34": "1E0300"}},
		stubAuthorizer{err: errors.New("gateway down")},
		&stubRepository{},
		fixedClock{now: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		metrics,
	)
	_, err := service.Process(context.Background(), ProcessTransactionCommand{TLVPayload: "ignored", Amount: 1000, Currency: "BRL"})
	if err == nil {
		t.Fatal("expected authorization error")
	}
}
