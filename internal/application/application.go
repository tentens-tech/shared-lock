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

	router.HandleFunc("/lease", getLeaseHandler(configuration))
	router.HandleFunc("/keepalive", keepaliveHandler(configuration))

	return router
}

func newStorageConnection(cfg *config.Config) storage.Connection {
	return etcd.NewConnection(cfg)
}

func getLeaseHandler(configuration *config.Config) http.HandlerFunc {
	storageConnection := newStorageConnection(configuration)

	return func(w http.ResponseWriter, r *http.Request) {
		var err error
		var lease leaseManagement.Lease
		var leaseID int64
		var getLeaseStatus string

		body, _ := io.ReadAll(r.Body)
		err = json.Unmarshal(body, &lease)
		if err != nil {
			log.Error(err)
			http.Error(w, "Failed to unmarshal request body", http.StatusBadRequest)
		}

		leaseTTLString := r.Header.Get(DefaultLeaseTTLHeader)

		getLeaseStatus, leaseID, err = leaseManagement.CreateLease(configuration, storageConnection, body, leaseTTLString, lease)
		if err != nil {
			log.Errorf("%v", err)
			w.WriteHeader(http.StatusInternalServerError)
		}

		switch getLeaseStatus {
		case "accepted":
			w.WriteHeader(http.StatusAccepted)
		case "created":
			w.WriteHeader(http.StatusCreated)
		}

		_, err = w.Write([]byte(fmt.Sprintf("%v", leaseID)))
		if err != nil {
			log.Errorf("Failed to write response for /lease endpoint, %v", err)
		}
	}
}

func keepaliveHandler(configuration *config.Config) http.HandlerFunc {
	storageConnection := newStorageConnection(configuration)

	return func(w http.ResponseWriter, r *http.Request) {
		var err error
		var lease leaseManagement.Lease
		var leaseID int64

		body, _ := io.ReadAll(r.Body)
		err = json.Unmarshal(body, &lease)
		if err != nil {
			log.Error(err)
			http.Error(w, "Failed to unmarshal request body", http.StatusBadRequest)
		}

		leaseID, err = strconv.ParseInt(string(body), 10, 64)
		if err != nil {
			log.Errorf("Failed to parse lease id from string, leaseIDString: %v, %v", string(body), err)
			w.WriteHeader(http.StatusInternalServerError)
		}

		err = leaseManagement.ReviveLease(body, storageConnection, leaseID)
		if err != nil {
			log.Warnf("Failed to prolong lease: %v", err)
			http.Error(w, "Failed to prolong lease", http.StatusNoContent)
		}
	}
}
