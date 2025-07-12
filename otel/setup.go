package otel

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/trace"

	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

// Options contains configuration options for OpenTelemetry setup functions
type Options struct {
	ServiceName     string
	ServiceVersion  string
	Host            string
	Port            string
	CollectorEnable bool
}

// Option is a function that modifies Options
type Option func(*Options)

// WithServiceName sets the service name
func WithServiceName(name string) Option {
	return func(opts *Options) {
		opts.ServiceName = name
	}
}

// WithServiceVersion sets the service version
func WithServiceVersion(version string) Option {
	return func(opts *Options) {
		opts.ServiceVersion = version
	}
}

// WithHost sets the host address
func WithHost(host string) Option {
	return func(opts *Options) {
		opts.Host = host
	}
}

// WithPort sets the port number
func WithPort(port string) Option {
	return func(opts *Options) {
		opts.Port = port
	}
}

// WithCollectorEnable enables or disables the collector
func WithCollectorEnable(enable bool) Option {
	return func(opts *Options) {
		opts.CollectorEnable = enable
	}
}

// newSetupOptions creates a new SetupOptions with default values and applies the given options
func newSetupOptions(options ...Option) *Options {
	opts := &Options{
		ServiceName:     "unknown",
		ServiceVersion:  "unknown",
		Host:            "localhost",
		Port:            "4317",
		CollectorEnable: false,
	}

	for _, option := range options {
		option(opts)
	}

	return opts
}

// SetupResult contains the initialized providers and cleanup functions
type SetupResult struct {
	TracerProvider *trace.TracerProvider
	LoggerProvider *log.LoggerProvider
	MeterProvider  *sdkmetric.MeterProvider
	Cleanup        func() error
}

// Setup initializes all OpenTelemetry components (tracing, logging, metrics) with the given options
// and returns a SetupResult containing the providers and a cleanup function.
func Setup(ctx context.Context, options ...Option) (*SetupResult, error) {

	// Setup tracing
	tp, err := SetupTracing(ctx, options...)
	if err != nil {
		return nil, fmt.Errorf("failed to setup tracing: %w", err)
	}

	// Setup logging
	lp, err := SetupLogging(ctx, options...)
	if err != nil {
		// Cleanup tracing provider if logging setup fails
		if shutdownErr := tp.Shutdown(ctx); shutdownErr != nil {
			return nil, fmt.Errorf("failed to setup logging: %w, and failed to cleanup tracing: %w", err, shutdownErr)
		}
		return nil, fmt.Errorf("failed to setup logging: %w", err)
	}

	// Setup metrics
	mp, err := SetupMetrics(ctx, options...)
	if err != nil {
		// Cleanup tracing and logging providers if metrics setup fails
		cleanupErrs := []error{}
		if shutdownErr := tp.Shutdown(ctx); shutdownErr != nil {
			cleanupErrs = append(cleanupErrs, fmt.Errorf("failed to cleanup tracing: %w", shutdownErr))
		}
		if shutdownErr := lp.Shutdown(ctx); shutdownErr != nil {
			cleanupErrs = append(cleanupErrs, fmt.Errorf("failed to cleanup logging: %w", shutdownErr))
		}
		if len(cleanupErrs) > 0 {
			return nil, fmt.Errorf("failed to setup metrics: %w, cleanup errors: %v", err, cleanupErrs)
		}
		return nil, fmt.Errorf("failed to setup metrics: %w", err)
	}

	// Create cleanup function that shuts down all providers
	cleanup := func() error {
		var errs []error

		if err := tp.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("failed to shutdown tracer provider: %w", err))
		}

		if err := lp.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("failed to shutdown logger provider: %w", err))
		}

		if err := mp.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("failed to shutdown meter provider: %w", err))
		}

		if len(errs) > 0 {
			return fmt.Errorf("cleanup errors: %v", errs)
		}

		return nil
	}

	return &SetupResult{
		TracerProvider: tp,
		LoggerProvider: lp,
		MeterProvider:  mp,
		Cleanup:        cleanup,
	}, nil
}

// SetupWithCleanup is a convenience function that sets up OpenTelemetry and returns
// a cleanup function that can be used with defer. It panics if setup fails.
func SetupWithCleanup(ctx context.Context, options ...Option) func() {
	result, err := Setup(ctx, options...)
	if err != nil {
		panic(fmt.Sprintf("failed to setup OpenTelemetry: %v", err))
	}

	return func() {
		if err := result.Cleanup(); err != nil {
			// Log the error but don't panic during cleanup
			// You might want to use a logger here if available
			fmt.Printf("warning: failed to cleanup OpenTelemetry: %v\n", err)
		}
	}
}
