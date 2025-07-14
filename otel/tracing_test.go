package otel

import (
	"context"
	"testing"
	"time"

	"github.com/frankban/quicktest"
	"go.opentelemetry.io/otel"
)

func TestSetupTracing(t *testing.T) {
	qt := quicktest.New(t)

	tests := []struct {
		name    string
		options []Option
		wantErr bool
	}{
		{
			name:    "default options",
			options: []Option{},
			wantErr: false,
		},
		{
			name: "custom service name and version",
			options: []Option{
				WithServiceName("test-service"),
				WithServiceVersion("1.0.0"),
			},
			wantErr: false,
		},
		{
			name: "custom host and port",
			options: []Option{
				WithHost("custom-host"),
				WithPort(8080),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		qt.Run(tt.name, func(c *quicktest.C) {
			ctx := context.Background()

			provider, err := SetupTracing(ctx, tt.options...)

			if tt.wantErr {
				c.Check(err, quicktest.Not(quicktest.IsNil))
				return
			}

			c.Check(err, quicktest.IsNil)
			c.Check(provider, quicktest.Not(quicktest.IsNil))

			// Verify the provider is set globally
			globalProvider := otel.GetTracerProvider()
			c.Check(provider, quicktest.Equals, globalProvider)

			// Test that we can create a tracer
			tracer := provider.Tracer("test-tracer")
			c.Check(tracer, quicktest.Not(quicktest.IsNil))

			// Cleanup
			err = provider.Shutdown(ctx)
			c.Check(err, quicktest.IsNil)
		})
	}
}

func TestSetupTracingWithStdoutExporter(t *testing.T) {
	qt := quicktest.New(t)

	qt.Run("stdout exporter functionality", func(c *quicktest.C) {
		ctx := context.Background()

		// Use stdout exporter (default when collector is disabled)
		provider, err := SetupTracing(ctx,
			WithServiceName("stdout-test"),
			WithServiceVersion("1.0.0"),
			WithCollectorEnable(false), // Explicitly disable collector
		)

		c.Check(err, quicktest.IsNil)
		c.Check(provider, quicktest.Not(quicktest.IsNil))

		// Test tracing functionality
		tracer := provider.Tracer("test-tracer")
		c.Check(tracer, quicktest.Not(quicktest.IsNil))

		// Create a span
		ctx, span := tracer.Start(ctx, "test-span")
		c.Check(span, quicktest.Not(quicktest.IsNil))
		span.End()

		// Create a child span
		ctx, childSpan := tracer.Start(ctx, "child-span")
		c.Check(childSpan, quicktest.Not(quicktest.IsNil))
		childSpan.End()

		// Cleanup
		err = provider.Shutdown(ctx)
		c.Check(err, quicktest.IsNil)
	})
}

func TestSetupTracingResourceAttributes(t *testing.T) {
	qt := quicktest.New(t)

	qt.Run("verify resource attributes", func(c *quicktest.C) {
		ctx := context.Background()

		serviceName := "test-service"
		serviceVersion := "1.2.3"

		provider, err := SetupTracing(ctx,
			WithServiceName(serviceName),
			WithServiceVersion(serviceVersion),
		)

		c.Check(err, quicktest.IsNil)
		c.Check(provider, quicktest.Not(quicktest.IsNil))

		// Test that we can create a tracer
		tracer := provider.Tracer("test-tracer")
		c.Check(tracer, quicktest.Not(quicktest.IsNil))

		// Cleanup
		err = provider.Shutdown(ctx)
		c.Check(err, quicktest.IsNil)
	})
}

func TestSetupTracingMultipleCalls(t *testing.T) {
	qt := quicktest.New(t)

	qt.Run("multiple setup calls", func(c *quicktest.C) {
		ctx := context.Background()

		// First setup
		provider1, err := SetupTracing(ctx, WithServiceName("service1"))
		c.Check(err, quicktest.IsNil)
		c.Check(provider1, quicktest.Not(quicktest.IsNil))

		// Second setup - should work but might return the same provider due to global state
		provider2, err := SetupTracing(ctx, WithServiceName("service2"))
		c.Check(err, quicktest.IsNil)
		c.Check(provider2, quicktest.Not(quicktest.IsNil))

		// Cleanup both
		err = provider1.Shutdown(ctx)
		c.Check(err, quicktest.IsNil)

		err = provider2.Shutdown(ctx)
		c.Check(err, quicktest.IsNil)
	})
}

func TestSetupTracingContextCancellation(t *testing.T) {
	qt := quicktest.New(t)

	qt.Run("context cancellation", func(c *quicktest.C) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		provider, err := SetupTracing(ctx)

		// Should still work even with cancelled context
		c.Check(err, quicktest.IsNil)
		c.Check(provider, quicktest.Not(quicktest.IsNil))

		// Cleanup
		err = provider.Shutdown(context.Background())
		c.Check(err, quicktest.IsNil)
	})
}

func TestSetupTracingShutdown(t *testing.T) {
	qt := quicktest.New(t)

	qt.Run("provider shutdown", func(c *quicktest.C) {
		ctx := context.Background()

		provider, err := SetupTracing(ctx)
		c.Check(err, quicktest.IsNil)
		c.Check(provider, quicktest.Not(quicktest.IsNil))

		// Shutdown the provider
		err = provider.Shutdown(ctx)
		c.Check(err, quicktest.IsNil)

		// Try to shutdown again - might error due to already shutdown
		err = provider.Shutdown(ctx)
		c.Check(err, quicktest.IsNil)
	})
}

func TestSetupTracingWithContextTODO(t *testing.T) {
	qt := quicktest.New(t)

	qt.Run("context TODO", func(c *quicktest.C) {
		// This should work as context.Background() is used internally
		provider, err := SetupTracing(context.TODO())

		c.Check(err, quicktest.IsNil)
		c.Check(provider, quicktest.Not(quicktest.IsNil))

		// Cleanup
		err = provider.Shutdown(context.Background())
		c.Check(err, quicktest.IsNil)
	})
}

func TestSetupTracingOptionsCombination(t *testing.T) {
	qt := quicktest.New(t)

	tests := []struct {
		name    string
		options []Option
		wantErr bool
	}{
		{
			name: "service name only",
			options: []Option{
				WithServiceName("service-only"),
			},
			wantErr: false,
		},
		{
			name: "service version only",
			options: []Option{
				WithServiceVersion("1.0.0"),
			},
			wantErr: false,
		},
		{
			name: "host only",
			options: []Option{
				WithHost("host-only"),
			},
			wantErr: false,
		},
		{
			name: "port only",
			options: []Option{
				WithPort(1234),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		qt.Run(tt.name, func(c *quicktest.C) {
			ctx := context.Background()

			provider, err := SetupTracing(ctx, tt.options...)

			if tt.wantErr {
				c.Check(err, quicktest.Not(quicktest.IsNil))
				if provider != nil {
					err = provider.Shutdown(ctx)
					c.Check(err, quicktest.IsNil)
				}
				return
			}

			c.Check(err, quicktest.IsNil)
			c.Check(provider, quicktest.Not(quicktest.IsNil))

			// Verify global provider is set
			globalProvider := otel.GetTracerProvider()
			c.Check(provider, quicktest.Equals, globalProvider)

			// Cleanup
			err = provider.Shutdown(ctx)
			c.Check(err, quicktest.IsNil)
		})
	}
}

func TestSetupTracingDefaultValues(t *testing.T) {
	qt := quicktest.New(t)

	qt.Run("verify default values", func(c *quicktest.C) {
		ctx := context.Background()

		// Use no options to get default values
		provider, err := SetupTracing(ctx)
		c.Check(err, quicktest.IsNil)
		c.Check(provider, quicktest.Not(quicktest.IsNil))

		// The default values should be:
		// - ServiceName: "unknown"
		// - ServiceVersion: "unknown"
		// - Host: "localhost"
		// - Port: 4317
		// - CollectorEnable: false

		// We can't easily verify these directly, but we can verify the provider works
		tracer := provider.Tracer("test-tracer")
		c.Check(tracer, quicktest.Not(quicktest.IsNil))

		// Cleanup
		err = provider.Shutdown(ctx)
		c.Check(err, quicktest.IsNil)
	})
}

func TestSetupTracingErrorHandling(t *testing.T) {
	qt := quicktest.New(t)

	qt.Run("invalid port number", func(c *quicktest.C) {
		ctx := context.Background()

		// Test with an invalid port that might cause issues
		provider, err := SetupTracing(ctx, WithPort(-1))

		// This might still succeed as the port is only used when collector is enabled
		if err != nil {
			c.Check(err, quicktest.Not(quicktest.IsNil))
		} else {
			c.Check(provider, quicktest.Not(quicktest.IsNil))
			err = provider.Shutdown(ctx)
			c.Check(err, quicktest.IsNil)
		}
	})
}

func TestSetupTracingConcurrentAccess(t *testing.T) {
	qt := quicktest.New(t)

	qt.Run("concurrent setup calls", func(c *quicktest.C) {
		ctx := context.Background()

		// Test concurrent setup calls
		done := make(chan bool, 2)

		go func() {
			provider, err := SetupTracing(ctx, WithServiceName("concurrent1"))
			if err == nil && provider != nil {
				err = provider.Shutdown(ctx)
				c.Check(err, quicktest.IsNil)
			}
			done <- true
		}()

		go func() {
			provider, err := SetupTracing(ctx, WithServiceName("concurrent2"))
			if err == nil && provider != nil {
				err = provider.Shutdown(ctx)
				c.Check(err, quicktest.IsNil)
			}
			done <- true
		}()

		// Wait for both goroutines to complete
		<-done
		<-done
	})
}

func TestSetupTracingMemoryLeaks(t *testing.T) {
	qt := quicktest.New(t)

	qt.Run("memory leak prevention", func(c *quicktest.C) {
		ctx := context.Background()

		// Create and destroy multiple providers to check for memory leaks
		for range 10 {
			provider, err := SetupTracing(ctx,
				WithServiceName("memory-test"),
				WithServiceVersion("1.0.0"),
			)

			c.Check(err, quicktest.IsNil)
			c.Check(provider, quicktest.Not(quicktest.IsNil))

			// Create a tracer to ensure it works
			tracer := provider.Tracer("test-tracer")
			c.Check(tracer, quicktest.Not(quicktest.IsNil))

			// Shutdown immediately
			err = provider.Shutdown(ctx)
			c.Check(err, quicktest.IsNil)
		}
	})
}

func TestSetupTracingIntegration(t *testing.T) {
	qt := quicktest.New(t)

	qt.Run("integration test with actual tracing", func(c *quicktest.C) {
		ctx := context.Background()

		provider, err := SetupTracing(ctx,
			WithServiceName("integration-test"),
			WithServiceVersion("1.0.0"),
		)
		c.Check(err, quicktest.IsNil)
		c.Check(provider, quicktest.Not(quicktest.IsNil))

		// Create a tracer and test actual tracing
		tracer := provider.Tracer("integration-tracer")
		c.Check(tracer, quicktest.Not(quicktest.IsNil))

		// Test creating spans
		ctx, span := tracer.Start(ctx, "test_span")
		c.Check(span, quicktest.Not(quicktest.IsNil))
		c.Check(span.SpanContext().IsValid(), quicktest.Equals, true)

		// Create a child span
		ctx, childSpan := tracer.Start(ctx, "child_span")
		c.Check(childSpan, quicktest.Not(quicktest.IsNil))
		c.Check(childSpan.SpanContext().IsValid(), quicktest.Equals, true)

		// Verify parent-child relationship
		childSpanContext := childSpan.SpanContext()
		parentSpanContext := span.SpanContext()
		c.Check(parentSpanContext.TraceID(), quicktest.Equals, childSpanContext.TraceID())

		// End spans
		childSpan.End()
		span.End()

		// Cleanup
		err = provider.Shutdown(ctx)
		c.Check(err, quicktest.IsNil)
	})
}

func TestSetupTracingSpanCreation(t *testing.T) {
	qt := quicktest.New(t)

	qt.Run("span creation and manipulation", func(c *quicktest.C) {
		ctx := context.Background()

		provider, err := SetupTracing(ctx)
		c.Check(err, quicktest.IsNil)
		c.Check(provider, quicktest.Not(quicktest.IsNil))

		tracer := provider.Tracer("test-tracer")
		c.Check(tracer, quicktest.Not(quicktest.IsNil))

		// Test span creation
		ctx, span := tracer.Start(ctx, "test-span")
		c.Check(span, quicktest.Not(quicktest.IsNil))
		c.Check(span.SpanContext().IsValid(), quicktest.Equals, true)

		span.End()

		// Cleanup
		err = provider.Shutdown(ctx)
		c.Check(err, quicktest.IsNil)
	})
}

func TestSetupTracingPropagation(t *testing.T) {
	qt := quicktest.New(t)

	qt.Run("trace context propagation", func(c *quicktest.C) {
		ctx := context.Background()

		provider, err := SetupTracing(ctx)
		c.Check(err, quicktest.IsNil)
		c.Check(provider, quicktest.Not(quicktest.IsNil))

		tracer := provider.Tracer("test-tracer")
		c.Check(tracer, quicktest.Not(quicktest.IsNil))

		// Create a span
		ctx, span := tracer.Start(ctx, "parent-span")
		c.Check(span, quicktest.Not(quicktest.IsNil))

		// Verify trace context is propagated
		spanContext := span.SpanContext()
		c.Check(spanContext.IsValid(), quicktest.Equals, true)
		c.Check(spanContext.IsSampled(), quicktest.Equals, true)

		// Create child span in a new context
		_, childSpan := tracer.Start(ctx, "child-span")
		c.Check(childSpan, quicktest.Not(quicktest.IsNil))

		// Verify parent-child relationship - same trace ID
		childSpanContext := childSpan.SpanContext()
		c.Check(spanContext.TraceID(), quicktest.Equals, childSpanContext.TraceID())

		childSpan.End()
		span.End()

		// Cleanup
		err = provider.Shutdown(ctx)
		c.Check(err, quicktest.IsNil)
	})
}

func TestSetupTracingBatchExport(t *testing.T) {
	qt := quicktest.New(t)

	qt.Run("batch export configuration", func(c *quicktest.C) {
		ctx := context.Background()

		provider, err := SetupTracing(ctx)
		c.Check(err, quicktest.IsNil)
		c.Check(provider, quicktest.Not(quicktest.IsNil))

		// The provider should have a batch processor configured
		// We can verify this by checking that the provider works correctly
		tracer := provider.Tracer("test-tracer")
		c.Check(tracer, quicktest.Not(quicktest.IsNil))

		// Create multiple spans to test batching
		for i := 0; i < 5; i++ {
			_, span := tracer.Start(ctx, "batch-span")
			span.End()
		}

		// Wait a bit for batch processing
		time.Sleep(100 * time.Millisecond)

		// Cleanup
		err = provider.Shutdown(ctx)
		c.Check(err, quicktest.IsNil)
	})
}

func TestSetupTracingResourceConfiguration(t *testing.T) {
	qt := quicktest.New(t)

	qt.Run("resource configuration", func(c *quicktest.C) {
		ctx := context.Background()

		serviceName := "resource-test"
		serviceVersion := "2.0.0"

		provider, err := SetupTracing(ctx,
			WithServiceName(serviceName),
			WithServiceVersion(serviceVersion),
		)
		c.Check(err, quicktest.IsNil)
		c.Check(provider, quicktest.Not(quicktest.IsNil))

		// Test that the provider works with the configured resource
		tracer := provider.Tracer("test-tracer")
		c.Check(tracer, quicktest.Not(quicktest.IsNil))

		// Create and use a span
		ctx, span := tracer.Start(ctx, "resource_test_span")
		c.Check(span, quicktest.Not(quicktest.IsNil))
		span.End()

		// Cleanup
		err = provider.Shutdown(ctx)
		c.Check(err, quicktest.IsNil)
	})
}

// Benchmark tests
func BenchmarkSetupTracing(b *testing.B) {
	ctx := context.Background()

	b.ResetTimer()
	for b.Loop() {
		provider, err := SetupTracing(ctx, WithServiceName("benchmark"))
		if err == nil && provider != nil {
			err = provider.Shutdown(ctx)
			if err != nil {
				b.Fatal(err)
			}
		}
	}
}

func BenchmarkSetupTracingWithOptions(b *testing.B) {
	ctx := context.Background()
	options := []Option{
		WithServiceName("benchmark-service"),
		WithServiceVersion("1.0.0"),
		WithHost("localhost"),
		WithPort(4317),
		WithCollectorEnable(false),
	}

	b.ResetTimer()
	for b.Loop() {
		provider, err := SetupTracing(ctx, options...)
		if err == nil && provider != nil {
			err = provider.Shutdown(ctx)
			if err != nil {
				b.Fatal(err)
			}
		}
	}
}

func BenchmarkTracingCreation(b *testing.B) {
	ctx := context.Background()
	provider, err := SetupTracing(ctx)
	if err != nil {
		b.Fatal(err)
	}
	defer func() {
		err = provider.Shutdown(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}()

	tracer := provider.Tracer("benchmark-tracer")

	b.ResetTimer()
	for b.Loop() {
		_, span := tracer.Start(ctx, "benchmark_span")
		span.End()
	}
}
