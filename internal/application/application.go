package application

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	log "github.com/sirupsen/logrus"
	"github.com/tentens-tech/shared-lock/internal/application/command/leaseManagement"
	"github.com/tentens-tech/shared-lock/internal/config"
	"github.com/tentens-tech/shared-lock/internal/infrastructure/storage"
	"github.com/tentens-tech/shared-lock/internal/infrastructure/storage/etcd"
)

const (
	DefaultLeaseTTLHeader = "x-lease-ttl"
)

func NewRouter(ctx context.Context, configuration *config.Config) *http.ServeMux {
	router := http.NewServeMux()

	router.HandleFunc("/lease", getLeaseHandler(ctx, configuration))
	router.HandleFunc("/keepalive", keepaliveHandler(ctx, configuration))
	router.HandleFunc("/health", healthHandler())

	return router
}

func newStorageConnection(cfg *config.Config) storage.Storage {
	return etcd.NewConnection(cfg)
}

func getLeaseHandler(ctx context.Context, configuration *config.Config) http.HandlerFunc {
	storageConnection := newStorageConnection(configuration)

	return func(w http.ResponseWriter, r *http.Request) {
		var err error
		var lease leaseManagement.Lease
		var leaseID int64
		var leaseStatus string

		body, err := io.ReadAll(r.Body)
		if err != nil {
			log.Errorf("Failed to read request body, %v", err)
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
		}

		err = json.Unmarshal(body, &lease)
		if err != nil {
			log.Errorf("Failed to unmarshal request body, %v", err)
			http.Error(w, "Failed to unmarshal request body", http.StatusBadRequest)
		}

		leaseTTLString := r.Header.Get(DefaultLeaseTTLHeader)

		leaseStatus, leaseID, err = leaseManagement.CreateLease(ctx, configuration, storageConnection, body, leaseTTLString, lease)
		if err != nil {
			log.Errorf("%v", err)
			w.WriteHeader(http.StatusInternalServerError)
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
	storageConnection := newStorageConnection(configuration)

	return func(w http.ResponseWriter, r *http.Request) {
		var err error
		var lease leaseManagement.Lease
		var leaseID int64

		body, _ := io.ReadAll(r.Body)
		err = json.Unmarshal(body, &lease)
		if err != nil {
			log.Errorf("Failed to unmarshal request body, %v", err)
			http.Error(w, "Failed to unmarshal request body", http.StatusBadRequest)
		}

		leaseID, err = strconv.ParseInt(string(body), 10, 64)
		if err != nil {
			log.Errorf("Failed to parse lease id from string, leaseIDString: %v, %v", string(body), err)
			w.WriteHeader(http.StatusInternalServerError)
		}

		err = leaseManagement.ReviveLease(ctx, storageConnection, leaseID)
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
