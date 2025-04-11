package mock

import (
	"context"
	"fmt"
	"sync"

	"github.com/tentens-tech/shared-lock/internal/infrastructure/storage"
)

const DefaultPrefix = "/shared-lock/"

type Storage struct {
	mu             sync.RWMutex
	ExistingLeases map[string]bool
}

func New() *Storage {
	return &Storage{
		ExistingLeases: make(map[string]bool),
	}
}

func (s *Storage) CheckLeasePresence(ctx context.Context, key string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	leaseKey := DefaultPrefix + key
	return s.ExistingLeases[leaseKey], nil
}

func (s *Storage) CreateLease(ctx context.Context, key string, leaseTTL int64, data []byte) (string, int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.ExistingLeases[key] {
		return storage.StatusAccepted, 456, nil
	}

	s.ExistingLeases[key] = true
	return storage.StatusCreated, 123, nil
}

func (s *Storage) KeepLeaseOnce(ctx context.Context, leaseID int64) error {
	if leaseID == 999 {
		return fmt.Errorf("lease not found")
	}
	return nil
}
