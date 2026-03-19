package domain

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type Transaction struct {
	PAN         string            `json:"pan"`
	ExpiryDate  string            `json:"expiry_date"`
	CVM         string            `json:"cvm"`
	Amount      int64             `json:"amount"`
	Currency    string            `json:"currency"`
	TLVs        map[string]string `json:"tlvs"`
	ProcessedAt time.Time         `json:"processed_at"`
}

var supportedCVMPrefixes = []string{"1E", "1F", "42", "03", "04"}

func NewTransaction(tlvs map[string]string, amount int64, currency string, now time.Time) (Transaction, error) {
	pan, ok := tlvs["5A"]
	if !ok {
		return Transaction{}, errors.New("missing TLV 5A PAN")
	}
	expiry, ok := tlvs["5F24"]
	if !ok {
		return Transaction{}, errors.New("missing TLV 5F24 expiry date")
	}
	cvm, ok := tlvs["9F34"]
	if !ok {
		return Transaction{}, errors.New("missing TLV 9F34 CVM")
	}

	t := Transaction{PAN: pan, ExpiryDate: expiry, CVM: cvm, Amount: amount, Currency: currency, TLVs: tlvs, ProcessedAt: now.UTC()}
	if err := t.Validate(now); err != nil {
		return Transaction{}, err
	}
	return t, nil
}

func (t Transaction) Validate(now time.Time) error {
	if err := validatePAN(t.PAN); err != nil {
		return err
	}
	if err := validateExpiry(t.ExpiryDate, now); err != nil {
		return err
	}
	if err := validateCVM(t.CVM); err != nil {
		return err
	}
	if t.Amount <= 0 {
		return errors.New("amount must be greater than zero")
	}
	if len(strings.TrimSpace(t.Currency)) != 3 {
		return errors.New("currency must contain ISO-4217 alpha-3 code")
	}
	return nil
}

func validatePAN(pan string) error {
	if len(pan) < 13 || len(pan) > 19 {
		return fmt.Errorf("invalid PAN length: %d", len(pan))
	}
	for _, c := range pan {
		if c < '0' || c > '9' {
			return errors.New("PAN must contain only digits")
		}
	}
	if !passesLuhn(pan) {
		return errors.New("PAN failed luhn validation")
	}
	return nil
}

func validateExpiry(value string, now time.Time) error {
	if len(value) != 6 {
		return errors.New("expiry date must be formatted as YYMMDD")
	}
	year, err := strconv.Atoi(value[0:2])
	if err != nil {
		return errors.New("expiry date contains invalid year")
	}
	month, err := strconv.Atoi(value[2:4])
	if err != nil {
		return errors.New("expiry date contains invalid month")
	}
	day, err := strconv.Atoi(value[4:6])
	if err != nil {
		return errors.New("expiry date contains invalid day")
	}
	expiry := time.Date(2000+year, time.Month(month), day, 23, 59, 59, 0, time.UTC)
	if expiry.Before(now.UTC()) {
		return errors.New("card is expired")
	}
	return nil
}

func validateCVM(value string) error {
	if len(value) < 2 {
		return errors.New("invalid CVM data")
	}
	prefix := strings.ToUpper(value[0:2])
	for _, supported := range supportedCVMPrefixes {
		if prefix == supported {
			return nil
		}
	}
	return fmt.Errorf("unsupported CVM method: %s", prefix)
}

func passesLuhn(value string) bool {
	sum := 0
	alt := false
	for i := len(value) - 1; i >= 0; i-- {
		n := int(value[i] - '0')
		if alt {
			n *= 2
			if n > 9 {
				n -= 9
			}
		}
		sum += n
		alt = !alt
	}
	return sum%10 == 0
}
