package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tentens-tech/shared-lock/internal/application"
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

func createTestApplication(ctx context.Context, cfg *config.Config, storageConnection storage.Storage, leaseCache *cache.Cache) *application.Application {
	return application.New(ctx, cfg, storageConnection, leaseCache)
}

func TestGetLeaseHandler(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Cache Hit - Accepted",
			requestBody:    `{"key": "existing-key", "value": "test-value"}`,
			expectedStatus: http.StatusCreated,
			expectedBody:   "123",
		},
		{
			name:           "Cache Hit - Created",
			requestBody:    `{"key": "new-key", "value": "test-value"}`,
			expectedStatus: http.StatusCreated,
			expectedBody:   "123",
		},
		{
			name:           "Cache Miss - New Lease",
			requestBody:    `{"key": "cache-miss-key", "value": "test-value"}`,
			expectedStatus: http.StatusCreated,
			expectedBody:   "123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			cfg := createTestConfig()
			storageConnection := mock.New()
			leaseCache := cache.New(1000)

			app := application.New(ctx, cfg, storageConnection, leaseCache)
			server := New(app)

			req := httptest.NewRequest(http.MethodPost, "/lease", strings.NewReader(tt.requestBody))
			rec := httptest.NewRecorder()

			server.handleLease(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)
			assert.Equal(t, tt.expectedBody, rec.Body.String())
		})
	}
}

func TestGetLeaseHandlerConcurrent(t *testing.T) {
	ctx := context.Background()
	cfg := createTestConfig()
	leaseCache := cache.New(1000)
	storageConnection := mock.New()

	leaseBody := map[string]string{
		"key": "concurrent-test-key",
	}
	body, err := json.Marshal(leaseBody)
	assert.NoError(t, err)

	numRequests := 100
	var wg sync.WaitGroup
	wg.Add(numRequests)

	app := createTestApplication(ctx, cfg, storageConnection, leaseCache)
	server := New(app)

	for i := 0; i < numRequests; i++ {
		go func() {
			defer wg.Done()

			req := httptest.NewRequest(http.MethodPost, "/lease", bytes.NewBuffer(body))
			req.Header.Set("x-lease-ttl", "1m")

			rr := httptest.NewRecorder()
			server.handleLease(rr, req)

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
	storageConnection := mock.New()

	var m runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m)
	initialAlloc := m.Alloc

	numLeases := 1000
	app := createTestApplication(ctx, cfg, storageConnection, leaseCache)
	server := New(app)

	for i := 0; i < numLeases; i++ {
		leaseBody := map[string]string{
			"key": fmt.Sprintf("memory-test-key-%d", i),
		}
		body, err := json.Marshal(leaseBody)
		assert.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/lease", bytes.NewBuffer(body))
		req.Header.Set("x-lease-ttl", "1m")

		rr := httptest.NewRecorder()
		server.handleLease(rr, req)

		assert.Equal(t, http.StatusCreated, rr.Code)
	}

	runtime.GC()
	runtime.ReadMemStats(&m)

	memoryAlloc := m.Alloc - initialAlloc
	t.Logf("Memory allocated for %d leases: %v bytes (%.2f MB)",
		numLeases, memoryAlloc, float64(memoryAlloc)/1024/1024)

	maxMemoryMB := uint64(50 * 1024 * 1024)
	assert.LessOrEqual(t, memoryAlloc, maxMemoryMB,
		"Memory usage should be less than 50MB for 1000 leases")
}

func TestKeepaliveHandler(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Successful keepalive",
			requestBody:    "123",
			expectedStatus: http.StatusOK,
			expectedBody:   "",
		},
		{
			name:           "Invalid lease ID format",
			requestBody:    "invalid-id",
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "",
		},
		{
			name:           "Empty request body",
			requestBody:    "",
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			cfg := createTestConfig()
			storageConnection := mock.New()

			req := httptest.NewRequest(http.MethodPost, "/keepalive", bytes.NewBufferString(tt.requestBody))
			rr := httptest.NewRecorder()

			app := createTestApplication(ctx, cfg, storageConnection, nil)
			server := New(app)
			server.handleKeepalive(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)
			if tt.expectedBody != "" {
				assert.Equal(t, tt.expectedBody, rr.Body.String())
			}
		})
	}
}

func TestKeepaliveHandlerConcurrent(t *testing.T) {
	ctx := context.Background()
	cfg := createTestConfig()
	storageConnection := mock.New()

	numRequests := 50
	var wg sync.WaitGroup
	wg.Add(numRequests)

	app := createTestApplication(ctx, cfg, storageConnection, nil)
	server := New(app)

	for i := 0; i < numRequests; i++ {
		go func() {
			defer wg.Done()

			req := httptest.NewRequest(http.MethodPost, "/keepalive", bytes.NewBufferString("123"))
			rr := httptest.NewRecorder()
			server.handleKeepalive(rr, req)

			assert.Equal(t, http.StatusOK, rr.Code)
		}()
	}

	wg.Wait()
}
