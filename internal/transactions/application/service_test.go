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
	calls  int
}

func (s *stubAuthorizer) Authorize(context.Context, domain.Transaction) (domain.AuthorizationResult, error) {
	s.calls++
	return s.result, s.err
}

type stubRepository struct {
	saved             *ProcessedTransaction
	records           []ProcessedTransaction
	idempotencyRecord map[string]ProcessedTransaction
	err               error
}

func (s *stubRepository) Save(_ context.Context, record ProcessedTransaction) error {
	s.saved = &record
	if s.idempotencyRecord == nil {
		s.idempotencyRecord = map[string]ProcessedTransaction{}
	}
	s.idempotencyRecord[record.IdempotencyKey] = record
	return s.err
}

func (s *stubRepository) GetByIdempotencyKey(_ context.Context, idempotencyKey string) (ProcessedTransaction, error) {
	if s.err != nil {
		return ProcessedTransaction{}, s.err
	}
	record, ok := s.idempotencyRecord[idempotencyKey]
	if !ok {
		return ProcessedTransaction{}, ErrNotFound
	}
	return record, nil
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
	repo := &stubRepository{idempotencyRecord: map[string]ProcessedTransaction{}}
	authorizer := &stubAuthorizer{result: domain.AuthorizationResult{Approved: true, Code: "00", Message: "approved", CorrelationID: "abc"}}
	service := NewCommandService(
		stubDecoder{tlvs: map[string]string{"5A": "4111111111111111", "5F24": "301231", "9F34": "1E0300"}},
		authorizer,
		repo,
		fixedClock{now: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		metrics,
	)

	result, err := service.Process(context.Background(), ProcessTransactionCommand{TLVPayload: "ignored", Amount: 1000, Currency: "BRL", IdempotencyKey: "pay-1"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Status != "approved" {
		t.Fatalf("expected approved, got %s", result.Status)
	}
	if result.IdempotencyKey != "pay-1" {
		t.Fatalf("expected idempotency key pay-1, got %s", result.IdempotencyKey)
	}
	if repo.saved == nil {
		t.Fatal("expected record to be persisted")
	}
	if authorizer.calls != 1 {
		t.Fatalf("expected authorizer to be called once, got %d", authorizer.calls)
	}
}

func TestCommandServiceProcessIdempotentReplayReturnsStoredRecord(t *testing.T) {
	metrics := observability.NewMetrics()
	existing := ProcessedTransaction{
		IdempotencyKey: "pay-1",
		Transaction:    domain.Transaction{PAN: "4111111111111111", ExpiryDate: "301231", CVM: "1E0300", Amount: 1000, Currency: "BRL", TLVs: map[string]string{"5A": "4111111111111111", "5F24": "301231", "9F34": "1E0300"}, ProcessedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
		Authorization:  domain.AuthorizationResult{Approved: true, Code: "00", Message: "approved", CorrelationID: "abc"},
		Status:         "approved",
	}
	repo := &stubRepository{idempotencyRecord: map[string]ProcessedTransaction{"pay-1": existing}}
	authorizer := &stubAuthorizer{result: domain.AuthorizationResult{Approved: true, Code: "00", Message: "approved", CorrelationID: "new"}}
	service := NewCommandService(
		stubDecoder{tlvs: map[string]string{"5A": "4111111111111111", "5F24": "301231", "9F34": "1E0300"}},
		authorizer,
		repo,
		fixedClock{now: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		metrics,
	)

	result, err := service.Process(context.Background(), ProcessTransactionCommand{TLVPayload: "ignored", Amount: 1000, Currency: "BRL", IdempotencyKey: "pay-1"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Authorization.CorrelationID != "abc" {
		t.Fatalf("expected stored correlation id abc, got %s", result.Authorization.CorrelationID)
	}
	if authorizer.calls != 0 {
		t.Fatalf("expected authorizer not to be called on replay, got %d", authorizer.calls)
	}
}

func TestCommandServiceProcessIdempotencyConflict(t *testing.T) {
	metrics := observability.NewMetrics()
	existing := ProcessedTransaction{
		IdempotencyKey: "pay-1",
		Transaction:    domain.Transaction{PAN: "4111111111111111", ExpiryDate: "301231", CVM: "1E0300", Amount: 1000, Currency: "BRL", TLVs: map[string]string{"5A": "4111111111111111", "5F24": "301231", "9F34": "1E0300"}, ProcessedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
		Authorization:  domain.AuthorizationResult{Approved: true, Code: "00", Message: "approved", CorrelationID: "abc"},
		Status:         "approved",
	}
	repo := &stubRepository{idempotencyRecord: map[string]ProcessedTransaction{"pay-1": existing}}
	authorizer := &stubAuthorizer{result: domain.AuthorizationResult{Approved: true, Code: "00", Message: "approved", CorrelationID: "new"}}
	service := NewCommandService(
		stubDecoder{tlvs: map[string]string{"5A": "4111111111111111", "5F24": "301231", "9F34": "1E0300"}},
		authorizer,
		repo,
		fixedClock{now: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		metrics,
	)

	_, err := service.Process(context.Background(), ProcessTransactionCommand{TLVPayload: "ignored", Amount: 2000, Currency: "BRL", IdempotencyKey: "pay-1"})
	if !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("expected idempotency conflict, got %v", err)
	}
	if authorizer.calls != 0 {
		t.Fatalf("expected authorizer not to be called on conflict, got %d", authorizer.calls)
	}
}

func TestCommandServiceRequiresIdempotencyKey(t *testing.T) {
	metrics := observability.NewMetrics()
	service := NewCommandService(
		stubDecoder{tlvs: map[string]string{"5A": "4111111111111111", "5F24": "301231", "9F34": "1E0300"}},
		&stubAuthorizer{},
		&stubRepository{idempotencyRecord: map[string]ProcessedTransaction{}},
		fixedClock{now: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		metrics,
	)

	_, err := service.Process(context.Background(), ProcessTransactionCommand{TLVPayload: "ignored", Amount: 1000, Currency: "BRL"})
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("expected invalid request, got %v", err)
	}
}

func TestCommandServiceProcessValidationError(t *testing.T) {
	metrics := observability.NewMetrics()
	service := NewCommandService(
		stubDecoder{tlvs: map[string]string{"5A": "123", "5F24": "301231", "9F34": "1E0300"}},
		&stubAuthorizer{},
		&stubRepository{idempotencyRecord: map[string]ProcessedTransaction{}},
		fixedClock{now: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		metrics,
	)
	_, err := service.Process(context.Background(), ProcessTransactionCommand{TLVPayload: "ignored", Amount: 1000, Currency: "BRL", IdempotencyKey: "pay-1"})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestCommandServiceProcessAuthorizationError(t *testing.T) {
	metrics := observability.NewMetrics()
	authorizer := &stubAuthorizer{err: errors.New("gateway down")}
	service := NewCommandService(
		stubDecoder{tlvs: map[string]string{"5A": "4111111111111111", "5F24": "301231", "9F34": "1E0300"}},
		authorizer,
		&stubRepository{idempotencyRecord: map[string]ProcessedTransaction{}},
		fixedClock{now: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		metrics,
	)
	_, err := service.Process(context.Background(), ProcessTransactionCommand{TLVPayload: "ignored", Amount: 1000, Currency: "BRL", IdempotencyKey: "pay-1"})
	if err == nil {
		t.Fatal("expected authorization error")
	}
}

func TestQueryServiceGetByCorrelationID(t *testing.T) {
	record := ProcessedTransaction{Authorization: domain.AuthorizationResult{CorrelationID: "abc-123"}}
	service := NewQueryService(&stubRepository{records: []ProcessedTransaction{record}, idempotencyRecord: map[string]ProcessedTransaction{}})
	got, err := service.GetByCorrelationID(context.Background(), "abc-123")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got.Authorization.CorrelationID != "abc-123" {
		t.Fatalf("expected abc-123, got %s", got.Authorization.CorrelationID)
	}
}
