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
	"github.com/tentens-tech/shared-lock/internal/infrastructure/storage"
	"github.com/tentens-tech/shared-lock/internal/infrastructure/storage/etcd"
)

const (
	defaultLeaseTTLHeader = "x-lease-ttl"
)

type leaseCacheRecord struct {
	Status string
	ID     int64
}

func NewRouter(ctx context.Context, configuration *config.Config, leaseCache *cache.Cache) *http.ServeMux {
	router := http.NewServeMux()

	router.HandleFunc("/lease", getLeaseHandler(ctx, configuration, leaseCache))
	router.HandleFunc("/keepalive", keepaliveHandler(ctx, configuration))
	router.HandleFunc("/health", healthHandler())

	return router
}

func newStorageConnection(cfg *config.Config) (storage.Storage, error) {
	storageConnection, err := etcd.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create etcd storage connection, %v", err)
	}

	return storageConnection, nil
}

func getLeaseHandler(ctx context.Context, configuration *config.Config, leaseCache *cache.Cache) http.HandlerFunc {
	storageConnection, err := newStorageConnection(configuration)
	if err != nil {
		log.Errorf("Failed to connect to storage adapater, %v", err)
		return nil
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var err error
		var lease leasemanagement.Lease
		var leaseID int64
		var leaseStatus string

		body, err := io.ReadAll(r.Body)
		if err != nil {
			log.Errorf("Failed to read request body, %v", err)
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
		}

		log.Debugf("Request body: %v", string(body))
		err = json.Unmarshal(body, &lease)
		if err != nil {
			log.Errorf("Failed to unmarshal request body, %v", err)
			http.Error(w, "Failed to unmarshal request body", http.StatusBadRequest)
		}

		if cachedValue, exists := leaseCache.Get(lease.Key); exists {
			if cachedLease, ok := cachedValue.(leaseCacheRecord); ok {
				leaseStatus = cachedLease.Status
				leaseID = cachedLease.ID
				log.Debugf("Cache hit for lease key: %v", lease.Key)
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
				w.WriteHeader(http.StatusInternalServerError)
			}

			leaseCache.Set(lease.Key, leaseCacheRecord{
				Status: leaseStatus,
				ID:     leaseID,
			}, leaseTTL)
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
		}
	}
}

func keepaliveHandler(ctx context.Context, configuration *config.Config) http.HandlerFunc {
	storageConnection, err := newStorageConnection(configuration)
	if err != nil {
		log.Errorf("Failed to connect to storage adapater, %v", err)
		return nil
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var err error
		var leaseID int64

		body, err := io.ReadAll(r.Body)
		if err != nil {
			log.Errorf("Failed to read request body, %v", err)
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
		}

		log.Debugf("Request body: %v", string(body))
		leaseID, err = strconv.ParseInt(string(body), 10, 64)
		if err != nil {
			log.Errorf("Failed to parse lease id from string, leaseIDString: %v, %v", string(body), err)
			w.WriteHeader(http.StatusInternalServerError)
		}

		log.Debugf("Trying to revive lease: %v", leaseID)
		err = leasemanagement.ReviveLease(ctx, storageConnection, leaseID)
		if err != nil {
			log.Warnf("Failed to prolong lease: %v", err)
			http.Error(w, "Failed to prolong lease", http.StatusNoContent)
		}
	}
}

func healthHandler() http.HandlerFunc {
	return func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusOK)
	}
}
