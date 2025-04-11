package application

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/tentens-tech/shared-lock/internal/application/command/leasemanagement"
	"github.com/tentens-tech/shared-lock/internal/config"
	"github.com/tentens-tech/shared-lock/internal/infrastructure/cache"
	"github.com/tentens-tech/shared-lock/internal/infrastructure/metrics"
	"github.com/tentens-tech/shared-lock/internal/infrastructure/storage"
)

const (
	defaultLeaseTTLHeader = "x-lease-ttl"
)

type leaseCacheRecord struct {
	Status string
	ID     int64
}

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

func GetLeaseHandler(ctx context.Context, configuration *config.Config, storageConnection storage.Storage, leaseCache *cache.Cache) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		defer func() {
			metrics.LeaseOperationDuration.WithLabelValues(metrics.LeaseOperationGet).Observe(time.Since(start).Seconds())
		}()

		var err error
		var lease leasemanagement.Lease
		var leaseID int64
		var leaseStatus string

		body, err := io.ReadAll(r.Body)
		if err != nil {
			log.Errorf("Failed to read request body, %v", err)
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}

		log.Debugf("Request body: %v", string(body))
		err = json.Unmarshal(body, &lease)
		if err != nil {
			log.Errorf("Failed to unmarshal request body, %v", err)
			http.Error(w, "Failed to unmarshal request body", http.StatusBadRequest)
			return
		}

		if leaseCache != nil {
			if cachedValue, exists := leaseCache.Get(lease.Key); exists {
				if cachedLease, ok := cachedValue.(leaseCacheRecord); ok {
					leaseStatus = cachedLease.Status
					leaseID = cachedLease.ID
					log.Debugf("Cache hit for lease key: %v", lease.Key)
					metrics.CacheOperations.WithLabelValues(metrics.LeaseOperationGet, "success").Inc()
				}
			}
		}

		if leaseStatus == "" {
			leaseTTL, err := time.ParseDuration(r.Header.Get(defaultLeaseTTLHeader))
			if err != nil {
				log.Warnf("Can't parse value of %v header. Using defaultLeaseDurationSeconds for %v", defaultLeaseTTLHeader, lease.Key)
				leaseTTL = leasemanagement.DefaultLeaseDurationSeconds
			}

			leaseStatus, leaseID, err = leasemanagement.CreateLease(ctx, storageConnection, body, leaseTTL, lease)
			if err != nil {
				log.Errorf("%v", err)
				metrics.LeaseOperations.WithLabelValues(metrics.LeaseOperationGet, "error").Inc()
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			if leaseCache != nil {
				leaseCache.Set(lease.Key, leaseCacheRecord{
					Status: leaseStatus,
					ID:     leaseID,
				}, leaseTTL)
			}
		}

		switch leaseStatus {
		case storage.StatusAccepted:
			w.WriteHeader(http.StatusAccepted)
		case storage.StatusCreated:
			w.WriteHeader(http.StatusCreated)
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}

		_, err = w.Write([]byte(fmt.Sprintf("%v", leaseID)))
		if err != nil {
			log.Errorf("Failed to write response for /lease endpoint, %v", err)
			return
		}

		metrics.LeaseOperations.WithLabelValues(metrics.LeaseOperationGet, leaseStatus).Inc()
	}
}

func KeepaliveHandler(ctx context.Context, configuration *config.Config, storageConnection storage.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var err error
		var leaseID int64

		body, err := io.ReadAll(r.Body)
		if err != nil {
			log.Errorf("Failed to read request body, %v", err)
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}

		log.Debugf("Request body: %v", string(body))
		leaseID, err = strconv.ParseInt(string(body), 10, 64)
		if err != nil {
			log.Errorf("Failed to parse lease id from string, leaseIDString: %v, %v", string(body), err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		log.Debugf("Trying to revive lease: %v", leaseID)
		err = leasemanagement.ReviveLease(ctx, storageConnection, leaseID)
		if err != nil {
			log.Warnf("Failed to prolong lease: %v", err)
			http.Error(w, "Failed to prolong lease", http.StatusNoContent)
			metrics.LeaseOperations.WithLabelValues(metrics.LeaseOperationProlong, "failure").Inc()
			return
		}

		metrics.LeaseOperations.WithLabelValues(metrics.LeaseOperationProlong, "success").Inc()
	}
}

func HealthHandler() http.HandlerFunc {
	return func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusOK)
	}
}
