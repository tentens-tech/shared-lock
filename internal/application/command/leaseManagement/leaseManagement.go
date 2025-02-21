package leaseManagement

import (
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/tentens-tech/shared-lock/internal/config"
	"github.com/tentens-tech/shared-lock/internal/infrastructure/storage"
)

const (
	DefaultPrefix               = "/shared-lock/"
	DefaultLeaseDurationSeconds = 10
)

func CreateLease(cfg *config.Config, storageConnection storage.Connection, data []byte, leaseTTLString string, lease Lease) (string, int64, error) {
	var err error
	var leaseTTL time.Duration
	var leaseID int64
	var getLeaseStatus string

	leaseTTL, err = time.ParseDuration(leaseTTLString)
	if err != nil {
		log.Warnf("Use defaultLeaseDurationSeconds for %v", lease.Key)
		leaseTTL = DefaultLeaseDurationSeconds
	}

	key := DefaultPrefix + lease.Key

	log.Debugf("Get lease for key: %v, with ttl: %v", key, leaseTTL)
	getLeaseStatus, leaseID, err = storageConnection.GetLease(key, data, int64(leaseTTL.Seconds()))
	if err != nil {
		log.Errorf("%v", err)
		return "", 0, err
	}

	return getLeaseStatus, leaseID, nil
}

func ReviveLease(data []byte, storageConnection storage.Connection, leaseID int64) error {
	err := storageConnection.KeepLeaseOnce(leaseID)
	if err != nil {
		log.Warnf("Failed to prolong lease: %v", err)
		return err
	}

	return nil
}
