package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	baseURL           = "http://localhost:8080"
	leaseEndpoint     = "/lease"
	keepaliveEndpoint = "/keepalive"
	leaseTTL          = "3s"            // Base time to live for a lease
	retryInterval     = 2 * time.Second // Time to wait before retrying to obtain a lease
	keepaliveInterval = 2 * time.Second // Time interval to send keepalive requests
)

type Lease struct {
	Key       string            `json:"key"`
	Value     string            `json:"value"`
	Labels    map[string]string `json:"labels"`
	CreatedAt time.Time         `json:"timestamp"`
}

func main() {
	lease := Lease{
		Key:       "example-key",
		Value:     "example-value",
		Labels:    map[string]string{"env": "production", "app": "go-trainer"},
		CreatedAt: time.Now(),
	}

	for {
		leaseID, err := obtainLease(lease)
		if err == nil {
			fmt.Println("Lease obtained successfully, starting application...")
			startApplication(leaseID)
			break
		} else {
			fmt.Printf("Failed to obtain lease: %v. Retrying in %v...\n", err, retryInterval)
			time.Sleep(retryInterval)
		}
	}
}

func obtainLease(lease Lease) (string, error) {
	leaseData, err := json.Marshal(lease)
	if err != nil {
		return "", fmt.Errorf("failed to marshal lease: %v", err)
	}

	req, err := http.NewRequest("POST", baseURL+leaseEndpoint, bytes.NewBuffer(leaseData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-lease-ttl", leaseTTL)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to create lease: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusAccepted {
		return "", fmt.Errorf("lease already exists")
	}

	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("unexpected response status: %v, body: %v", resp.Status, resp.Body)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %v", err)
	}
	leaseID := string(bodyBytes)
	fmt.Printf("Lease created successfully with ID: %v\n", leaseID)

	return leaseID, nil
}

func startApplication(leaseID string) {
	ticker := time.NewTicker(keepaliveInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			err := sendKeepalive(leaseID)
			if err != nil {
				fmt.Printf("Failed to send keepalive: %v\n", err)
				if err.Error() == "lease is expired" {
					fmt.Println("Lease is expired, stopping application...")
					return
				}
				continue
			}

			fmt.Println("Keepalive sent successfully")
		}
	}
}

func sendKeepalive(leaseID string) error {
	keepaliveData := []byte(leaseID)

	req, err := http.NewRequest("POST", baseURL+keepaliveEndpoint, bytes.NewBuffer(keepaliveData))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send keepalive: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		return fmt.Errorf("lease is expired")
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected response status: %v, body: %v", resp.Status, resp.Body)
	}

	return nil
}
