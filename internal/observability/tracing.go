package observability

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

type Tracing struct {
	provider *sdktrace.TracerProvider
	enabled  bool
}

type HTTPSpan struct {
	span trace.Span
}

func NewTracing(ctx context.Context, endpoint string) (*Tracing, error) {
	tracing := &Tracing{}
	if strings.TrimSpace(endpoint) == "" {
		return tracing, nil
	}
	if parsed, err := url.Parse(endpoint); err == nil && (parsed.Path == "" || parsed.Path == "/") {
		endpoint = strings.TrimRight(endpoint, "/") + "/v1/traces"
	}
	exporter, err := otlptracehttp.New(ctx, otlptracehttp.WithEndpointURL(endpoint))
	if err != nil {
		return nil, err
	}
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource.NewWithAttributes("", attribute.String("service.name", "stocker-api"))),
	)
	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	tracing.provider = provider
	tracing.enabled = true
	return tracing, nil
}

func (tracing *Tracing) Enabled() bool { return tracing != nil && tracing.enabled }

func (tracing *Tracing) StartHTTP(ctx context.Context, headers http.Header, method, path string) (context.Context, HTTPSpan) {
	if !tracing.Enabled() {
		return ctx, HTTPSpan{}
	}
	ctx = otel.GetTextMapPropagator().Extract(ctx, propagation.HeaderCarrier(headers))
	ctx, span := otel.Tracer("stocker/http").Start(ctx, method+" http.request", trace.WithSpanKind(trace.SpanKindServer), trace.WithAttributes(attribute.String("http.request.method", method), attribute.String("url.path", path)))
	return ctx, HTTPSpan{span: span}
}

func (span HTTPSpan) IDs() (string, string) {
	if span.span == nil {
		return "", ""
	}
	context := span.span.SpanContext()
	return context.TraceID().String(), context.SpanID().String()
}

func (span HTTPSpan) Finish(method, route string, status int) {
	if span.span == nil {
		return
	}
	if route == "" {
		route = "unmatched"
	}
	span.span.SetName(method + " " + route)
	span.span.SetAttributes(attribute.String("http.route", route), attribute.Int("http.response.status_code", status))
	if status >= http.StatusInternalServerError {
		span.span.SetStatus(codes.Error, http.StatusText(status))
	}
	span.span.End()
}

func (tracing *Tracing) Shutdown(ctx context.Context) error {
	if !tracing.Enabled() {
		return nil
	}
	return tracing.provider.Shutdown(ctx)
}
