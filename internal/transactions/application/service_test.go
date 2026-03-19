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
	saved   *ProcessedTransaction
	records []ProcessedTransaction
	err     error
}

func (s *stubRepository) Save(_ context.Context, record ProcessedTransaction) error {
	s.saved = &record
	return s.err
}

func (s *stubRepository) List(_ context.Context, limit int) ([]ProcessedTransaction, error) {
	if limit > len(s.records) {
		limit = len(s.records)
	}
	return s.records[:limit], s.err
}

func (s *stubRepository) GetByCorrelationID(_ context.Context, correlationID string) (ProcessedTransaction, error) {
	for _, record := range s.records {
		if record.Authorization.CorrelationID == correlationID {
			return record, s.err
		}
	}
	return ProcessedTransaction{}, ErrNotFound
}

type fixedClock struct{ now time.Time }

func (f fixedClock) Now() time.Time { return f.now }

func TestCommandServiceProcessApproved(t *testing.T) {
	metrics := observability.NewMetrics()
	repo := &stubRepository{}
	service := NewCommandService(
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

func TestCommandServiceProcessValidationError(t *testing.T) {
	metrics := observability.NewMetrics()
	service := NewCommandService(
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

func TestCommandServiceProcessAuthorizationError(t *testing.T) {
	metrics := observability.NewMetrics()
	service := NewCommandService(
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

func TestQueryServiceGetByCorrelationID(t *testing.T) {
	record := ProcessedTransaction{Authorization: domain.AuthorizationResult{CorrelationID: "abc-123"}}
	service := NewQueryService(&stubRepository{records: []ProcessedTransaction{record}})
	got, err := service.GetByCorrelationID(context.Background(), "abc-123")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got.Authorization.CorrelationID != "abc-123" {
		t.Fatalf("expected abc-123, got %s", got.Authorization.CorrelationID)
	}
}
