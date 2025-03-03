package storage

import (
	"context"
)

const (
	StatusAccepted = "accepted"
	StatusCreated  = "created"
)

type Storage interface {
	CheckLeasePresence(ctx context.Context, key string) (isPresent bool, err error)
	CreateLease(ctx context.Context, key string, leaseTTL int64, data []byte) (leaseStatus string, leaseID int64, err error)
	KeepLeaseOnce(ctx context.Context, leaseID int64) error
}
