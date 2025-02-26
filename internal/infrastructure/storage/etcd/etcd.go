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
	DefaultLeaseValue = "lock-value"
)

type Connection struct {
	Cli *clientv3.Client
}

func NewConnection(cfg *config.Config) *Connection {
	etcdConnection := &Connection{Cli: func() *clientv3.Client {
		var (
			err       error
			tlsConfig *tls.Config
		)

		tlsConfig = func() *tls.Config {
			if !cfg.Etcd.TLSEnabled {
				return nil
			}

			// Configure TLS
			tlsInfo := transport.TLSInfo{
				TrustedCAFile: cfg.Etcd.ServerCACertPath,
				CertFile:      cfg.Etcd.ServerClientCertPath,
				KeyFile:       cfg.Etcd.ServerClientKeyPath,
			}

			tlsConfig, err = tlsInfo.ClientConfig()
			if err != nil {
				log.Fatalf("Failed to create TLS configuration for etcd endpoints: %v", err)
			}

			return tlsConfig
		}()

		cli, err := clientv3.New(clientv3.Config{
			Endpoints:   cfg.Etcd.EtcdAddrList,
			DialTimeout: 5 * time.Second,
			TLS:         tlsConfig,
		})
		if err != nil {
			log.Fatalf("Failed to connect to etcd, %v", err)
		}

		return cli
	}()}

	return etcdConnection
}

func (con *Connection) CheckLeasePresence(ctx context.Context, key string) (bool, error) {
	var err error
	getCtx, cancel := context.WithTimeout(ctx, 5*time.Second)

	resp, err := con.Cli.Get(getCtx, key)
	cancel()
	if err != nil {
		return false, fmt.Errorf("failed to get key from etcd: %v", err)
	}
	if len(resp.Kvs) != 0 {
		log.Debugf("Lock %v, already exists", key)
		return true, nil
	}

	return false, fmt.Errorf("lock %v does not exist", key)
}

func (con *Connection) CreateLease(ctx context.Context, key string, leaseTTL int64, data []byte) (string, int64, error) {
	var leaseResp *clientv3.LeaseGrantResponse
	var err error
	var value string

	if data == nil {
		value = DefaultLeaseValue
	} else {
		value = string(data)
	}

	leaseResp, err = con.Cli.Grant(ctx, leaseTTL)
	if err != nil {
		return "", 0, fmt.Errorf("failed to create lease: %v", err)
	}

	var TxnResp *clientv3.TxnResponse
	TxnResp, err = con.Cli.Txn(ctx).
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

func (con *Connection) KeepLeaseOnce(ctx context.Context, leaseID int64) error {
	ctxWithCancel, cancel := context.WithCancel(ctx)
	defer cancel()

	_, err := con.Cli.KeepAliveOnce(ctxWithCancel, clientv3.LeaseID(leaseID))
	if err != nil {
		return err
	}

	log.Printf("KeepAlive lease: %v", leaseID)
	return nil
}
