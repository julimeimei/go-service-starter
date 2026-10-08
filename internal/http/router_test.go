package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealthEndpoint(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/health", nil)
	recorder := httptest.NewRecorder()

	NewRouter(RouterConfig{
		ServiceName:  "go-service-starter",
		MaxBodyBytes: 1024,
	}).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	if contentType := recorder.Header().Get("Content-Type"); contentType != "application/json" {
		t.Fatalf("expected Content-Type application/json, got %q", contentType)
	}

	var response healthResponse
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	if response.Status != "ok" {
		t.Fatalf("expected status ok, got %q", response.Status)
	}

	if response.Service != "go-service-starter" {
		t.Fatalf("expected service go-service-starter, got %q", response.Service)
	}
}

func TestHealthEndpointRejectsUnsupportedMethod(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/health", nil)
	recorder := httptest.NewRecorder()

	NewRouter(RouterConfig{
		ServiceName:  "go-service-starter",
		MaxBodyBytes: 1024,
	}).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, recorder.Code)
	}

	var response errorResponse
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	if response.Error.Code != "method_not_allowed" {
		t.Fatalf("expected method_not_allowed error, got %q", response.Error.Code)
	}
}

func TestHealthEndpointDoesNotDependOnReadiness(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/health", nil)
	recorder := httptest.NewRecorder()

	NewRouter(RouterConfig{
		ServiceName:  "go-service-starter",
		MaxBodyBytes: 1024,
		ReadinessCheck: func(context.Context) error {
			return errors.New("dependency unavailable")
		},
	}).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
}

func TestReadinessEndpointReturnsOKWhenDependenciesAreReady(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/ready", nil)
	recorder := httptest.NewRecorder()

	NewRouter(RouterConfig{
		ServiceName:  "go-service-starter",
		MaxBodyBytes: 1024,
		ReadinessCheck: func(context.Context) error {
			return nil
		},
	}).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	var response readinessResponse
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	if response.Status != "ready" {
		t.Fatalf("expected status ready, got %q", response.Status)
	}

	if response.Service != "go-service-starter" {
		t.Fatalf("expected service go-service-starter, got %q", response.Service)
	}

	if response.Dependencies["database"] != "ok" {
		t.Fatalf("expected database dependency ok, got %q", response.Dependencies["database"])
	}
}

func TestReadinessEndpointReturnsUnavailableWhenDatabaseIsNotConfigured(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/ready", nil)
	recorder := httptest.NewRecorder()

	NewRouter(RouterConfig{
		ServiceName:  "go-service-starter",
		MaxBodyBytes: 1024,
	}).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, recorder.Code)
	}

	var response readinessResponse
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	if response.Status != "not_ready" {
		t.Fatalf("expected status not_ready, got %q", response.Status)
	}

	if response.Dependencies["database"] != "not_configured" {
		t.Fatalf("expected database dependency not_configured, got %q", response.Dependencies["database"])
	}
}

func TestReadinessEndpointReturnsUnavailableWhenDependencyFails(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/ready", nil)
	recorder := httptest.NewRecorder()
	var logs bytes.Buffer

	NewRouter(RouterConfig{
		ServiceName:  "go-service-starter",
		MaxBodyBytes: 1024,
		Logger:       slog.New(slog.NewTextHandler(&logs, nil)),
		ReadinessCheck: func(context.Context) error {
			return errors.New("database password is secret")
		},
	}).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, recorder.Code)
	}

	body := recorder.Body.String()
	if strings.Contains(body, "secret") {
		t.Fatalf("expected readiness response not to leak dependency details, got %q", body)
	}

	if strings.Contains(logs.String(), "secret") {
		t.Fatalf("expected readiness log not to leak dependency details, got %q", logs.String())
	}

	var response readinessResponse
	if err := json.NewDecoder(strings.NewReader(body)).Decode(&response); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	if response.Status != "not_ready" {
		t.Fatalf("expected status not_ready, got %q", response.Status)
	}

	if response.Dependencies["database"] != "unavailable" {
		t.Fatalf("expected database dependency unavailable, got %q", response.Dependencies["database"])
	}
}

