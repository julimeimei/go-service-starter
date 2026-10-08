package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"
)

type ReadinessCheck func(context.Context) error

type Middleware func(http.Handler) http.Handler

type RouterConfig struct {
	ServiceName       string
	MaxBodyBytes      int64
	Logger            *slog.Logger
	ReadinessCheck    ReadinessCheck
	ItemsHandler      http.Handler
	MetricsHandler    http.Handler
	MetricsMiddleware Middleware
	TracingMiddleware Middleware
}

type healthResponse struct {
	Status  string `json:"status"`
	Service string `json:"service"`
}

type readinessResponse struct {
	Status       string            `json:"status"`
	Service      string            `json:"service"`
	Dependencies map[string]string `json:"dependencies"`
}

type errorResponse struct {
	Error errorDetails `json:"error"`
}

type errorDetails struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func NewRouter(cfg RouterConfig) http.Handler {
	log := cfg.Logger
	if log == nil {
		log = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	router := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/health":
			if r.Method != http.MethodGet {
				w.Header().Set("Allow", http.MethodGet)
				writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
				return
			}

			healthHandler(cfg.ServiceName).ServeHTTP(w, r)
		case r.URL.Path == "/ready":
			if r.Method != http.MethodGet {
				w.Header().Set("Allow", http.MethodGet)
				writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
				return
			}

			readinessHandler(cfg.ServiceName, cfg.ReadinessCheck, log).ServeHTTP(w, r)
		case r.URL.Path == "/metrics":
			if r.Method != http.MethodGet {
				w.Header().Set("Allow", http.MethodGet)
				writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
				return
			}

			if cfg.MetricsHandler == nil {
				writeError(w, http.StatusNotFound, "not_found", "route not found")
				return
			}

			cfg.MetricsHandler.ServeHTTP(w, r)
		case r.URL.Path == "/items" || strings.HasPrefix(r.URL.Path, "/items/"):
			if cfg.ItemsHandler == nil {
				writeError(w, http.StatusServiceUnavailable, "service_unavailable", "service unavailable")
				return
			}

			cfg.ItemsHandler.ServeHTTP(w, r)
		default:
			writeError(w, http.StatusNotFound, "not_found", "route not found")
		}
	})

	handler := accessLogMiddleware(log, limitRequestBody(cfg.MaxBodyBytes, router))
	if cfg.MetricsMiddleware != nil {
		handler = cfg.MetricsMiddleware(handler)
	}
	if cfg.TracingMiddleware != nil {
		handler = cfg.TracingMiddleware(handler)
	}

	return requestIDMiddleware(handler)
}

func healthHandler(serviceName string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		response := healthResponse{
			Status:  "ok",
			Service: serviceName,
		}

		writeJSON(w, http.StatusOK, response)
	}
}

func readinessHandler(serviceName string, check ReadinessCheck, log *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		response := readinessResponse{
			Status:  "ready",
			Service: serviceName,
			Dependencies: map[string]string{
				"database": "ok",
			},
		}

		if check == nil {
			response.Status = "not_ready"
			response.Dependencies["database"] = "not_configured"
			writeJSON(w, http.StatusServiceUnavailable, response)
			return
		}

		if err := check(r.Context()); err != nil {
			log.WarnContext(r.Context(), "readiness check failed", "dependency", "database")
			response.Status = "not_ready"
			response.Dependencies["database"] = "unavailable"
			writeJSON(w, http.StatusServiceUnavailable, response)
			return
		}

		writeJSON(w, http.StatusOK, response)
	}
}

func limitRequestBody(maxBodyBytes int64, next http.Handler) http.Handler {
	if maxBodyBytes <= 0 {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ContentLength > maxBodyBytes {
			writeError(w, http.StatusRequestEntityTooLarge, "request_body_too_large", "request body is too large")
			return
		}

		if r.Body != nil {
			r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
		}

		next.ServeHTTP(w, r)
	})
}

func writeError(w http.ResponseWriter, statusCode int, code, message string) {
	writeJSON(w, statusCode, errorResponse{
		Error: errorDetails{
			Code:    code,
			Message: message,
		},
	})
}

func writeJSON(w http.ResponseWriter, statusCode int, response any) {
	body, err := json.Marshal(response)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":{"code":"internal_server_error","message":"internal server error"}}` + "\n"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if _, err := w.Write(append(body, '\n')); err != nil {
		return
	}
}
