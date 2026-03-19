package application

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/tupi-fintech/desafio-tecnico/internal/platform/observability"
	"github.com/tupi-fintech/desafio-tecnico/internal/transactions/domain"
)

type TLVDecoder interface {
	Decode(input string) (map[string]string, error)
}

type Authorizer interface {
	Authorize(ctx context.Context, transaction domain.Transaction) (domain.AuthorizationResult, error)
}

type Repository interface {
	Save(ctx context.Context, record ProcessedTransaction) error
}

type Clock interface {
	Now() time.Time
}

type ProcessTransactionCommand struct {
	TLVPayload string `json:"tlv_payload"`
	Amount     int64  `json:"amount"`
	Currency   string `json:"currency"`
}

type ProcessedTransaction struct {
	Transaction   domain.Transaction         `json:"transaction"`
	Authorization domain.AuthorizationResult `json:"authorization"`
	Status        string                     `json:"status"`
}

type Service struct {
	decoder    TLVDecoder
	authorizer Authorizer
	repo       Repository
	clock      Clock
	logger     *slog.Logger
	metrics    *observability.Metrics
}

func NewService(decoder TLVDecoder, authorizer Authorizer, repo Repository, clock Clock, logger *slog.Logger, metrics *observability.Metrics) *Service {
	return &Service{decoder: decoder, authorizer: authorizer, repo: repo, clock: clock, logger: logger, metrics: metrics}
}

func (s *Service) Process(ctx context.Context, command ProcessTransactionCommand) (ProcessedTransaction, error) {
	startedAt := time.Now()
	tlvs, err := s.decoder.Decode(command.TLVPayload)
	if err != nil {
		s.observe("rejected", startedAt)
		return ProcessedTransaction{}, fmt.Errorf("decode tlvs: %w", err)
	}

	tx, err := domain.NewTransaction(tlvs, command.Amount, command.Currency, s.clock.Now())
	if err != nil {
		s.observe("rejected", startedAt)
		return ProcessedTransaction{}, err
	}

	result, err := s.authorizer.Authorize(ctx, tx)
	if err != nil {
		s.observe("error", startedAt)
		return ProcessedTransaction{}, fmt.Errorf("authorize transaction: %w", err)
	}

	status := "rejected"
	if result.Approved {
		status = "approved"
	}

	processed := ProcessedTransaction{Transaction: tx, Authorization: result, Status: status}
	if err := s.repo.Save(ctx, processed); err != nil {
		s.observe("error", startedAt)
		return ProcessedTransaction{}, fmt.Errorf("persist transaction: %w", err)
	}

	s.metrics.AuthorizationResults.Inc(status)
	s.observe(status, startedAt)
	s.logger.Info("transaction processed",
		slog.String("status", status),
		slog.String("correlation_id", correlationID(result)),
		slog.Int64("amount", tx.Amount),
		slog.String("currency", tx.Currency),
	)

	return processed, nil
}

func (s *Service) observe(status string, startedAt time.Time) {
	s.metrics.TransactionsTotal.Inc(status)
	s.metrics.TransactionDuration.Observe(status, time.Since(startedAt))
}

func correlationID(result domain.AuthorizationResult) string {
	return result.CorrelationID
}

type SystemClock struct{}

func (SystemClock) Now() time.Time { return time.Now().UTC() }

var ErrInvalidRequest = errors.New("invalid request")
