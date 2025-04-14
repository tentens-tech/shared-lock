package bootstrap

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/tentens-tech/shared-lock/internal/application"
	"github.com/tentens-tech/shared-lock/internal/config"
	"github.com/tentens-tech/shared-lock/internal/infrastructure/cache"
	"github.com/tentens-tech/shared-lock/internal/infrastructure/storage"
	"github.com/tentens-tech/shared-lock/internal/infrastructure/storage/etcd"
	"github.com/tentens-tech/shared-lock/internal/infrastructure/storage/mock"
)

func newStorageConnection(cfg *config.Config) (storage.Storage, error) {
	if cfg.Storage.Type == "etcd" {
		storageConnection, err := etcd.New(cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to create etcd storage connection, %v", err)
		}
		return storageConnection, nil
	} else if cfg.Storage.Type == "mock" {
		return &mock.Storage{}, nil
	}

	return nil, fmt.Errorf("unsupported storage type: %v", cfg.Storage.Type)
}

func newCache(cfg *config.Config) (*cache.Cache, error) {
	if cfg.Cache.Enabled {
		log.Info("Cache is enabled")
		return cache.New(cfg.Cache.Size), nil
	}

	log.Info("Cache is disabled")
	return nil, nil
}

func NewApplication(ctx context.Context, cfg *config.Config) (*application.Application, error) {
	leaseCache, err := newCache(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache: %v", err)
	}

	storageConnection, err := newStorageConnection(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage connection: %v", err)
	}

	app := application.New(ctx, cfg, storageConnection, leaseCache)

	return app, nil
}
