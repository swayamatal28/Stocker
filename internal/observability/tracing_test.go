package observability

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestTracingExportsOTLPHTTPSpan(t *testing.T) {
	var requests atomic.Int32
	collector := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		body, _ := io.ReadAll(request.Body)
		if request.URL.Path != "/v1/traces" || len(body) == 0 {
			t.Errorf("unexpected OTLP request path=%s bytes=%d", request.URL.Path, len(body))
		}
		requests.Add(1)
		writer.Header().Set("Content-Type", "application/x-protobuf")
		writer.WriteHeader(http.StatusOK)
	}))
	defer collector.Close()

	tracing, err := NewTracing(context.Background(), collector.URL)
	if err != nil {
		t.Fatal(err)
	}
	ctx, span := tracing.StartHTTP(context.Background(), http.Header{}, http.MethodGet, "/health/live")
	if ctx == nil {
		t.Fatal("span did not return a context")
	}
	traceID, spanID := span.IDs()
	if len(traceID) != 32 || len(spanID) != 16 {
		t.Fatalf("invalid trace identifiers: %q %q", traceID, spanID)
	}
	span.Finish(http.MethodGet, "/health/live", http.StatusOK)
	shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := tracing.Shutdown(shutdown); err != nil {
		t.Fatal(err)
	}
	if requests.Load() != 1 {
		t.Fatalf("expected one OTLP export, got %d", requests.Load())
	}
}
