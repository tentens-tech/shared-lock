package leasemanagement

import (
	"time"
)

type Lease struct {
	Key       string            `json:"key"`
	Value     string            `json:"value"`
	Labels    map[string]string `json:"labels"`
	CreatedAt time.Time         `json:"timestamp"`
}
