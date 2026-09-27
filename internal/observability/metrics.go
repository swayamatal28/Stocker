package observability

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

var durationBuckets = []float64{0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 0.75, 1, 2.5, 5}

type requestMetric struct {
	Count   uint64
	Sum     float64
	Buckets []uint64
}

type Registry struct {
	mu       sync.RWMutex
	requests map[string]*requestMetric
}

func NewRegistry() *Registry { return &Registry{requests: map[string]*requestMetric{}} }

func (registry *Registry) ObserveRequest(method, route string, status int, duration time.Duration) {
	if route == "" {
		route = "unmatched"
	}
	key := fmt.Sprintf("%s\x00%s\x00%d", method, route, status)
	registry.mu.Lock()
	defer registry.mu.Unlock()
	metric := registry.requests[key]
	if metric == nil {
		metric = &requestMetric{Buckets: make([]uint64, len(durationBuckets))}
		registry.requests[key] = metric
	}
	seconds := duration.Seconds()
	metric.Count++
	metric.Sum += seconds
	for index, bucket := range durationBuckets {
		if seconds <= bucket {
			metric.Buckets[index]++
		}
	}
}

func (registry *Registry) Prometheus() string {
	registry.mu.RLock()
	defer registry.mu.RUnlock()
	keys := make([]string, 0, len(registry.requests))
	for key := range registry.requests {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var output strings.Builder
	output.WriteString("# HELP stocker_http_requests_total HTTP requests by method, route, and status.\n# TYPE stocker_http_requests_total counter\n")
	for _, key := range keys {
		parts := strings.Split(key, "\x00")
		metric := registry.requests[key]
		fmt.Fprintf(&output, "stocker_http_requests_total{method=%q,route=%q,status=%q} %d\n", parts[0], parts[1], parts[2], metric.Count)
	}
	output.WriteString("# HELP stocker_http_request_duration_seconds Request duration histogram.\n# TYPE stocker_http_request_duration_seconds histogram\n")
	for _, key := range keys {
		parts := strings.Split(key, "\x00")
		metric := registry.requests[key]
		labels := fmt.Sprintf("method=%q,route=%q,status=%q", parts[0], parts[1], parts[2])
		for index, bucket := range durationBuckets {
			fmt.Fprintf(&output, "stocker_http_request_duration_seconds_bucket{%s,le=%q} %d\n", labels, fmt.Sprintf("%g", bucket), metric.Buckets[index])
		}
		fmt.Fprintf(&output, "stocker_http_request_duration_seconds_bucket{%s,le=\"+Inf\"} %d\n", labels, metric.Count)
		fmt.Fprintf(&output, "stocker_http_request_duration_seconds_sum{%s} %.6f\n", labels, metric.Sum)
		fmt.Fprintf(&output, "stocker_http_request_duration_seconds_count{%s} %d\n", labels, metric.Count)
	}
	return output.String()
}
