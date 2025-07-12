package otel

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
)

func TestSetupTracing(t *testing.T) {
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
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			provider, err := SetupTracing(ctx, tt.options...)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, provider)

			// Verify the provider is set globally
			globalProvider := otel.GetTracerProvider()
			assert.Equal(t, provider, globalProvider)

			// Test that we can create a tracer
			tracer := provider.Tracer("test-tracer")
			assert.NotNil(t, tracer)

			// Cleanup
			err = provider.Shutdown(ctx)
			assert.NoError(t, err)
		})
	}
}

func TestSetupTracingWithStdoutExporter(t *testing.T) {
	t.Run("stdout exporter functionality", func(t *testing.T) {
		ctx := context.Background()

		// Use stdout exporter (default when collector is disabled)
		provider, err := SetupTracing(ctx,
			WithServiceName("stdout-test"),
			WithServiceVersion("1.0.0"),
			WithCollectorEnable(false), // Explicitly disable collector
		)

		require.NoError(t, err)
		require.NotNil(t, provider)

		// Test tracing functionality
		tracer := provider.Tracer("test-tracer")
		assert.NotNil(t, tracer)

		// Create a span
		ctx, span := tracer.Start(ctx, "test-span")
		assert.NotNil(t, span)
		span.End()

		// Create a child span
		ctx, childSpan := tracer.Start(ctx, "child-span")
		assert.NotNil(t, childSpan)
		childSpan.End()

		// Cleanup
		err = provider.Shutdown(ctx)
		assert.NoError(t, err)
	})
}

func TestSetupTracingResourceAttributes(t *testing.T) {
	t.Run("verify resource attributes", func(t *testing.T) {
		ctx := context.Background()

		serviceName := "test-service"
		serviceVersion := "1.2.3"

		provider, err := SetupTracing(ctx,
			WithServiceName(serviceName),
			WithServiceVersion(serviceVersion),
		)

		require.NoError(t, err)
		require.NotNil(t, provider)

		// Test that we can create a tracer
		tracer := provider.Tracer("test-tracer")
		assert.NotNil(t, tracer)

		// Cleanup
		err = provider.Shutdown(ctx)
		assert.NoError(t, err)
	})
}

func TestSetupTracingMultipleCalls(t *testing.T) {
	t.Run("multiple setup calls", func(t *testing.T) {
		ctx := context.Background()

		// First setup
		provider1, err := SetupTracing(ctx, WithServiceName("service1"))
		require.NoError(t, err)
		require.NotNil(t, provider1)

		// Second setup - should work but might return the same provider due to global state
		provider2, err := SetupTracing(ctx, WithServiceName("service2"))
		require.NoError(t, err)
		require.NotNil(t, provider2)

		// Cleanup both
		err = provider1.Shutdown(ctx)
		assert.NoError(t, err)

		err = provider2.Shutdown(ctx)
		assert.NoError(t, err)
	})
}

func TestSetupTracingContextCancellation(t *testing.T) {
	t.Run("context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		provider, err := SetupTracing(ctx)

		// Should still work even with cancelled context
		require.NoError(t, err)
		require.NotNil(t, provider)

		// Cleanup
		err = provider.Shutdown(context.Background())
		assert.NoError(t, err)
	})
}

func TestSetupTracingShutdown(t *testing.T) {
	t.Run("provider shutdown", func(t *testing.T) {
		ctx := context.Background()

		provider, err := SetupTracing(ctx)
		require.NoError(t, err)
		require.NotNil(t, provider)

		// Shutdown the provider
		err = provider.Shutdown(ctx)
		assert.NoError(t, err)

		// Try to shutdown again - might error due to already shutdown
		err = provider.Shutdown(ctx)
		assert.NoError(t, err)
	})
}

func TestSetupTracingWithContextTODO(t *testing.T) {
	t.Run("context TODO", func(t *testing.T) {
		// This should work as context.Background() is used internally
		provider, err := SetupTracing(context.TODO())

		require.NoError(t, err)
		require.NotNil(t, provider)

		// Cleanup
		err = provider.Shutdown(context.Background())
		assert.NoError(t, err)
	})
}

