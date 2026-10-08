package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRequestIDMiddlewareUsesValidIncomingRequestID(t *testing.T) {
	t.Parallel()

	var observedRequestID string
	handler := requestIDMiddleware(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		observedRequestID, _ = RequestIDFromContext(r.Context())
	}))

	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/health", nil)
	request.Header.Set(RequestIDHeader, "client-request-1")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if observedRequestID != "client-request-1" {
		t.Fatalf("expected incoming request ID in context, got %q", observedRequestID)
	}

	if recorder.Header().Get(RequestIDHeader) != "client-request-1" {
		t.Fatalf("expected response request ID header, got %q", recorder.Header().Get(RequestIDHeader))
	}
}

func TestRequestIDMiddlewareGeneratesRequestIDWhenMissingOrInvalid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		requestID string
	}{
		{
			name:      "missing",
			requestID: "",
		},
		{
			name:      "invalid characters",
			requestID: "bad\r\nid",
		},
		{
			name:      "too long",
			requestID: strings.Repeat("a", 129),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var observedRequestID string
			handler := requestIDMiddleware(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				observedRequestID, _ = RequestIDFromContext(r.Context())
			}))

			request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/health", nil)
			if tt.requestID != "" {
				request.Header.Set(RequestIDHeader, tt.requestID)
			}
			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, request)

			if observedRequestID == "" {
				t.Fatal("expected generated request ID in context")
			}

			if !isValidRequestID(observedRequestID) {
				t.Fatalf("expected generated request ID to be valid, got %q", observedRequestID)
			}

			if recorder.Header().Get(RequestIDHeader) != observedRequestID {
				t.Fatalf("expected response header to match generated request ID")
			}
		})
	}
}

func TestAccessLogMiddlewareWritesStructuredRequestLog(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&output, &slog.HandlerOptions{Level: slog.LevelInfo}))

	handler := requestIDMiddleware(accessLogMiddleware(log, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("created"))
	})))

	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/items/018fb3b2-0f9d-4f59-8a63-7ef4bb812345?token=secret", nil)
	request.RemoteAddr = "127.0.0.1:12345"
	request.Header.Set(RequestIDHeader, "request-123")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	var entry map[string]any
	if err := json.Unmarshal(output.Bytes(), &entry); err != nil {
		t.Fatalf("expected valid JSON log, got error: %v", err)
	}

	if entry["msg"] != "http_request" {
		t.Fatalf("expected http_request message, got %v", entry["msg"])
	}

	if entry["request_id"] != "request-123" {
		t.Fatalf("expected request ID in log, got %v", entry["request_id"])
	}

	if entry["method"] != http.MethodGet {
		t.Fatalf("expected method GET in log, got %v", entry["method"])
	}

	if entry["path"] != "/items/{id}" {
		t.Fatalf("expected normalized path in log, got %v", entry["path"])
	}

	if entry["status"] != float64(http.StatusCreated) {
		t.Fatalf("expected status 201 in log, got %v", entry["status"])
	}

	for _, leaked := range []string{"018fb3b2", "secret", "token"} {
		if strings.Contains(output.String(), leaked) {
			t.Fatalf("expected access log to avoid sensitive path and query values, got %q", output.String())
		}
	}
}
