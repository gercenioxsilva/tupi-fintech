package infrastructure

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/tupi-fintech/desafio-tecnico/internal/transactions/domain"
)

func TestMockAuthorizerApprove(t *testing.T) {
	authorizer := MockAuthorizer{
		Reader: bytes.NewReader([]byte{2, 0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff, 0x10, 0x20}),
		Now:    func() time.Time { return time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC) },
	}

	result, err := authorizer.Authorize(context.Background(), domain.Transaction{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !result.Approved {
		t.Fatal("expected approved result")
	}
	if result.Code != "00" {
		t.Fatalf("expected code 00, got %s", result.Code)
	}
	if result.CorrelationID != "aabbccddeeff1020" {
		t.Fatalf("unexpected correlation id: %s", result.CorrelationID)
	}
}

func TestMockAuthorizerReject(t *testing.T) {
	authorizer := MockAuthorizer{
		Reader: bytes.NewReader([]byte{1, 1, 2, 3, 4, 5, 6, 7, 8}),
		Now:    func() time.Time { return time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC) },
	}

	result, err := authorizer.Authorize(context.Background(), domain.Transaction{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Approved {
		t.Fatal("expected rejected result")
	}
	if result.Code != "05" {
		t.Fatalf("expected code 05, got %s", result.Code)
	}
	if result.Message != "rejected by mock gateway" {
		t.Fatalf("unexpected message: %s", result.Message)
	}
}
