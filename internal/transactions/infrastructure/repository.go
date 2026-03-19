package infrastructure

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/tupi-fintech/desafio-tecnico/internal/transactions/application"
)

type PostgresRepository struct {
	connectionString string
}

func NewPostgresRepository(connectionString string) (*PostgresRepository, error) {
	repo := &PostgresRepository{connectionString: connectionString}
	if err := repo.migrate(context.Background()); err != nil {
		return nil, err
	}
	return repo, nil
}

func (r *PostgresRepository) Close() error { return nil }

func (r *PostgresRepository) migrate(ctx context.Context) error {
	const query = `
CREATE TABLE IF NOT EXISTS transactions (
    correlation_id TEXT PRIMARY KEY,
    idempotency_key TEXT,
    pan TEXT NOT NULL,
    expiry_date TEXT NOT NULL,
    cvm TEXT NOT NULL,
    amount BIGINT NOT NULL,
    currency TEXT NOT NULL,
    tlvs JSONB NOT NULL,
    processed_at TIMESTAMPTZ NOT NULL,
    approved BOOLEAN NOT NULL,
    authorization_code TEXT NOT NULL,
    authorization_message TEXT NOT NULL,
    authorized_at TIMESTAMPTZ NOT NULL,
    status TEXT NOT NULL
);
ALTER TABLE transactions ADD COLUMN IF NOT EXISTS idempotency_key TEXT;
UPDATE transactions SET idempotency_key = correlation_id WHERE idempotency_key IS NULL;
ALTER TABLE transactions ALTER COLUMN idempotency_key SET NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_transactions_idempotency_key ON transactions (idempotency_key);
CREATE INDEX IF NOT EXISTS idx_transactions_processed_at ON transactions (processed_at DESC);
`
	return r.execStatement(ctx, query)
}

func (r *PostgresRepository) Save(ctx context.Context, record application.ProcessedTransaction) error {
	tlvs, err := json.Marshal(record.Transaction.TLVs)
	if err != nil {
		return fmt.Errorf("marshal tlvs: %w", err)
	}
	query := fmt.Sprintf(`
INSERT INTO transactions (
    correlation_id, idempotency_key, pan, expiry_date, cvm, amount, currency, tlvs,
    processed_at, approved, authorization_code, authorization_message, authorized_at, status
) VALUES (%s,%s,%s,%s,%s,%d,%s,%s::jsonb,%s,%t,%s,%s,%s,%s)
ON CONFLICT (correlation_id) DO UPDATE SET
    idempotency_key = EXCLUDED.idempotency_key,
    pan = EXCLUDED.pan,
    expiry_date = EXCLUDED.expiry_date,
    cvm = EXCLUDED.cvm,
    amount = EXCLUDED.amount,
    currency = EXCLUDED.currency,
    tlvs = EXCLUDED.tlvs,
    processed_at = EXCLUDED.processed_at,
    approved = EXCLUDED.approved,
    authorization_code = EXCLUDED.authorization_code,
    authorization_message = EXCLUDED.authorization_message,
    authorized_at = EXCLUDED.authorized_at,
    status = EXCLUDED.status;
`, sqlLiteral(record.Authorization.CorrelationID), sqlLiteral(record.IdempotencyKey), sqlLiteral(record.Transaction.PAN), sqlLiteral(record.Transaction.ExpiryDate), sqlLiteral(record.Transaction.CVM), record.Transaction.Amount, sqlLiteral(record.Transaction.Currency), sqlLiteral(string(tlvs)), sqlLiteral(record.Transaction.ProcessedAt.UTC().Format(time.RFC3339Nano)), record.Authorization.Approved, sqlLiteral(record.Authorization.Code), sqlLiteral(record.Authorization.Message), sqlLiteral(record.Authorization.AuthorizedAt.UTC().Format(time.RFC3339Nano)), sqlLiteral(record.Status))
	return r.execStatement(ctx, query)
}