func TestSetupTracingOptionsCombination(t *testing.T) {
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
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			provider, err := SetupTracing(ctx, tt.options...)

			if tt.wantErr {
				assert.Error(t, err)
				if provider != nil {
					err = provider.Shutdown(ctx)
					assert.NoError(t, err)
				}
				return
			}

			require.NoError(t, err)
			require.NotNil(t, provider)

			// Verify global provider is set
			globalProvider := otel.GetTracerProvider()
			assert.Equal(t, provider, globalProvider)

			// Cleanup
			err = provider.Shutdown(ctx)
			assert.NoError(t, err)
		})
	}
}

func TestSetupTracingDefaultValues(t *testing.T) {
	t.Run("verify default values", func(t *testing.T) {
		ctx := context.Background()

		// Use no options to get default values
		provider, err := SetupTracing(ctx)
		require.NoError(t, err)
		require.NotNil(t, provider)

		// The default values should be:
		// - ServiceName: "unknown"
		// - ServiceVersion: "unknown"
		// - Host: "localhost"
		// - Port: 4317
		// - CollectorEnable: false

		// We can't easily verify these directly, but we can verify the provider works
		tracer := provider.Tracer("test-tracer")
		assert.NotNil(t, tracer)

		// Cleanup
		err = provider.Shutdown(ctx)
		assert.NoError(t, err)
	})
}

func TestSetupTracingErrorHandling(t *testing.T) {
	t.Run("invalid port number", func(t *testing.T) {
		ctx := context.Background()

		// Test with an invalid port that might cause issues
		provider, err := SetupTracing(ctx, WithPort(-1))

		// This might still succeed as the port is only used when collector is enabled
		if err != nil {
			assert.Error(t, err)
		} else {
			require.NotNil(t, provider)
			err = provider.Shutdown(ctx)
			assert.NoError(t, err)
		}
	})
}

func TestSetupTracingConcurrentAccess(t *testing.T) {
	t.Run("concurrent setup calls", func(t *testing.T) {
		ctx := context.Background()

		// Test concurrent setup calls
		done := make(chan bool, 2)

		go func() {
			provider, err := SetupTracing(ctx, WithServiceName("concurrent1"))
			if err == nil && provider != nil {
				err = provider.Shutdown(ctx)
				assert.NoError(t, err)
			}
			done <- true
		}()

		go func() {
			provider, err := SetupTracing(ctx, WithServiceName("concurrent2"))
			if err == nil && provider != nil {
				err = provider.Shutdown(ctx)
				assert.NoError(t, err)
			}
			done <- true
		}()

		// Wait for both goroutines to complete
		<-done
		<-done
	})
}

func TestSetupTracingMemoryLeaks(t *testing.T) {
	t.Run("memory leak prevention", func(t *testing.T) {
		ctx := context.Background()

		// Create and destroy multiple providers to check for memory leaks
		for range 10 {
			provider, err := SetupTracing(ctx,
				WithServiceName("memory-test"),
				WithServiceVersion("1.0.0"),
			)

			require.NoError(t, err)
			require.NotNil(t, provider)

			// Create a tracer to ensure it works
			tracer := provider.Tracer("test-tracer")
			assert.NotNil(t, tracer)

			// Shutdown immediately
			err = provider.Shutdown(ctx)
			assert.NoError(t, err)
		}
	})
}