func TestReadinessEndpointRejectsUnsupportedMethod(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/ready", nil)
	recorder := httptest.NewRecorder()

	NewRouter(RouterConfig{
		ServiceName:  "go-service-starter",
		MaxBodyBytes: 1024,
	}).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, recorder.Code)
	}

	if allow := recorder.Header().Get("Allow"); allow != http.MethodGet {
		t.Fatalf("expected Allow header GET, got %q", allow)
	}

	var response errorResponse
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	if response.Error.Code != "method_not_allowed" {
		t.Fatalf("expected method_not_allowed error, got %q", response.Error.Code)
	}
}

func TestMetricsEndpointDispatchesMetricsHandler(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/metrics", nil)
	recorder := httptest.NewRecorder()

	NewRouter(RouterConfig{
		ServiceName:  "go-service-starter",
		MaxBodyBytes: 1024,
		MetricsHandler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/metrics" {
				t.Fatalf("expected metrics path, got %q", r.URL.Path)
			}

			w.Header().Set("Content-Type", "text/plain; version=0.0.4")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("# HELP test_metric A test metric.\n"))
		}),
	}).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	if body := recorder.Body.String(); !strings.Contains(body, "test_metric") {
		t.Fatalf("expected metrics handler response, got %q", body)
	}
}

func TestMetricsEndpointRejectsUnsupportedMethod(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/metrics", nil)
	recorder := httptest.NewRecorder()

	NewRouter(RouterConfig{
		ServiceName:  "go-service-starter",
		MaxBodyBytes: 1024,
		MetricsHandler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	}).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, recorder.Code)
	}

	if allow := recorder.Header().Get("Allow"); allow != http.MethodGet {
		t.Fatalf("expected Allow header GET, got %q", allow)
	}

	var response errorResponse
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	if response.Error.Code != "method_not_allowed" {
		t.Fatalf("expected method_not_allowed error, got %q", response.Error.Code)
	}
}

func TestRouterDispatchesItemsRequests(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/items", strings.NewReader(`{"name":"Example item"}`))
	recorder := httptest.NewRecorder()

	NewRouter(RouterConfig{
		ServiceName:  "go-service-starter",
		MaxBodyBytes: 1024,
		ItemsHandler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/items" {
				t.Fatalf("expected items path, got %q", r.URL.Path)
			}

			w.WriteHeader(http.StatusCreated)
		}),
	}).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, recorder.Code)
	}
}

func TestRouterReturnsUnavailableWhenItemsHandlerIsNotConfigured(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/items", strings.NewReader(`{"name":"Example item"}`))
	recorder := httptest.NewRecorder()

	NewRouter(RouterConfig{
		ServiceName:  "go-service-starter",
		MaxBodyBytes: 1024,
	}).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, recorder.Code)
	}

	var response errorResponse
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	if response.Error.Code != "service_unavailable" {
		t.Fatalf("expected service_unavailable error, got %q", response.Error.Code)
	}
}

func TestRouterReturnsJSONNotFound(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/missing", nil)
	recorder := httptest.NewRecorder()

	NewRouter(RouterConfig{
		ServiceName:  "go-service-starter",
		MaxBodyBytes: 1024,
	}).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, recorder.Code)
	}

	var response errorResponse
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	if response.Error.Code != "not_found" {
		t.Fatalf("expected not_found error, got %q", response.Error.Code)
	}
}

func TestRouterRejectsRequestsAboveMaxBodySize(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/health", strings.NewReader("too large"))
	request.ContentLength = int64(len("too large"))
	recorder := httptest.NewRecorder()

	NewRouter(RouterConfig{
		ServiceName:  "go-service-starter",
		MaxBodyBytes: 3,
	}).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected status %d, got %d", http.StatusRequestEntityTooLarge, recorder.Code)
	}

	var response errorResponse
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	if response.Error.Code != "request_body_too_large" {
		t.Fatalf("expected request_body_too_large error, got %q", response.Error.Code)
	}
}