func (r *PostgresRepository) List(ctx context.Context, limit int) ([]application.ProcessedTransaction, error) {
	query := fmt.Sprintf(`COPY (
SELECT json_build_object(
    'idempotency_key', idempotency_key,
    'transaction', json_build_object(
        'pan', pan,
        'expiry_date', expiry_date,
        'cvm', cvm,
        'amount', amount,
        'currency', currency,
        'tlvs', tlvs,
        'processed_at', processed_at
    ),
    'authorization', json_build_object(
        'approved', approved,
        'code', authorization_code,
        'message', authorization_message,
        'authorized_at', authorized_at,
        'correlation_id', correlation_id
    ),
    'status', status
)
FROM transactions
ORDER BY processed_at DESC
LIMIT %d
) TO STDOUT;`, limit)
	return r.queryRecords(ctx, query)
}

func (r *PostgresRepository) GetByCorrelationID(ctx context.Context, correlationID string) (application.ProcessedTransaction, error) {
	query := fmt.Sprintf(`COPY (
SELECT json_build_object(
    'idempotency_key', idempotency_key,
    'transaction', json_build_object(
        'pan', pan,
        'expiry_date', expiry_date,
        'cvm', cvm,
        'amount', amount,
        'currency', currency,
        'tlvs', tlvs,
        'processed_at', processed_at
    ),
    'authorization', json_build_object(
        'approved', approved,
        'code', authorization_code,
        'message', authorization_message,
        'authorized_at', authorized_at,
        'correlation_id', correlation_id
    ),
    'status', status
)
FROM transactions
WHERE correlation_id = %s
) TO STDOUT;`, sqlLiteral(correlationID))
	records, err := r.queryRecords(ctx, query)
	if err != nil {
		return application.ProcessedTransaction{}, err
	}
	if len(records) == 0 {
		return application.ProcessedTransaction{}, application.ErrNotFound
	}
	return records[0], nil
}

func (r *PostgresRepository) GetByIdempotencyKey(ctx context.Context, idempotencyKey string) (application.ProcessedTransaction, error) {
	query := fmt.Sprintf(`COPY (
SELECT json_build_object(
    'idempotency_key', idempotency_key,
    'transaction', json_build_object(
        'pan', pan,
        'expiry_date', expiry_date,
        'cvm', cvm,
        'amount', amount,
        'currency', currency,
        'tlvs', tlvs,
        'processed_at', processed_at
    ),
    'authorization', json_build_object(
        'approved', approved,
        'code', authorization_code,
        'message', authorization_message,
        'authorized_at', authorized_at,
        'correlation_id', correlation_id
    ),
    'status', status
)
FROM transactions
WHERE idempotency_key = %s
) TO STDOUT;`, sqlLiteral(idempotencyKey))
	records, err := r.queryRecords(ctx, query)
	if err != nil {
		return application.ProcessedTransaction{}, err
	}
	if len(records) == 0 {
		return application.ProcessedTransaction{}, application.ErrNotFound
	}
	return records[0], nil
}

func (r *PostgresRepository) execStatement(ctx context.Context, query string) error {
	cmd := exec.CommandContext(ctx, "psql", r.connectionString, "-v", "ON_ERROR_STOP=1", "-X", "-q", "-c", query)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("execute psql statement: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func (r *PostgresRepository) queryRecords(ctx context.Context, query string) ([]application.ProcessedTransaction, error) {
	cmd := exec.CommandContext(ctx, "psql", r.connectionString, "-v", "ON_ERROR_STOP=1", "-X", "-q", "-t", "-A", "-c", query)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("execute psql query: %w: %s", err, strings.TrimSpace(string(output)))
	}
	var records []application.ProcessedTransaction
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var record application.ProcessedTransaction
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			return nil, fmt.Errorf("decode transaction row: %w", err)
		}
		records = append(records, normalizeRecord(record))
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan query result: %w", err)
	}
	return records, nil
}

func normalizeRecord(record application.ProcessedTransaction) application.ProcessedTransaction {
	record.Transaction.ProcessedAt = record.Transaction.ProcessedAt.UTC()
	record.Authorization.AuthorizedAt = record.Authorization.AuthorizedAt.UTC()
	return record
}

func sqlLiteral(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

var _ application.TransactionWriter = (*PostgresRepository)(nil)
var _ application.TransactionReader = (*PostgresRepository)(nil)
