package cache

import (
	"encoding/json"
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const (
	targetRPS    = 1000
	testDuration = 10 * time.Second
	cacheSize    = 10000
)

type LeaseRecord struct {
	Key       string                 `json:"key"`
	Value     string                 `json:"value"`
	Labels    map[string]string      `json:"labels"`
	CreatedAt time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata"`
}

func createComplexLeaseRecord(index int) LeaseRecord {
	return LeaseRecord{
		Key:   fmt.Sprintf("service-run-command-%d", index),
		Value: fmt.Sprintf("This is a large value that contains a lot of data for lease %d", index),
		Labels: map[string]string{
			"command":  "/app/service run-command",
			"hostname": "service-singleton-worker-6b9865cfc6-dt2fv",
			"env":      "production",
			"region":   "us",
			"service":  "service-service",
			"version":  "1.2.3",
			"team":     "platform",
			"owner":    "devops",
			"priority": "high",
			"type":     "scheduled",
			"schedule": "0 0 * * *",
			"timeout":  "300s",
			"retries":  "3",
			"tags":     "automation,maintenance,cleanup",
			"jira":     "PLAT-1234",
			"slack":    "#platform-alerts",
			"email":    "platform@company.com",
			"status":   "active",
		},
		CreatedAt: time.Now().UTC(),
		Metadata: map[string]interface{}{
			"created": "2023-01-01T00:00:00Z",
			"updated": "2023-01-02T00:00:00Z",
			"expires": "2023-12-31T23:59:59Z",
			"nested": map[string]interface{}{
				"field1": "value1",
				"field2": 123,
				"field3": true,
				"field4": []string{"item1", "item2", "item3"},
			},
			"metrics": map[string]interface{}{
				"cpu":    0.75,
				"memory": 1024,
				"disk":   5120,
			},
		},
	}
}

func BenchmarkCacheUnderLoad(b *testing.B) {
	c := New(cacheSize)

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	initialAlloc := m.Alloc
	initialSys := m.Sys

	done := make(chan struct{})
	var wg sync.WaitGroup

	startTime := time.Now()
	var requestCount int64

	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(time.Second / time.Duration(targetRPS))
		defer ticker.Stop()

		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				key := fmt.Sprintf("key-%d", requestCount)
				value := createComplexLeaseRecord(int(requestCount))

				c.Set(key, value, time.Minute)

				_, exists := c.Get(key)
				assert.True(b, exists)

				requestCount++
			}
		}
	}()

	time.Sleep(testDuration)
	close(done)
	wg.Wait()

	duration := time.Since(startTime)
	actualRPS := float64(requestCount) / duration.Seconds()

	runtime.GC()
	runtime.ReadMemStats(&m)

	memoryAlloc := m.Alloc - initialAlloc
	memorySys := m.Sys - initialSys

	b.Logf("Test Duration: %v", duration)
	b.Logf("Total Requests: %d", requestCount)
	b.Logf("Actual RPS: %.2f", actualRPS)
	b.Logf("Memory Allocated: %v bytes (%.2f MB)", memoryAlloc, float64(memoryAlloc)/1024/1024)
	b.Logf("System Memory: %v bytes (%.2f MB)", memorySys, float64(memorySys)/1024/1024)
	b.Logf("Current Cache Size: %d items", len(c.items))

	assert.GreaterOrEqual(b, actualRPS, float64(targetRPS-100), "RPS should be at least 900")
	assert.LessOrEqual(b, actualRPS, float64(targetRPS+100), "RPS should not exceed 1100")
	assert.Less(b, float64(memoryAlloc), float64(100*1024*1024), "Memory allocation should be less than 100MB")
}

func BenchmarkCacheMemoryGrowth(b *testing.B) {
	cacheSize := 10000
	c := New(cacheSize)

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	initialAlloc := m.Alloc

	numItems := int(float64(cacheSize) * 0.8)
	for i := 0; i < numItems; i++ {
		key := fmt.Sprintf("key-%d", i)
		value := createComplexLeaseRecord(i)
		c.Set(key, value, time.Hour)
	}

	runtime.GC()
	runtime.ReadMemStats(&m)
	filledAlloc := m.Alloc - initialAlloc

	c.SetMaxSize(0)

	runtime.GC()
	runtime.ReadMemStats(&m)
	clearedAlloc := m.Alloc - initialAlloc

	b.Logf("Memory after filling cache: %v bytes (%.2f MB)", filledAlloc, float64(filledAlloc)/1024/1024)
	b.Logf("Memory after clearing cache: %v bytes (%.2f MB)", clearedAlloc, float64(clearedAlloc)/1024/1024)
	b.Logf("Memory difference: %v bytes (%.2f MB)", filledAlloc-clearedAlloc, float64(filledAlloc-clearedAlloc)/1024/1024)

	assert.Less(b, float64(filledAlloc), float64(50*1024*1024), "Memory usage with 80% filled cache should be less than 50MB")
	assert.Less(b, float64(filledAlloc-clearedAlloc), float64(25*1024*1024), "Memory difference after clearing should be less than 25MB")
}

func BenchmarkCacheWithJSON(b *testing.B) {
	c := New(cacheSize)

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	initialAlloc := m.Alloc

	lease := createComplexLeaseRecord(1)

	jsonData, err := json.Marshal(lease)
	assert.NoError(b, err)

	c.Set("json-lease", jsonData, time.Hour)

	var retrievedData []byte
	var exists bool
	var value interface{}
	value, exists = c.Get("json-lease")
	retrievedData = value.([]byte)
	assert.True(b, exists)

	var retrievedLease LeaseRecord
	err = json.Unmarshal(retrievedData, &retrievedLease)
	assert.NoError(b, err)

	assert.Equal(b, lease.Key, retrievedLease.Key)
	assert.Equal(b, lease.Value, retrievedLease.Value)
	assert.Equal(b, len(lease.Labels), len(retrievedLease.Labels))

	runtime.GC()
	runtime.ReadMemStats(&m)

	memoryAlloc := m.Alloc - initialAlloc
	b.Logf("Memory allocated for JSON serialization/deserialization: %v bytes (%.2f MB)",
		memoryAlloc, float64(memoryAlloc)/1024/1024)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		jsonData, err = json.Marshal(lease)
		assert.NoError(b, err)
		c.Set(fmt.Sprintf("json-lease-%d", i), jsonData, time.Hour)

		value, _ = c.Get(fmt.Sprintf("json-lease-%d", i))
		retrievedData = value.([]byte)

		err = json.Unmarshal(retrievedData, &retrievedLease)
		assert.NoError(b, err)
	}
}
