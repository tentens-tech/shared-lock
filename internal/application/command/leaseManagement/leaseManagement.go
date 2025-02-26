package leaseManagement

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/tentens-tech/shared-lock/internal/config"
	"github.com/tentens-tech/shared-lock/internal/infrastructure/storage"
)

const (
	DefaultPrefix               = "/shared-lock/"
	DefaultLeaseDurationSeconds = 10
)

func CreateLease(ctx context.Context, cfg *config.Config, storageConnection storage.Connection, data []byte, leaseTTLString string, lease Lease) (string, int64, error) {
	var err error
	var leaseTTL time.Duration
	var leaseID int64
	var leaseStatus string

	leaseTTL, err = time.ParseDuration(leaseTTLString)
	if err != nil {
		log.Warnf("Use defaultLeaseDurationSeconds for %v", lease.Key)
		leaseTTL = DefaultLeaseDurationSeconds
	}

	key := DefaultPrefix + lease.Key

	log.Debugf("Get lease for key: %v, with ttl: %v", key, leaseTTL)
	leaseStatus, leaseID, err = storageConnection.GetLease(ctx, key, data, int64(leaseTTL.Seconds()))
	if err != nil {
		log.Errorf("%v", err)
		return "", 0, err
	}

	return leaseStatus, leaseID, nil
}

func ReviveLease(ctx context.Context, storageConnection storage.Connection, leaseID int64) error {
	err := storageConnection.KeepLeaseOnce(ctx, leaseID)
	if err != nil {
		log.Warnf("Failed to prolong lease: %v", err)
		return err
	}

	return nil
}
