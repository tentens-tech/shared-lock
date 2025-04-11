package http

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	_ "net/http/pprof"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"github.com/tentens-tech/shared-lock/internal/application"
	"github.com/tentens-tech/shared-lock/internal/application/command/leasemanagement"
	"github.com/tentens-tech/shared-lock/internal/config"
	"github.com/tentens-tech/shared-lock/internal/infrastructure/metrics"
	"github.com/tentens-tech/shared-lock/internal/infrastructure/storage"
)

const (
	defaultLeaseTTLHeader = "x-lease-ttl"
	defaultLeaseDuration  = 10 * time.Second
)

type Server struct {
	app    *application.Application
	Server *http.Server
}

func New(app *application.Application) *Server {
	return &Server{
		app: app,
	}
}

func (s *Server) Start(cfg *config.ServerCfg) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/lease", s.handleLease)
	mux.HandleFunc("/keepalive", s.handleKeepalive)
	mux.HandleFunc("/health", s.handleHealth)
	mux.Handle("/metrics", promhttp.Handler())

	if cfg.PPROFEnabled {
		mux.HandleFunc("/debug/pprof/", http.HandlerFunc(http.DefaultServeMux.ServeHTTP))
	}

	s.Server = &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      mux,
		ReadTimeout:  cfg.Timeout.Read,
		WriteTimeout: cfg.Timeout.Write,
		IdleTimeout:  cfg.Timeout.Idle,
	}

	return s.Server.ListenAndServe()
}

func (s *Server) handleLease(w http.ResponseWriter, r *http.Request) {
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

	isLeasePresentInCache, err := s.app.CheckLeasePresenceInCache(lease.Key)
	if err != nil {
		log.Errorf("Failed to check lease presence in cache, %v", err)
	}

	if isLeasePresentInCache {
		leaseStatus, leaseID = s.app.GetLeaseFromCache(lease.Key)
	}

	if leaseStatus == "" {
		leaseTTL, err := time.ParseDuration(r.Header.Get(defaultLeaseTTLHeader))
		if err != nil {
			log.Warnf("Can't parse value of %v header. Using defaultLeaseDuration for %v", defaultLeaseTTLHeader, lease.Key)
			leaseTTL = defaultLeaseDuration
		}

		leaseStatus, leaseID, err = s.app.CreateLease(leaseTTL, lease)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		s.app.AddLeaseToCache(lease.Key, leaseStatus, leaseID, leaseTTL)
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
}

func (s *Server) handleKeepalive(w http.ResponseWriter, r *http.Request) {
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
	err = s.app.ReviveLease(leaseID)
	if err != nil {
		log.Warnf("Failed to prolong lease: %v", err)
		http.Error(w, "Failed to prolong lease", http.StatusNoContent)
		return
	}
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}
