package storage

import "context"

type Connection interface {
	GetLease(ctx context.Context, key string, data []byte, leaseTTL int64) (string, int64, error)
	KeepLeaseOnce(ctx context.Context, leaseID int64) error
}
