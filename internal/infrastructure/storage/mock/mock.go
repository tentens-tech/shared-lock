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
	ExistingLeases map[string]int64
}

func New() *Storage {
	return &Storage{
		ExistingLeases: make(map[string]int64),
	}
}

func (s *Storage) CheckLeasePresence(ctx context.Context, key string) (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	leaseKey := DefaultPrefix + key
	if leaseID, exists := s.ExistingLeases[leaseKey]; exists {
		return leaseID, nil
	}
	return 0, nil
}

func (s *Storage) CreateLease(ctx context.Context, key string, leaseTTL int64, data []byte) (string, int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if leaseID, exists := s.ExistingLeases[key]; exists {
		return storage.StatusAccepted, leaseID, nil
	}

	leaseID := int64(123)
	s.ExistingLeases[key] = leaseID
	return storage.StatusCreated, leaseID, nil
}

func (s *Storage) KeepLeaseOnce(ctx context.Context, leaseID int64) error {
	if leaseID == 999 {
		return fmt.Errorf("lease not found")
	}
	return nil
}
