package tracing

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

type Options struct {
	Enabled     bool
	ServiceName string
	Environment string
	Exporter    string
	SampleRatio float64
}

type HTTPMiddlewareOptions struct {
	ServiceName  string
	RoutePattern func(string) string
}

func Setup(ctx context.Context, options Options) (func(context.Context) error, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("tracing setup canceled: %w", ctx.Err())
	default:
	}

	if !options.Enabled {
		return func(context.Context) error { return nil }, nil
	}

	if options.ServiceName == "" {
		return nil, fmt.Errorf("tracing service name is required")
	}

	if options.SampleRatio < 0 || options.SampleRatio > 1 {
		return nil, fmt.Errorf("tracing sample ratio must be between 0 and 1")
	}

	exporter, err := newExporter(options.Exporter)
	if err != nil {
		return nil, err
	}

	provider := tracesdk.NewTracerProvider(
		tracesdk.WithBatcher(exporter),
		tracesdk.WithResource(resource.NewWithAttributes(
			"",
			attribute.String("service.name", options.ServiceName),
			attribute.String("deployment.environment", options.Environment),
		)),
		tracesdk.WithSampler(tracesdk.ParentBased(tracesdk.TraceIDRatioBased(options.SampleRatio))),
	)

	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return provider.Shutdown, nil
}

func HTTPMiddleware(options HTTPMiddlewareOptions) func(http.Handler) http.Handler {
	serviceName := options.ServiceName
	if serviceName == "" {
		serviceName = "go-service-starter"
	}

	routePattern := options.RoutePattern
	if routePattern == nil {
		routePattern = func(string) string {
			return "unknown"
		}
	}

	tracer := otel.Tracer(serviceName)
	propagator := otel.GetTextMapPropagator()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			route := routePattern(r.URL.Path)
			ctx := propagator.Extract(r.Context(), propagation.HeaderCarrier(r.Header))
			ctx, span := tracer.Start(
				ctx,
				"HTTP "+r.Method+" "+route,
				trace.WithSpanKind(trace.SpanKindServer),
				trace.WithAttributes(
					attribute.String("http.request.method", r.Method),
					attribute.String("http.route", route),
				),
			)
			defer span.End()

			recorder := &statusRecorder{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			next.ServeHTTP(recorder, r.WithContext(ctx))

			span.SetAttributes(
				attribute.Int("http.response.status_code", recorder.statusCode),
			)
			if recorder.statusCode >= http.StatusInternalServerError {
				span.SetStatus(codes.Error, strconv.Itoa(recorder.statusCode))
			}
		})
	}
}

func newExporter(exporterName string) (tracesdk.SpanExporter, error) {
	switch exporterName {
	case "stdout":
		exporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
		if err != nil {
			return nil, fmt.Errorf("create stdout trace exporter: %w", err)
		}

		return exporter, nil
	default:
		return nil, fmt.Errorf("TRACING_EXPORTER must be one of: stdout")
	}
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

func ShutdownContext(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if parent == nil {
		parent = context.Background()
	}

	return context.WithTimeout(parent, timeout)
}
