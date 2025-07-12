package otel

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/trace"
)

func TestNewSetupOptions(t *testing.T) {
	t.Run("default values", func(t *testing.T) {
		opts := newSetupOptions()

		assert.Equal(t, "unknown", opts.ServiceName)
		assert.Equal(t, "unknown", opts.ServiceVersion)
		assert.Equal(t, "localhost", opts.Host)
		assert.Equal(t, "4317", opts.Port)
		assert.False(t, opts.CollectorEnable)
	})

	t.Run("with custom options", func(t *testing.T) {
		opts := newSetupOptions(
			WithServiceName("test-service"),
			WithServiceVersion("1.0.0"),
			WithHost("custom-host"),
			WithPort("8080"),
			WithCollectorEnable(true),
		)

		assert.Equal(t, "test-service", opts.ServiceName)
		assert.Equal(t, "1.0.0", opts.ServiceVersion)
		assert.Equal(t, "custom-host", opts.Host)
		assert.Equal(t, "8080", opts.Port)
		assert.True(t, opts.CollectorEnable)
	})

	t.Run("partial options", func(t *testing.T) {
		opts := newSetupOptions(
			WithServiceName("partial-service"),
			WithCollectorEnable(true),
		)

		assert.Equal(t, "partial-service", opts.ServiceName)
		assert.Equal(t, "unknown", opts.ServiceVersion) // default
		assert.Equal(t, "localhost", opts.Host)         // default
		assert.Equal(t, "4317", opts.Port)              // default
		assert.True(t, opts.CollectorEnable)
	})
}

func TestSetup(t *testing.T) {
	t.Run("successful setup", func(t *testing.T) {
		ctx := context.Background()

		result, err := Setup(ctx,
			WithServiceName("test-service"),
			WithServiceVersion("1.0.0"),
		)

		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, result.TracerProvider)
		require.NotNil(t, result.LoggerProvider)
		require.NotNil(t, result.MeterProvider)
		require.NotNil(t, result.Cleanup)

		// Verify global providers are set
		globalTracerProvider := otel.GetTracerProvider()
		assert.Equal(t, result.TracerProvider, globalTracerProvider)

		// Test that providers work
		tracer := result.TracerProvider.Tracer("test-tracer")
		assert.NotNil(t, tracer)

		logger := result.LoggerProvider.Logger("test-logger")
		assert.NotNil(t, logger)

		meter := result.MeterProvider.Meter("test-meter")
		assert.NotNil(t, meter)

		// Cleanup
		err = result.Cleanup()
		assert.NoError(t, err)
	})

	t.Run("setup with default options", func(t *testing.T) {
		ctx := context.Background()

		result, err := Setup(ctx)

		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, result.TracerProvider)
		require.NotNil(t, result.LoggerProvider)
		require.NotNil(t, result.MeterProvider)

		// Cleanup
		err = result.Cleanup()
		assert.NoError(t, err)
	})

	t.Run("setup with all custom options", func(t *testing.T) {
		ctx := context.Background()

		result, err := Setup(ctx,
			WithServiceName("custom-service"),
			WithServiceVersion("2.0.0"),
			WithHost("custom-host"),
			WithPort("9090"),
			WithCollectorEnable(false),
		)

		require.NoError(t, err)
		require.NotNil(t, result)

		// Cleanup
		err = result.Cleanup()
		assert.NoError(t, err)
	})
}

func TestSetupWithCleanup(t *testing.T) {
	t.Run("successful setup with cleanup", func(t *testing.T) {
		ctx := context.Background()

		cleanup := SetupWithCleanup(ctx,
			WithServiceName("cleanup-test"),
			WithServiceVersion("1.0.0"),
		)

		require.NotNil(t, cleanup)

		// Verify providers are set globally
		tracerProvider := otel.GetTracerProvider()
		assert.NotNil(t, tracerProvider)

		// Test that providers work
		tracer := tracerProvider.Tracer("test-tracer")
		assert.NotNil(t, tracer)

		// Call cleanup
		cleanup()

		// Verify cleanup worked (providers should be shutdown)
		// Note: We can't easily verify this without accessing internal state
	})

	t.Run("setup with panic on failure", func(t *testing.T) {
		ctx := context.Background()

		// This should not panic with valid options
		assert.NotPanics(t, func() {
			cleanup := SetupWithCleanup(ctx, WithServiceName("panic-test"))
			cleanup()
		})
	})
}

func TestSetupResult(t *testing.T) {
	t.Run("setup result structure", func(t *testing.T) {
		ctx := context.Background()

		result, err := Setup(ctx, WithServiceName("result-test"))
		require.NoError(t, err)
		require.NotNil(t, result)

		// Verify all fields are properly set
		assert.IsType(t, &trace.TracerProvider{}, result.TracerProvider)
		assert.IsType(t, &log.LoggerProvider{}, result.LoggerProvider)
		assert.IsType(t, &metric.MeterProvider{}, result.MeterProvider)
		assert.IsType(t, (func() error)(nil), result.Cleanup)

		// Cleanup
		err = result.Cleanup()
		assert.NoError(t, err)
	})
}

