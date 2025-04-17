package application

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/tentens-tech/shared-lock/internal/application/command/leasemanagement"
	"github.com/tentens-tech/shared-lock/internal/config"
	"github.com/tentens-tech/shared-lock/internal/infrastructure/cache"
	"github.com/tentens-tech/shared-lock/internal/infrastructure/metrics"
	"github.com/tentens-tech/shared-lock/internal/infrastructure/storage"
)

type Application struct {
	config            *config.Config
	leaseCache        *cache.Cache
	ctx               context.Context
	storageConnection storage.Storage
}

func New(ctx context.Context, config *config.Config, storageConnection storage.Storage, leaseCache *cache.Cache) *Application {
	return &Application{
		config:            config,
		leaseCache:        leaseCache,
		storageConnection: storageConnection,
		ctx:               ctx,
	}
}

func (a *Application) CreateLease(
	leaseTTL time.Duration,
	lease leasemanagement.Lease,
) (leaseStatus string, leaseID int64, err error) {
	defer func() {
		metrics.LeaseOperations.WithLabelValues(metrics.LeaseOperationGet, leaseStatus).Inc()
	}()

	cachedLeaseID := a.checkLeasePresenceInCache(lease.Key)
	if cachedLeaseID == 0 {
		leaseStatus, leaseID, err = leasemanagement.CreateLease(a.ctx, a.storageConnection, leaseTTL, lease)
		if err != nil {
			log.Errorf("%v", err)
			metrics.LeaseOperations.WithLabelValues(metrics.LeaseOperationGet, "error").Inc()

			return "", leaseID, err
		}

		log.Debugf("Adding to cache: %d", leaseID)
		a.addLeaseToCache(lease.Key, leaseStatus, leaseID, leaseTTL)

		return leaseStatus, leaseID, nil
	}

	log.Debugf("Lease already created with ID: %d", cachedLeaseID)
	leaseStatus = storage.StatusAccepted

	return leaseStatus, cachedLeaseID, nil
}

func (a *Application) ReviveLease(leaseID int64) error {
	err := leasemanagement.ReviveLease(a.ctx, a.storageConnection, leaseID)
	if err != nil {
		log.Errorf("Failed to prolong lease: %v", err)
		metrics.LeaseOperations.WithLabelValues(metrics.LeaseOperationProlong, "failure").Inc()
		return err
	}

	metrics.LeaseOperations.WithLabelValues(metrics.LeaseOperationProlong, "success").Inc()
	return nil
}

func (a *Application) checkLeasePresenceInCache(key string) int64 {
	if a.leaseCache == nil {
		return 0
	}

	if cachedValue, exists := a.leaseCache.Get(key); exists {
		if cachedLease, ok := cachedValue.(cache.LeaseCacheRecord); ok {
			log.Debugf("Cache hit for lease key: %v", key)
			return cachedLease.ID
		}
	}

	return 0
}

func (a *Application) addLeaseToCache(key string, status string, id int64, ttl time.Duration) {
	if a.leaseCache == nil {
		return
	}

	a.leaseCache.Set(key, cache.LeaseCacheRecord{
		Status: status,
		ID:     id,
	}, ttl)
}
