package storage

import "context"

type Connection interface {
	CheckLeasePresence(ctx context.Context, key string) (bool, error)
	CreateLease(ctx context.Context, key string, leaseTTL int64, data []byte) (string, int64, error)
	KeepLeaseOnce(ctx context.Context, leaseID int64) error
}
