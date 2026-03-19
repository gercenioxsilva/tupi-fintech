package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/tupi-fintech/desafio-tecnico/internal/transactions/application"
)

type FileRepository struct {
	path string
	mu   sync.Mutex
}

func NewFileRepository(path string) (*FileRepository, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil && filepath.Dir(path) != "." {
		return nil, fmt.Errorf("create log directory: %w", err)
	}
	return &FileRepository{path: path}, nil
}

func (r *FileRepository) Close() error { return nil }

func (r *FileRepository) Save(_ context.Context, record application.ProcessedTransaction) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	file, err := os.OpenFile(r.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open transaction log file: %w", err)
	}
	defer file.Close()
	payload, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("marshal transaction log: %w", err)
	}
	if _, err := file.Write(append(payload, '\n')); err != nil {
		return fmt.Errorf("append transaction log: %w", err)
	}
	return nil
}
