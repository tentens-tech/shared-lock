package mock

import (
	"context"

	"github.com/tentens-tech/shared-lock/internal/infrastructure/storage"
)

type Storage struct{}

func (s *Storage) CheckLeasePresence(ctx context.Context, key string) (bool, error) {
	return false, nil
}

func (s *Storage) CreateLease(ctx context.Context, key string, leaseTTL int64, data []byte) (string, int64, error) {
	return storage.StatusCreated, 123, nil
}

func (s *Storage) KeepLeaseOnce(ctx context.Context, leaseID int64) error {
	return nil
}
