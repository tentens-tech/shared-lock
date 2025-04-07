package cache

import (
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
				value := fmt.Sprintf("value-%d", requestCount)

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

	// Track memory stats
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	initialAlloc := m.Alloc

	// Fill cache to 80% capacity
	numItems := int(float64(cacheSize) * 0.8)
	for i := 0; i < numItems; i++ {
		key := fmt.Sprintf("key-%d", i)
		value := fmt.Sprintf("value-%d", i)
		c.Set(key, value, time.Hour)
	}

	// Get memory stats after filling
	runtime.GC()
	runtime.ReadMemStats(&m)
	filledAlloc := m.Alloc - initialAlloc

	// Clear cache
	c.SetMaxSize(0)

	// Get memory stats after clearing
	runtime.GC()
	runtime.ReadMemStats(&m)
	clearedAlloc := m.Alloc - initialAlloc

	b.Logf("Memory after filling cache: %v bytes (%.2f MB)", filledAlloc, float64(filledAlloc)/1024/1024)
	b.Logf("Memory after clearing cache: %v bytes (%.2f MB)", clearedAlloc, float64(clearedAlloc)/1024/1024)
	b.Logf("Memory difference: %v bytes (%.2f MB)", filledAlloc-clearedAlloc, float64(filledAlloc-clearedAlloc)/1024/1024)

	assert.Less(b, float64(filledAlloc), float64(50*1024*1024), "Memory usage with 80% filled cache should be less than 50MB")
	assert.Less(b, float64(filledAlloc-clearedAlloc), float64(10*1024*1024), "Memory difference after clearing should be less than 10MB")
}
