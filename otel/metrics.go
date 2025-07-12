package otel

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"

	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
)

// SetupMetrics sets up the metrics provider and returns it.
func SetupMetrics(ctx context.Context, options ...Option) (*metric.MeterProvider, error) {
	opts := newSetupOptions(options...)

	var exporter metric.Exporter
	var err error
	if opts.CollectorEnable {
		exporter, err = otlpmetricgrpc.New(
			ctx,
			otlpmetricgrpc.WithEndpoint(fmt.Sprintf("%s:%s", opts.Host, opts.Port)),
			otlpmetricgrpc.WithInsecure(),
		)
		if err != nil {
			return nil, err
		}
	} else {
		exporter, err = stdoutmetric.New(
			stdoutmetric.WithEncoder(json.NewEncoder(io.Discard)),
			stdoutmetric.WithoutTimestamps(),
		)
		if err != nil {
			return nil, err
		}
	}

	// labels/tags/resources that are common to all metrics.
	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String(opts.ServiceName),
		semconv.ServiceVersionKey.String(opts.ServiceVersion),
	)

	provider := metric.NewMeterProvider(
		metric.WithResource(res),
		metric.WithReader(
			// collects and exports metric data every 10 seconds.
			metric.NewPeriodicReader(exporter, metric.WithInterval(10*time.Second)),
		),
	)

	otel.SetMeterProvider(provider)

	return provider, nil
}
