package infrastructure

import (
	"context"
	crand "crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"time"

	"github.com/tupi-fintech/desafio-tecnico/internal/transactions/domain"
)

type MockAuthorizer struct {
	Reader io.Reader
	Now    func() time.Time
}

func (a MockAuthorizer) Authorize(_ context.Context, _ domain.Transaction) (domain.AuthorizationResult, error) {
	random := a.Reader
	if random == nil {
		random = crand.Reader
	}
	now := a.Now
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}

	decision := make([]byte, 1)
	if _, err := io.ReadFull(random, decision); err != nil {
		return domain.AuthorizationResult{}, fmt.Errorf("read authorization randomness: %w", err)
	}

	correlation := make([]byte, 8)
	if _, err := io.ReadFull(random, correlation); err != nil {
		return domain.AuthorizationResult{}, fmt.Errorf("read correlation id randomness: %w", err)
	}

	approved := decision[0]%2 == 0
	code := "05"
	message := "rejected by mock gateway"
	if approved {
		code = "00"
		message = "approved"
	}

	return domain.AuthorizationResult{
		Approved:      approved,
		Code:          code,
		Message:       message,
		AuthorizedAt:  now().UTC(),
		CorrelationID: hex.EncodeToString(correlation),
	}, nil
}
