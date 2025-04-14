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
	leaseStatus, leaseID, err = leasemanagement.CreateLease(a.ctx, a.storageConnection, leaseTTL, lease)
	if err != nil {
		log.Errorf("%v", err)
		metrics.LeaseOperations.WithLabelValues(metrics.LeaseOperationGet, "error").Inc()

		return "", 0, err
	}

	metrics.LeaseOperations.WithLabelValues(metrics.LeaseOperationGet, leaseStatus).Inc()
	return leaseStatus, leaseID, nil
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

func (a *Application) CheckLeasePresenceInCache(key string) bool {
	if a.leaseCache == nil {
		return false
	}

	if _, exists := a.leaseCache.Get(key); exists {
		return true
	}

	return false
}

func (a *Application) GetLeaseFromCache(key string) (string, int64) {
	if cachedValue, exists := a.leaseCache.Get(key); exists {
		if cachedLease, ok := cachedValue.(cache.LeaseCacheRecord); ok {
			log.Debugf("Cache hit for lease key: %v", key)
			metrics.CacheOperations.WithLabelValues(metrics.LeaseOperationGet, "success").Inc()

			return cachedLease.Status, cachedLease.ID
		}
	}

	return "", 0
}

func (a *Application) AddLeaseToCache(key string, status string, id int64, ttl time.Duration) {
	if a.leaseCache == nil {
		return
	}

	a.leaseCache.Set(key, cache.LeaseCacheRecord{
		Status: status,
		ID:     id,
	}, ttl)
}
