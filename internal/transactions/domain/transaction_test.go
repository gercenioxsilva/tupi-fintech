package domain

import (
	"testing"
	"time"
)

func TestNewTransactionValidatesPANAndLuhn(t *testing.T) {
	_, err := NewTransaction(map[string]string{"5A": "4111111111111111", "5F24": "301231", "9F34": "1E0300"}, 1000, "BRL", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("expected valid transaction, got %v", err)
	}
}

func TestNewTransactionRejectsExpiredCard(t *testing.T) {
	_, err := NewTransaction(map[string]string{"5A": "4111111111111111", "5F24": "240101", "9F34": "1E0300"}, 1000, "BRL", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	if err == nil {
		t.Fatal("expected expired card error")
	}
}
