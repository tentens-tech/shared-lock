package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

const (
	LeaseOperationProlong = "prolong"
	LeaseOperationGet     = "get"
)

var (
	LeaseOperations = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "shared_lock_lease_operations_total",
			Help: "Total number of lease operations",
		},
		[]string{"operation", "status"},
	)

	LeaseOperationDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "shared_lock_lease_grant_duration_seconds",
			Help:    "Duration of lease grant in seconds",
			Buckets: prometheus.ExponentialBuckets(1, 2, 10),
		},
		[]string{"operation"},
	)

	CacheOperations = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "shared_lock_cache_operations_total",
			Help: "Total number of cache operations",
		},
		[]string{"operation", "status"},
	)
)

func init() {
	prometheus.MustRegister(LeaseOperations)
	prometheus.MustRegister(LeaseOperationDuration)
	prometheus.MustRegister(CacheOperations)
}
