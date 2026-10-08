package tracing

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/julimeimei/go-service-starter/internal/platform/observability"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestHTTPMiddlewareCreatesServerSpanWithSafeAttributes(t *testing.T) {
	exporter := configureTestTracer(t)

	handler := HTTPMiddleware(HTTPMiddlewareOptions{
		ServiceName:  "go-service-starter",
		RoutePattern: observability.RoutePattern,
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if spanContext := trace.SpanContextFromContext(r.Context()); !spanContext.IsValid() {
			t.Fatal("expected handler context to include a valid span context")
		}

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

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}

	span := spans[0]
	if span.Name != "HTTP GET /items/{id}" {
		t.Fatalf("expected normalized span name, got %q", span.Name)
	}

	if span.SpanKind != trace.SpanKindServer {
		t.Fatalf("expected server span kind, got %s", span.SpanKind)
	}

	attributes := attributeMap(span.Attributes)
	if attributes["http.request.method"] != "GET" {
		t.Fatalf("expected method attribute GET, got %q", attributes["http.request.method"])
	}

	if attributes["http.route"] != "/items/{id}" {
		t.Fatalf("expected normalized route attribute, got %q", attributes["http.route"])
	}

	if attributes["http.response.status_code"] != "201" {
		t.Fatalf("expected status code attribute 201, got %q", attributes["http.response.status_code"])
	}

	spanText := fmt.Sprint(span)
	for _, leaked := range []string{"018fb3b2", "secret-token", "token"} {
		if strings.Contains(spanText, leaked) {
			t.Fatalf("expected span not to leak %q, got:\n%s", leaked, spanText)
		}
	}
}

func TestHTTPMiddlewareExtractsTraceParent(t *testing.T) {
	exporter := configureTestTracer(t)
	var handlerSpanContext trace.SpanContext

	handler := HTTPMiddleware(HTTPMiddlewareOptions{
		ServiceName:  "go-service-starter",
		RoutePattern: observability.RoutePattern,
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerSpanContext = trace.SpanContextFromContext(r.Context())
		w.WriteHeader(http.StatusNoContent)
	}))

	const traceID = "4bf92f3577b34da6a3ce929d0e0e4736"
	const parentSpanID = "00f067aa0ba902b7"

	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/health", nil)
	request.Header.Set("traceparent", "00-"+traceID+"-"+parentSpanID+"-01")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if got := handlerSpanContext.TraceID().String(); got != traceID {
		t.Fatalf("expected handler trace id %q, got %q", traceID, got)
	}

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}

	if got := spans[0].Parent.TraceID().String(); got != traceID {
		t.Fatalf("expected parent trace id %q, got %q", traceID, got)
	}

	if got := spans[0].Parent.SpanID().String(); got != parentSpanID {
		t.Fatalf("expected parent span id %q, got %q", parentSpanID, got)
	}
}

func TestSetupReturnsNoopShutdownWhenDisabled(t *testing.T) {
	t.Parallel()

	shutdown, err := Setup(context.Background(), Options{
		Enabled: false,
	})
	if err != nil {
		t.Fatalf("expected disabled tracing setup to succeed, got error: %v", err)
	}

	if err := shutdown(context.Background()); err != nil {
		t.Fatalf("expected disabled tracing shutdown to succeed, got error: %v", err)
	}
}

func TestSetupRejectsInvalidExporterWhenEnabled(t *testing.T) {
	t.Parallel()

	_, err := Setup(context.Background(), Options{
		Enabled:     true,
		ServiceName: "go-service-starter",
		Environment: "test",
		Exporter:    "unknown",
		SampleRatio: 1,
	})
	if err == nil {
		t.Fatal("expected invalid exporter error, got nil")
	}

	if !strings.Contains(err.Error(), "TRACING_EXPORTER") {
		t.Fatalf("expected TRACING_EXPORTER error, got %q", err.Error())
	}
}

func configureTestTracer(t *testing.T) *tracetest.InMemoryExporter {
	t.Helper()

	exporter := tracetest.NewInMemoryExporter()
	provider := tracesdk.NewTracerProvider(
		tracesdk.WithSyncer(exporter),
		tracesdk.WithSampler(tracesdk.AlwaysSample()),
	)

	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	t.Cleanup(func() {
		_ = provider.Shutdown(context.Background())
		otel.SetTracerProvider(noop.NewTracerProvider())
		otel.SetTextMapPropagator(propagation.TraceContext{})
	})

	return exporter
}

func attributeMap(attributes []attribute.KeyValue) map[string]string {
	result := make(map[string]string, len(attributes))
	for _, attr := range attributes {
		switch attr.Value.Type() {
		case attribute.STRING:
			result[string(attr.Key)] = attr.Value.AsString()
		case attribute.INT64:
			result[string(attr.Key)] = strconv.FormatInt(attr.Value.AsInt64(), 10)
		default:
			result[string(attr.Key)] = attr.Value.String()
		}
	}

	return result
}
