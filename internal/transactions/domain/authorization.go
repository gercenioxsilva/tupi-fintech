package domain

import "time"

type AuthorizationResult struct {
	Approved      bool      `json:"approved"`
	Code          string    `json:"code"`
	Message       string    `json:"message"`
	AuthorizedAt  time.Time `json:"authorized_at"`
	CorrelationID string    `json:"correlation_id"`
}
