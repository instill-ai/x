package otel

import (
	"context"
	"fmt"
	"io"

	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutlog"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"

	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
)

// SetupLogging initializes the OpenTelemetry logging system
func SetupLogging(ctx context.Context, options ...Option) (*log.LoggerProvider, error) {
	opts := newSetupOptions(options...)

	var exporter log.Exporter
	var err error

	if opts.CollectorEnable {
		// Use OTLP exporter for external logging
		exporter, err = otlploggrpc.New(
			ctx,
			otlploggrpc.WithEndpoint(fmt.Sprintf("%s:%d", opts.Host, opts.Port)),
			otlploggrpc.WithInsecure(),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create OTLP log exporter: %w", err)
		}
	} else {
		// Use stdout exporter for local development
		// Discard the logs to avoid cluttering the console with the default zap logger
		exporter, err = stdoutlog.New(
			stdoutlog.WithWriter(io.Discard),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create stdout log exporter: %w", err)
		}
	}

	// Create resource with service information
	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String(opts.ServiceName),
		semconv.ServiceVersionKey.String(opts.ServiceVersion),
	)

	// Create logger provider
	provider := log.NewLoggerProvider(
		log.WithProcessor(log.NewBatchProcessor(exporter)),
		log.WithResource(res),
	)

	// Set the logger provider globally
	global.SetLoggerProvider(provider)

	return provider, nil
}