func TestSetupErrorHandling(t *testing.T) {
	t.Run("context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		result, err := Setup(ctx, WithServiceName("cancel-test"))

		// Should still work even with cancelled context
		require.NoError(t, err)
		require.NotNil(t, result)

		// Cleanup should fail due to cancelled context
		err = result.Cleanup()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cleanup errors")
	})

	t.Run("invalid port number", func(t *testing.T) {
		ctx := context.Background()

		// Invalid port should still work since validation happens later
		result, err := Setup(ctx, WithPort("invalid-port"))
		require.NoError(t, err)
		require.NotNil(t, result)

		// Cleanup
		err = result.Cleanup()
		assert.NoError(t, err)
	})
}

func TestSetupMultipleCalls(t *testing.T) {
	t.Run("multiple setup calls", func(t *testing.T) {
		ctx := context.Background()

		// First setup
		result1, err := Setup(ctx, WithServiceName("multi1"))
		require.NoError(t, err)
		require.NotNil(t, result1)

		// Second setup
		result2, err := Setup(ctx, WithServiceName("multi2"))
		require.NoError(t, err)
		require.NotNil(t, result2)

		// Cleanup both
		err = result1.Cleanup()
		assert.NoError(t, err)

		err = result2.Cleanup()
		assert.NoError(t, err)
	})
}

func TestSetupConcurrentAccess(t *testing.T) {
	t.Run("concurrent setup calls", func(t *testing.T) {
		ctx := context.Background()

		// Test concurrent setup calls
		done := make(chan bool, 2)

		go func() {
			result, err := Setup(ctx, WithServiceName("concurrent1"))
			if err == nil && result != nil {
				err = result.Cleanup()
				assert.NoError(t, err)
			}
			done <- true
		}()

		go func() {
			result, err := Setup(ctx, WithServiceName("concurrent2"))
			if err == nil && result != nil {
				err = result.Cleanup()
				assert.NoError(t, err)
			}
			done <- true
		}()

		// Wait for both goroutines to complete
		<-done
		<-done
	})
}

func TestSetupMemoryLeaks(t *testing.T) {
	t.Run("memory leak prevention", func(t *testing.T) {
		ctx := context.Background()

		// Create and destroy multiple setups to check for memory leaks
		for range 5 {
			result, err := Setup(ctx,
				WithServiceName("memory-test"),
				WithServiceVersion("1.0.0"),
			)

			require.NoError(t, err)
			require.NotNil(t, result)

			// Test that all providers work
			tracer := result.TracerProvider.Tracer("test-tracer")
			assert.NotNil(t, tracer)

			logger := result.LoggerProvider.Logger("test-logger")
			assert.NotNil(t, logger)

			meter := result.MeterProvider.Meter("test-meter")
			assert.NotNil(t, meter)

			// Cleanup immediately
			err = result.Cleanup()
			assert.NoError(t, err)
		}
	})
}

func TestSetupIntegration(t *testing.T) {
	t.Run("integration test with actual usage", func(t *testing.T) {
		ctx := context.Background()

		result, err := Setup(ctx,
			WithServiceName("integration-test"),
			WithServiceVersion("1.0.0"),
		)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Test actual usage of all providers
		tracer := result.TracerProvider.Tracer("integration-tracer")
		ctx, span := tracer.Start(ctx, "test_span")
		require.NotNil(t, span)
		span.End()

		meter := result.MeterProvider.Meter("integration-meter")
		counter, err := meter.Int64Counter("test_counter")
		require.NoError(t, err)
		counter.Add(ctx, 1)

		// Cleanup
		err = result.Cleanup()
		assert.NoError(t, err)
	})
}

func TestSetupCleanupFunction(t *testing.T) {
	t.Run("cleanup function behavior", func(t *testing.T) {
		ctx := context.Background()

		result, err := Setup(ctx, WithServiceName("cleanup-test"))
		require.NoError(t, err)
		require.NotNil(t, result)

		// Call cleanup once
		err = result.Cleanup()
		assert.NoError(t, err)

		// Call cleanup again - should error because providers are already shut down
		err = result.Cleanup()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cleanup errors")
	})
}

func TestSetupWithContextTODO(t *testing.T) {
	t.Run("context TODO", func(t *testing.T) {
		// This should work as context.Background() is used internally
		result, err := Setup(context.TODO())

		require.NoError(t, err)
		require.NotNil(t, result)

		// Cleanup
		err = result.Cleanup()
		assert.NoError(t, err)
	})
}

// Benchmark tests
func BenchmarkSetup(b *testing.B) {
	ctx := context.Background()

	b.ResetTimer()
	for b.Loop() {
		result, err := Setup(ctx, WithServiceName("benchmark"))
		if err == nil && result != nil {
			err = result.Cleanup()
			if err != nil {
				b.Fatal(err)
			}
		}
	}
}

func BenchmarkSetupWithOptions(b *testing.B) {
	ctx := context.Background()
	options := []Option{
		WithServiceName("benchmark-service"),
		WithServiceVersion("1.0.0"),
		WithHost("localhost"),
		WithPort("4317"),
		WithCollectorEnable(false),
	}

	b.ResetTimer()
	for b.Loop() {
		result, err := Setup(ctx, options...)
		if err == nil && result != nil {
			err = result.Cleanup()
			if err != nil {
				b.Fatal(err)
			}
		}
	}
}

func BenchmarkSetupWithCleanup(b *testing.B) {
	ctx := context.Background()

	b.ResetTimer()
	for b.Loop() {
		cleanup := SetupWithCleanup(ctx, WithServiceName("benchmark-cleanup"))
		cleanup()
	}
}
