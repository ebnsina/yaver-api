// Package otel wires OpenTelemetry tracing. Exporting is opt-in: it activates
// only when an OTLP endpoint is configured (OTEL_EXPORTER_OTLP_ENDPOINT or
// _TRACES_ENDPOINT), so local dev and tests run with a no-op tracer and no
// network calls. All other knobs use the standard OTEL_* env vars.
package otel

import (
	"context"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// Setup installs a global tracer provider. It returns a shutdown func that
// flushes pending spans; the func is always non-nil (a no-op when disabled).
func Setup(ctx context.Context, serviceName string) (func(context.Context) error, error) {
	noop := func(context.Context) error { return nil }
	if os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") == "" && os.Getenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT") == "" {
		return noop, nil // exporting disabled — keep the default no-op tracer
	}

	exp, err := otlptracehttp.New(ctx) // endpoint/headers from OTEL_* env
	if err != nil {
		return noop, err
	}
	res, err := resource.Merge(resource.Default(), resource.NewSchemaless(
		semconv.ServiceName(serviceName),
	))
	if err != nil {
		return noop, err
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)
	return tp.Shutdown, nil
}