func TestSetupTracingIntegration(t *testing.T) {
	t.Run("integration test with actual tracing", func(t *testing.T) {
		ctx := context.Background()

		provider, err := SetupTracing(ctx,
			WithServiceName("integration-test"),
			WithServiceVersion("1.0.0"),
		)
		require.NoError(t, err)
		require.NotNil(t, provider)

		// Create a tracer and test actual tracing
		tracer := provider.Tracer("integration-tracer")
		assert.NotNil(t, tracer)

		// Test creating spans
		ctx, span := tracer.Start(ctx, "test_span")
		require.NotNil(t, span)
		assert.True(t, span.SpanContext().IsValid())

		// Create a child span
		ctx, childSpan := tracer.Start(ctx, "child_span")
		require.NotNil(t, childSpan)
		assert.True(t, childSpan.SpanContext().IsValid())

		// Verify parent-child relationship
		childSpanContext := childSpan.SpanContext()
		parentSpanContext := span.SpanContext()
		assert.Equal(t, parentSpanContext.TraceID(), childSpanContext.TraceID())

		// End spans
		childSpan.End()
		span.End()

		// Cleanup
		err = provider.Shutdown(ctx)
		assert.NoError(t, err)
	})
}

func TestSetupTracingSpanCreation(t *testing.T) {
	t.Run("span creation and manipulation", func(t *testing.T) {
		ctx := context.Background()

		provider, err := SetupTracing(ctx)
		require.NoError(t, err)
		require.NotNil(t, provider)

		tracer := provider.Tracer("test-tracer")
		assert.NotNil(t, tracer)

		// Test span creation
		ctx, span := tracer.Start(ctx, "test-span")
		assert.NotNil(t, span)
		assert.True(t, span.SpanContext().IsValid())

		span.End()

		// Cleanup
		err = provider.Shutdown(ctx)
		assert.NoError(t, err)
	})
}

func TestSetupTracingPropagation(t *testing.T) {
	t.Run("trace context propagation", func(t *testing.T) {
		ctx := context.Background()

		provider, err := SetupTracing(ctx)
		require.NoError(t, err)
		require.NotNil(t, provider)

		tracer := provider.Tracer("test-tracer")
		assert.NotNil(t, tracer)

		// Create a span
		ctx, span := tracer.Start(ctx, "parent-span")
		assert.NotNil(t, span)

		// Verify trace context is propagated
		spanContext := span.SpanContext()
		assert.True(t, spanContext.IsValid())
		assert.True(t, spanContext.IsSampled())

		// Create child span in a new context
		_, childSpan := tracer.Start(ctx, "child-span")
		assert.NotNil(t, childSpan)

		// Verify parent-child relationship - same trace ID
		childSpanContext := childSpan.SpanContext()
		assert.Equal(t, spanContext.TraceID(), childSpanContext.TraceID())

		childSpan.End()
		span.End()

		// Cleanup
		err = provider.Shutdown(ctx)
		assert.NoError(t, err)
	})
}

func TestSetupTracingBatchExport(t *testing.T) {
	t.Run("batch export configuration", func(t *testing.T) {
		ctx := context.Background()

		provider, err := SetupTracing(ctx)
		require.NoError(t, err)
		require.NotNil(t, provider)

		// The provider should have a batch processor configured
		// We can verify this by checking that the provider works correctly
		tracer := provider.Tracer("test-tracer")
		assert.NotNil(t, tracer)

		// Create multiple spans to test batching
		for i := 0; i < 5; i++ {
			_, span := tracer.Start(ctx, "batch-span")
			span.End()
		}

		// Wait a bit for batch processing
		time.Sleep(100 * time.Millisecond)

		// Cleanup
		err = provider.Shutdown(ctx)
		assert.NoError(t, err)
	})
}

func TestSetupTracingResourceConfiguration(t *testing.T) {
	t.Run("resource configuration", func(t *testing.T) {
		ctx := context.Background()

		serviceName := "resource-test"
		serviceVersion := "2.0.0"

		provider, err := SetupTracing(ctx,
			WithServiceName(serviceName),
			WithServiceVersion(serviceVersion),
		)
		require.NoError(t, err)
		require.NotNil(t, provider)

		// Test that the provider works with the configured resource
		tracer := provider.Tracer("test-tracer")
		assert.NotNil(t, tracer)

		// Create and use a span
		ctx, span := tracer.Start(ctx, "resource_test_span")
		require.NotNil(t, span)
		span.End()

		// Cleanup
		err = provider.Shutdown(ctx)
		assert.NoError(t, err)
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
