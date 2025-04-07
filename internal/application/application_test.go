package application

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/tentens-tech/shared-lock/internal/config"
	"github.com/tentens-tech/shared-lock/internal/infrastructure/cache"
	"github.com/tentens-tech/shared-lock/internal/infrastructure/storage"
)

func createTestConfig() *config.Config {
	cfg := config.NewConfig()
	cfg.Storage.Type = "mock"

	return cfg
}

func TestGetLeaseHandler(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		leaseTTL       string
		cacheKey       string
		cacheValue     interface{}
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "Cache Hit - Accepted",
			requestBody: map[string]string{
				"key": "test-key",
			},
			leaseTTL: "1m",
			cacheKey: "test-key",
			cacheValue: leaseCacheRecord{
				Status: storage.StatusAccepted,
				ID:     123,
			},
			expectedStatus: http.StatusAccepted,
			expectedBody:   "123",
		},
		{
			name: "Cache Hit - Created",
			requestBody: map[string]string{
				"key": "test-key",
			},
			leaseTTL: "1m",
			cacheKey: "test-key",
			cacheValue: leaseCacheRecord{
				Status: storage.StatusCreated,
				ID:     456,
			},
			expectedStatus: http.StatusCreated,
			expectedBody:   "456",
		},
		{
			name: "Cache Miss - New Lease",
			requestBody: map[string]string{
				"key": "new-key",
			},
			leaseTTL:       "1m",
			expectedStatus: http.StatusCreated,
			expectedBody:   "123",
		},
		{
			name:           "Invalid Request Body",
			requestBody:    "invalid json",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Failed to unmarshal request body\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			cfg := createTestConfig()
			leaseCache := cache.New(1000)

			if tt.cacheValue != nil {
				leaseCache.Set(tt.cacheKey, tt.cacheValue, time.Minute)
			}

			var body []byte
			var err error
			if str, ok := tt.requestBody.(string); ok {
				body = []byte(str)
			} else {
				body, err = json.Marshal(tt.requestBody)
				assert.NoError(t, err)
			}

			req := httptest.NewRequest(http.MethodPost, "/lease", bytes.NewBuffer(body))
			if tt.leaseTTL != "" {
				req.Header.Set("x-lease-ttl", tt.leaseTTL)
			}

			rr := httptest.NewRecorder()

			handler := GetLeaseHandler(ctx, cfg, leaseCache)
			assert.NotNil(t, handler)

			handler.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)
			assert.Equal(t, tt.expectedBody, rr.Body.String())
		})
	}
}

func TestGetLeaseHandlerConcurrent(t *testing.T) {
	ctx := context.Background()
	cfg := createTestConfig()
	leaseCache := cache.New(1000)

	leaseBody := map[string]string{
		"key": "concurrent-test-key",
	}
	body, err := json.Marshal(leaseBody)
	assert.NoError(t, err)

	numRequests := 100
	var wg sync.WaitGroup
	wg.Add(numRequests)

	for i := 0; i < numRequests; i++ {
		go func() {
			defer wg.Done()

			req := httptest.NewRequest(http.MethodPost, "/lease", bytes.NewBuffer(body))
			req.Header.Set("x-lease-ttl", "1m")

			rr := httptest.NewRecorder()
			handler := GetLeaseHandler(ctx, cfg, leaseCache)
			handler.ServeHTTP(rr, req)

			assert.Equal(t, http.StatusCreated, rr.Code)
		}()
	}

	wg.Wait()

	cachedValue, exists := leaseCache.Get("concurrent-test-key")
	assert.True(t, exists)
	assert.NotNil(t, cachedValue)
}

func TestGetLeaseHandlerMemoryUsage(t *testing.T) {
	ctx := context.Background()
	cfg := createTestConfig()
	leaseCache := cache.New(1000)

	var m runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m)
	initialAlloc := m.Alloc

	numLeases := 1000
	for i := 0; i < numLeases; i++ {
		leaseBody := map[string]string{
			"key": fmt.Sprintf("memory-test-key-%d", i),
		}
		body, err := json.Marshal(leaseBody)
		assert.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/lease", bytes.NewBuffer(body))
		req.Header.Set("x-lease-ttl", "1m")

		rr := httptest.NewRecorder()
		handler := GetLeaseHandler(ctx, cfg, leaseCache)
		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusCreated, rr.Code)
	}

	runtime.GC()
	runtime.ReadMemStats(&m)

	var memoryAlloc uint64
	if m.Alloc > initialAlloc {
		memoryAlloc = m.Alloc - initialAlloc
	} else {
		memoryAlloc = 0
	}

	t.Logf("Memory allocated for %d leases: %v bytes (%.2f MB)",
		numLeases, memoryAlloc, float64(memoryAlloc)/1024/1024)

	maxMemoryMB := uint64(50 * 1024 * 1024)
	assert.LessOrEqual(t, memoryAlloc, maxMemoryMB,
		"Memory usage should be less than 50MB for 1000 leases")
}
