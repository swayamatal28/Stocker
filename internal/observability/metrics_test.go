package observability

import (
	"strings"
	"testing"
	"time"
)

func TestRegistryRendersPrometheusHistogram(t *testing.T) {
	registry := NewRegistry()
	registry.ObserveRequest("GET", "/health/live", 200, 20*time.Millisecond)
	output := registry.Prometheus()
	for _, expected := range []string{"stocker_http_requests_total", `route="/health/live"`, "stocker_http_request_duration_seconds_bucket", `le="0.025"} 1`} {
		if !strings.Contains(output, expected) {
			t.Fatalf("metric output missing %q:\n%s", expected, output)
		}
	}
}
