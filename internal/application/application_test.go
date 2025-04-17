package application

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/tentens-tech/shared-lock/internal/application/command/leasemanagement"
	"github.com/tentens-tech/shared-lock/internal/config"
	"github.com/tentens-tech/shared-lock/internal/infrastructure/cache"
	"github.com/tentens-tech/shared-lock/internal/infrastructure/storage"
	"github.com/tentens-tech/shared-lock/internal/infrastructure/storage/mock"
)

func createTestConfig() *config.Config {
	cfg := config.NewConfig()
	cfg.Storage.Type = "mock"
	return cfg
}

func TestApplication_CreateLease(t *testing.T) {
	tests := []struct {
		name           string
		leaseTTL       time.Duration
		lease          leasemanagement.Lease
		expectedStatus string
		expectedID     int64
		expectError    bool
	}{
		{
			name:     "Successful lease creation",
			leaseTTL: time.Minute,
			lease: leasemanagement.Lease{
				Key:   "test-key",
				Value: "test-value",
				Labels: map[string]string{
					"test": "label",
				},
			},
			expectedStatus: storage.StatusCreated,
			expectedID:     123,
			expectError:    false,
		},
		{
			name:     "Lease already exists",
			leaseTTL: time.Minute,
			lease: leasemanagement.Lease{
				Key:   "existing-key",
				Value: "test-value",
			},
			expectedStatus: storage.StatusAccepted,
			expectedID:     123,
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			cfg := createTestConfig()
			storageConnection := mock.New()
			leaseCache := cache.New(1000)

			if tt.name == "Lease already exists" {
				leaseCache.Set(tt.lease.Key, cache.LeaseCacheRecord{
					Status: storage.StatusCreated,
					ID:     123,
				}, tt.leaseTTL)
			}

			app := New(ctx, cfg, storageConnection, leaseCache)

			status, id, err := app.CreateLease(tt.leaseTTL, tt.lease)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedStatus, status)
				assert.Equal(t, tt.expectedID, id)
			}
		})
	}
}

func TestApplication_ReviveLease(t *testing.T) {
	tests := []struct {
		name        string
		leaseID     int64
		expectError bool
	}{
		{
			name:        "Successful lease revival",
			leaseID:     123,
			expectError: false,
		},
		{
			name:        "Failed lease revival",
			leaseID:     999,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			cfg := createTestConfig()
			storageConnection := mock.New()

			app := New(ctx, cfg, storageConnection, nil)

			err := app.ReviveLease(tt.leaseID)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestApplication_ConcurrentLeaseOperations(t *testing.T) {
	ctx := context.Background()
	cfg := createTestConfig()
	storageConnection := mock.New()
	leaseCache := cache.New(1000)

	app := New(ctx, cfg, storageConnection, leaseCache)

	numOperations := 100
	var wg sync.WaitGroup
	wg.Add(numOperations)

	for i := 0; i < numOperations; i++ {
		go func(index int) {
			defer wg.Done()

			lease := leasemanagement.Lease{
				Key:   fmt.Sprintf("concurrent-key-%d", index),
				Value: fmt.Sprintf("value-%d", index),
			}

			status, id, err := app.CreateLease(time.Minute, lease)
			assert.NoError(t, err)
			assert.NotEmpty(t, status)
			assert.NotZero(t, id)

			err = app.ReviveLease(id)
			assert.NoError(t, err)
		}(i)
	}

	wg.Wait()
}

func TestApplication_CacheDisabled(t *testing.T) {
	ctx := context.Background()
	cfg := createTestConfig()
	storageConnection := mock.New()

	app := New(ctx, cfg, storageConnection, nil)

	lease := leasemanagement.Lease{
		Key:   "test-key",
		Value: "test-value",
	}

	status, id, err := app.CreateLease(time.Minute, lease)
	assert.NoError(t, err)
	assert.NotEmpty(t, status)
	assert.NotZero(t, id)

	err = app.ReviveLease(id)
	assert.NoError(t, err)
}

func TestApplication_ErrorHandling(t *testing.T) {
	ctx := context.Background()
	cfg := createTestConfig()
	storageConnection := mock.New()
	leaseCache := cache.New(1000)

	app := New(ctx, cfg, storageConnection, leaseCache)

	lease := leasemanagement.Lease{
		Key:   "error-key",
		Value: "error-value",
	}

	status, id, err := app.CreateLease(time.Minute, lease)
	assert.NoError(t, err)
	assert.NotEmpty(t, status)
	assert.NotZero(t, id)

	err = app.ReviveLease(123)
	assert.NoError(t, err)
}
