package leasemanagement

import (
	"context"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/tentens-tech/shared-lock/internal/infrastructure/storage"
)

const (
	DefaultPrefix               = "/shared-lock/"
	DefaultLeaseDurationSeconds = 10
)

func CreateLease(ctx context.Context, storageConnection storage.Storage, data []byte, leaseTTL time.Duration, lease Lease) (string, int64, error) {
	var err error
	var leaseID int64
	var leaseStatus string
	var isLeasePresent bool

	key := DefaultPrefix + lease.Key

	log.Debugf("Checking lease presence for the key: %v", key)
	isLeasePresent, err = storageConnection.CheckLeasePresence(ctx, key)
	if err != nil {
		return "", 0, fmt.Errorf("failed to check lease presence: %v", err)
	}
	if isLeasePresent {
		return "accepted", 0, nil
	}

	log.Debugf("Creating lease for the key: %v", key)
	leaseStatus, leaseID, err = storageConnection.CreateLease(ctx, key, int64(leaseTTL.Seconds()), data)
	if err != nil {
		return "", 0, err
	}

	log.Debugf("Prolong lease for the key: %v, with ttl: %v", key, leaseTTL)
	err = storageConnection.KeepLeaseOnce(ctx, leaseID)
	if err != nil {
		return "", 0, fmt.Errorf("failed to prolong lease with leaseID: %v, %v", leaseID, err)
	}

	return leaseStatus, leaseID, nil
}

func ReviveLease(ctx context.Context, storageConnection storage.Storage, leaseID int64) error {
	err := storageConnection.KeepLeaseOnce(ctx, leaseID)
	if err != nil {
		return err
	}

	return nil
}
