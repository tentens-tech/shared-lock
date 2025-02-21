package leaseManagement

import (
	"encoding/json"
	"time"
)

type Lease struct {
	Key     string            `json:"key"`
	Value   string            `json:"value"`
	Labels  map[string]string `json:"labels"`
	Created time.Time         `json:"timestamp"`
}

func (l *Lease) ToJSON() ([]byte, error) {
	return json.Marshal(l)
}
