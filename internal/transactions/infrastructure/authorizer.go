package infrastructure

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/tupi-fintech/desafio-tecnico/internal/transactions/domain"
)

type MockAuthorizer struct{}

func (MockAuthorizer) Authorize(_ context.Context, transaction domain.Transaction) (domain.AuthorizationResult, error) {
	hash := sha1.Sum([]byte(fmt.Sprintf("%s:%s:%d:%s", transaction.PAN, transaction.ExpiryDate, transaction.Amount, transaction.Currency)))
	approved := hash[0]%2 == 0
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
		AuthorizedAt:  time.Now().UTC(),
		CorrelationID: hex.EncodeToString(hash[:8]),
	}, nil
}
