package etcd

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"github.com/tentens-tech/shared-lock/internal/config"

	log "github.com/sirupsen/logrus"
	"github.com/tentens-tech/shared-lock/internal/infrastructure/storage"
	"go.etcd.io/etcd/client/pkg/v3/transport"
	clientv3 "go.etcd.io/etcd/client/v3"
)

const (
	defaultLeaseValue  = "lock-value"
	defaultDialTimeout = 5 * time.Second
)

type Etcd struct {
	Client *clientv3.Client
}

func New(cfg *config.Config) (*Etcd, error) {
	tlsConfig, err := generateTLSConfig(cfg)
	if err != nil {
		return nil, err
	}

	client, err := clientv3.New(clientv3.Config{
		Endpoints:   cfg.Etcd.EtcdAddrList,
		DialTimeout: defaultDialTimeout,
		TLS:         tlsConfig,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to etcd, %v", err)
	}

	return &Etcd{
		Client: client,
	}, nil
}

func generateTLSConfig(cfg *config.Config) (*tls.Config, error) {
	if !cfg.Etcd.TLSEnabled {
		return nil, nil
	}

	var tlsConfig *tls.Config
	var err error

	tlsInfo := transport.TLSInfo{
		TrustedCAFile: cfg.Etcd.ServerCACertPath,
		CertFile:      cfg.Etcd.ServerClientCertPath,
		KeyFile:       cfg.Etcd.ServerClientKeyPath,
	}

	tlsConfig, err = tlsInfo.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create TLS configuration for etcd endpoints: %v", err)
	}

	return tlsConfig, nil
}

func (etcd *Etcd) CheckLeasePresence(ctx context.Context, key string) (bool, error) {
	var err error
	getCtx, cancel := context.WithTimeout(ctx, 5*time.Second)

	resp, err := etcd.Client.Get(getCtx, key)
	cancel()
	if err != nil {
		return false, fmt.Errorf("failed to get key from etcd: %v", err)
	}
	if len(resp.Kvs) != 0 {
		log.Debugf("Lock %v, already exists", key)
		return true, nil
	}

	return false, nil
}

func (etcd *Etcd) CreateLease(ctx context.Context, key string, leaseTTL int64, data []byte) (string, int64, error) {
	var leaseResp *clientv3.LeaseGrantResponse
	var err error
	var value string

	if data == nil {
		value = defaultLeaseValue
	} else {
		value = string(data)
	}

	log.Debugf("Creating lease for the key: %v", key)
	leaseResp, err = etcd.Client.Grant(ctx, leaseTTL)
	if err != nil {
		return "", 0, fmt.Errorf("failed to create lease: %v", err)
	}

	var TxnResp *clientv3.TxnResponse
	TxnResp, err = etcd.Client.Txn(ctx).
		If(clientv3.Compare(clientv3.CreateRevision(key), "=", 0)).
		Then(clientv3.OpPut(key, value, clientv3.WithLease(leaseResp.ID))).
		Commit()
	if err != nil {
		return "", 0, err
	}

	if !TxnResp.Succeeded {
		log.Warnf("Lease race")
		return storage.StatusAccepted, 0, nil
	}

	log.Printf("%v key created with a new lease %v", key, leaseResp.ID)
	return storage.StatusCreated, int64(leaseResp.ID), nil
}

func (etcd *Etcd) KeepLeaseOnce(ctx context.Context, leaseID int64) error {
	ctxWithCancel, cancel := context.WithCancel(ctx)
	defer cancel()

	_, err := etcd.Client.KeepAliveOnce(ctxWithCancel, clientv3.LeaseID(leaseID))
	if err != nil {
		return err
	}

	log.Printf("KeepAlive lease: %v", leaseID)
	return nil
}
