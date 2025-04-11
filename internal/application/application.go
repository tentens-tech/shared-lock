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
	Config            *config.Config
	LeaseCache        *cache.Cache
	Ctx               context.Context
	StorageConnection storage.Storage
}

func New(ctx context.Context, config *config.Config, storageConnection storage.Storage, leaseCache *cache.Cache) *Application {
	return &Application{
		Config:            config,
		LeaseCache:        leaseCache,
		StorageConnection: storageConnection,
		Ctx:               ctx,
	}
}

func (a *Application) CreateLease(
	leaseTTL time.Duration,
	lease leasemanagement.Lease,
) (leaseStatus string, leaseID int64, err error) {
	leaseStatus, leaseID, err = leasemanagement.CreateLease(a.Ctx, a.StorageConnection, leaseTTL, lease)
	if err != nil {
		log.Errorf("%v", err)
		metrics.LeaseOperations.WithLabelValues(metrics.LeaseOperationGet, "error").Inc()

		return "", 0, err
	}

	metrics.LeaseOperations.WithLabelValues(metrics.LeaseOperationGet, leaseStatus).Inc()
	return leaseStatus, leaseID, nil
}

func (a *Application) ReviveLease(leaseID int64) error {
	err := leasemanagement.ReviveLease(a.Ctx, a.StorageConnection, leaseID)
	if err != nil {
		log.Errorf("Failed to prolong lease: %v", err)
		metrics.LeaseOperations.WithLabelValues(metrics.LeaseOperationProlong, "failure").Inc()
		return err
	}

	metrics.LeaseOperations.WithLabelValues(metrics.LeaseOperationProlong, "success").Inc()
	return nil
}
