package otel

import (
	"context"
	"fmt"
	"io"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"

	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
)

// SetupTracing initializes the OpenTelemetry tracing system
func SetupTracing(ctx context.Context, options ...Option) (*trace.TracerProvider, error) {
	opts := newSetupOptions(options...)

	var exporter trace.SpanExporter
	var err error
	if opts.CollectorEnable {
		exporter, err = otlptracegrpc.New(
			ctx,
			otlptracegrpc.WithEndpoint(fmt.Sprintf("%s:%d", opts.Host, opts.Port)),
			otlptracegrpc.WithInsecure(),
		)
		if err != nil {
			return nil, err
		}
	} else {
		exporter, err = stdouttrace.New(
			stdouttrace.WithWriter(io.Discard),
		)
		if err != nil {
			return nil, err
		}
	}

	// labels/tags/resources that are common to all traces.
	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String(opts.ServiceName),
		semconv.ServiceVersionKey.String(opts.ServiceVersion),
	)

	provider := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(res),
	)

	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	return provider, nil
}
