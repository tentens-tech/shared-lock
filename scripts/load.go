package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

type LeaseRequest struct {
	Key    string            `json:"key"`
	Value  string            `json:"value"`
	Labels map[string]string `json:"labels"`
}

type LeaseResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

var (
	url         = flag.String("url", "http://localhost:8080", "Base URL of the application")
	endpoint    = flag.String("endpoint", "/lease", "API endpoint to test")
	method      = flag.String("method", "POST", "HTTP method to use")
	concurrency = flag.Int("concurrency", 150, "Number of concurrent clients")
	duration    = flag.Duration("duration", 30*time.Second, "Duration of the load test")
	requestBody = flag.String("body", `{"key":"test-key","value":"test-value","labels":{"test":"load-test"}}`, "Request body (JSON)")
	verbose     = flag.Bool("verbose", false, "Enable verbose output")
)

func main() {
	flag.Parse()

	var body LeaseRequest
	if err := json.Unmarshal([]byte(*requestBody), &body); err != nil {
		log.Fatalf("Failed to parse request body: %v", err)
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	var (
		totalRequests   int64
		successRequests int64
		failedRequests  int64
		totalLatency    int64
	)

	startTime := time.Now()

	var wg sync.WaitGroup

	stop := make(chan struct{})

	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				elapsed := time.Since(startTime).Seconds()
				currentTotal := atomic.LoadInt64(&totalRequests)
				currentSuccess := atomic.LoadInt64(&successRequests)
				currentFailed := atomic.LoadInt64(&failedRequests)
				currentLatency := atomic.LoadInt64(&totalLatency)

				var avgLatency float64
				if currentTotal > 0 {
					avgLatency = float64(currentLatency) / float64(currentTotal) / float64(time.Millisecond)
				}

				rps := float64(currentTotal) / elapsed

				fmt.Printf("\rRequests: %d, Success: %d, Failed: %d, RPS: %.2f, Avg Latency: %.2f ms",
					currentTotal, currentSuccess, currentFailed, rps, avgLatency)
			}
		}
	}()

	go func() {
		time.Sleep(*duration)
		close(stop)
	}()

	for i := 0; i < *concurrency; i++ {
		wg.Add(1)
		go func(clientID int) {
			defer wg.Done()

			for {
				select {
				case <-stop:
					return
				default:
					uniqueBody := body
					uniqueBody.Key = fmt.Sprintf("%s-%d-%d", body.Key, clientID, atomic.LoadInt64(&totalRequests))

					jsonBody, err := json.Marshal(uniqueBody)
					if err != nil {
						log.Printf("Failed to marshal request body: %v", err)
						atomic.AddInt64(&failedRequests, 1)
						continue
					}

					req, err := http.NewRequest(*method, *url+*endpoint, bytes.NewBuffer(jsonBody))
					if err != nil {
						log.Printf("Failed to create request: %v", err)
						atomic.AddInt64(&failedRequests, 1)
						continue
					}

					req.Header.Set("Content-Type", "application/json")

					start := time.Now()
					resp, err := client.Do(req)
					latency := time.Since(start).Milliseconds()
					atomic.AddInt64(&totalLatency, latency)

					atomic.AddInt64(&totalRequests, 1)

					if err != nil {
						if *verbose {
							log.Printf("Request failed: %v", err)
						}
						atomic.AddInt64(&failedRequests, 1)
						continue
					}

					respBody, err := ioutil.ReadAll(resp.Body)
					resp.Body.Close()

					if err != nil {
						if *verbose {
							log.Printf("Failed to read response body: %v", err)
						}
						atomic.AddInt64(&failedRequests, 1)
						continue
					}

					if resp.StatusCode >= 200 && resp.StatusCode < 300 {
						atomic.AddInt64(&successRequests, 1)
						if *verbose {
							log.Printf("Request successful: %s", string(respBody))
						}
					} else {
						if *verbose {
							log.Printf("Request failed with status %d: %s", resp.StatusCode, string(respBody))
						}
						atomic.AddInt64(&failedRequests, 1)
					}
				}
			}
		}(i)
	}

	wg.Wait()

	elapsed := time.Since(startTime).Seconds()
	fmt.Printf("\n\nLoad test completed in %.2f seconds\n", elapsed)
	fmt.Printf("Total requests: %d\n", atomic.LoadInt64(&totalRequests))
	fmt.Printf("Successful requests: %d\n", atomic.LoadInt64(&successRequests))
	fmt.Printf("Failed requests: %d\n", atomic.LoadInt64(&failedRequests))
	fmt.Printf("Requests per second: %.2f\n", float64(atomic.LoadInt64(&totalRequests))/elapsed)

	if atomic.LoadInt64(&totalRequests) > 0 {
		fmt.Printf("Average latency: %.2f ms\n", float64(atomic.LoadInt64(&totalLatency))/float64(atomic.LoadInt64(&totalRequests))/float64(time.Millisecond))
	}
}
