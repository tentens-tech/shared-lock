package sharedLock

import (
	"log"
	"net/http"
	"tentens-tech/shared-lock/sharedLock/config"
	"tentens-tech/shared-lock/sharedLock/lease"
	"tentens-tech/shared-lock/sharedLock/lock"
	"time"
)

func newEtcdConnection(cfg *config.EtcdCfg) *lock.Lock {
	return lock.NewLock(cfg)
}

func newSharedLockServer(cfg *config.Config, etcdConnection *lock.Lock) error {
	http.HandleFunc("/lease", func(writer http.ResponseWriter, request *http.Request) {
		lease.CreateLease(cfg, etcdConnection, writer, request)
	})
	http.HandleFunc("/keepalive", func(writer http.ResponseWriter, request *http.Request) {
		lease.ReviveLease(cfg, etcdConnection, writer, request)
	})
	http.HandleFunc("/health", func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusOK)
	})

	log.Printf("Starting shared-lock server at %v", cfg.ServerPort)

	server := &http.Server{
		Addr:         cfg.ServerPort,
		Handler:      nil, // You can set your handler here
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	server.ListenAndServe()

	return server.ListenAndServe()
}

func StartSharedLock() error {
	cfg := config.Load()
	etcdConnection := newEtcdConnection(&cfg.EtcdCfg)

	err := newSharedLockServer(cfg, etcdConnection)
	if err != nil {
		return err
	}

	return nil
}
