package metrics

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func TestHTTPMetricsRecordsRequestsWithStableRouteLabels(t *testing.T) {
	t.Parallel()

	registry := prometheus.NewRegistry()
	httpMetrics, err := NewHTTPMetrics(registry)
	if err != nil {
		t.Fatalf("failed to create HTTP metrics: %v", err)
	}

	handler := httpMetrics.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))

	request := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		"/items/018fb3b2-0f9d-4f59-8a63-7ef4bb812345?token=secret-token",
		nil,
	)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	metricsBody := collectMetrics(t, registry)

	expectedCounter := `http_requests_total{method="GET",route="/items/{id}",status="201"} 1`
	if !strings.Contains(metricsBody, expectedCounter) {
		t.Fatalf("expected metrics to contain %q, got:\n%s", expectedCounter, metricsBody)
	}

	if !strings.Contains(metricsBody, `http_request_duration_seconds_bucket{method="GET",route="/items/{id}",status="201"`) {
		t.Fatalf("expected duration histogram for items route, got:\n%s", metricsBody)
	}

	for _, leaked := range []string{"018fb3b2", "secret-token", "token"} {
		if strings.Contains(metricsBody, leaked) {
			t.Fatalf("expected metrics not to leak %q, got:\n%s", leaked, metricsBody)
		}
	}
}

func TestRoutePatternUsesLowCardinalityRoutes(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"/health":  "/health",
		"/ready":   "/ready",
		"/metrics": "/metrics",
		"/items":   "/items",
		"/items/018fb3b2-0f9d-4f59-8a63-7ef4bb812345": "/items/{id}",
		"/unknown/path": "unknown",
	}

	for path, expected := range tests {
		t.Run(path, func(t *testing.T) {
			t.Parallel()

			if got := RoutePattern(path); got != expected {
				t.Fatalf("expected route pattern %q, got %q", expected, got)
			}
		})
	}
}

func TestNewHTTPMetricsRequiresRegistry(t *testing.T) {
	t.Parallel()

	if _, err := NewHTTPMetrics(nil); err == nil {
		t.Fatal("expected error when registry is nil")
	}
}

func collectMetrics(t *testing.T, registry *prometheus.Registry) string {
	t.Helper()

	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/metrics", nil)
	recorder := httptest.NewRecorder()
	Handler(registry).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected metrics status %d, got %d", http.StatusOK, recorder.Code)
	}

	return recorder.Body.String()
}
