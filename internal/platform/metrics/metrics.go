package metrics

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/julimeimei/go-service-starter/internal/platform/observability"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type HTTPMetrics struct {
	requestsTotal   *prometheus.CounterVec
	requestDuration *prometheus.HistogramVec
}

func NewRegistry() *prometheus.Registry {
	registry := prometheus.NewRegistry()
	registry.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)

	return registry
}

func NewHTTPMetrics(registry *prometheus.Registry) (*HTTPMetrics, error) {
	if registry == nil {
		return nil, fmt.Errorf("metrics registry is required")
	}

	httpMetrics := &HTTPMetrics{
		requestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "Total number of HTTP requests processed by method, route, and status.",
			},
			[]string{"method", "route", "status"},
		),
		requestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_duration_seconds",
				Help:    "HTTP request duration in seconds by method, route, and status.",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "route", "status"},
		),
	}

	if err := registry.Register(httpMetrics.requestsTotal); err != nil {
		return nil, fmt.Errorf("register http_requests_total: %w", err)
	}

	if err := registry.Register(httpMetrics.requestDuration); err != nil {
		return nil, fmt.Errorf("register http_request_duration_seconds: %w", err)
	}

	return httpMetrics, nil
}

func Handler(registry *prometheus.Registry) http.Handler {
	return promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
}

func (m *HTTPMetrics) Middleware(next http.Handler) http.Handler {
	if m == nil {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startedAt := time.Now()
		recorder := &statusRecorder{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		next.ServeHTTP(recorder, r)

		status := strconv.Itoa(recorder.statusCode)
		route := observability.RoutePattern(r.URL.Path)

		m.requestsTotal.WithLabelValues(r.Method, route, status).Inc()
		m.requestDuration.WithLabelValues(r.Method, route, status).Observe(time.Since(startedAt).Seconds())
	})
}

func RoutePattern(path string) string {
	return observability.RoutePattern(path)
}

type statusRecorder struct {
	http.ResponseWriter
	statusCode  int
	wroteHeader bool
}

func (r *statusRecorder) WriteHeader(statusCode int) {
	if r.wroteHeader {
		return
	}

	r.statusCode = statusCode
	r.wroteHeader = true
	r.ResponseWriter.WriteHeader(statusCode)
}

func (r *statusRecorder) Write(body []byte) (int, error) {
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}

	return r.ResponseWriter.Write(body)
}
