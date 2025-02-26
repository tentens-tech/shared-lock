package leaseManagement

import (
	"time"
)

type Lease struct {
	Key     string            `json:"key"`
	Value   string            `json:"value"`
	Labels  map[string]string `json:"labels"`
	Created time.Time         `json:"timestamp"`
}
