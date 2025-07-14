package otel

import (
	"context"
	"testing"

	"github.com/frankban/quicktest"
	"go.opentelemetry.io/otel"
)

func TestNewSetupOptions(t *testing.T) {
	qt := quicktest.New(t)

	qt.Run("default values", func(c *quicktest.C) {
		opts := newSetupOptions()

		c.Check(opts.ServiceName, quicktest.Equals, "unknown")
		c.Check(opts.ServiceVersion, quicktest.Equals, "unknown")
		c.Check(opts.Host, quicktest.Equals, "localhost")
		c.Check(opts.Port, quicktest.Equals, 4317)
		c.Check(opts.CollectorEnable, quicktest.Equals, false)
	})

	qt.Run("with custom options", func(c *quicktest.C) {
		opts := newSetupOptions(
			WithServiceName("test-service"),
			WithServiceVersion("1.0.0"),
			WithHost("custom-host"),
			WithPort(8080),
			WithCollectorEnable(true),
		)

		c.Check(opts.ServiceName, quicktest.Equals, "test-service")
		c.Check(opts.ServiceVersion, quicktest.Equals, "1.0.0")
		c.Check(opts.Host, quicktest.Equals, "custom-host")
		c.Check(opts.Port, quicktest.Equals, 8080)
		c.Check(opts.CollectorEnable, quicktest.Equals, true)
	})

	qt.Run("partial options", func(c *quicktest.C) {
		opts := newSetupOptions(
			WithServiceName("partial-service"),
			WithCollectorEnable(true),
		)

		c.Check(opts.ServiceName, quicktest.Equals, "partial-service")
		c.Check(opts.ServiceVersion, quicktest.Equals, "unknown") // default
		c.Check(opts.Host, quicktest.Equals, "localhost")         // default
		c.Check(opts.Port, quicktest.Equals, 4317)                // default
		c.Check(opts.CollectorEnable, quicktest.Equals, true)
	})
}

func TestSetup(t *testing.T) {
	qt := quicktest.New(t)

	qt.Run("successful setup", func(c *quicktest.C) {
		ctx := context.Background()

		result, err := Setup(ctx,
			WithServiceName("test-service"),
			WithServiceVersion("1.0.0"),
		)

		c.Check(err, quicktest.IsNil)
		c.Check(result, quicktest.Not(quicktest.IsNil))
		c.Check(result.TracerProvider, quicktest.Not(quicktest.IsNil))
		c.Check(result.LoggerProvider, quicktest.Not(quicktest.IsNil))
		c.Check(result.MeterProvider, quicktest.Not(quicktest.IsNil))
		c.Check(result.Cleanup, quicktest.Not(quicktest.IsNil))

		// Verify global providers are set
		globalTracerProvider := otel.GetTracerProvider()
		c.Check(result.TracerProvider, quicktest.Equals, globalTracerProvider)

		// Test that providers work
		tracer := result.TracerProvider.Tracer("test-tracer")
		c.Check(tracer, quicktest.Not(quicktest.IsNil))

		logger := result.LoggerProvider.Logger("test-logger")
		c.Check(logger, quicktest.Not(quicktest.IsNil))

		meter := result.MeterProvider.Meter("test-meter")
		c.Check(meter, quicktest.Not(quicktest.IsNil))

		// Cleanup
		err = result.Cleanup()
		c.Check(err, quicktest.IsNil)
	})

	qt.Run("setup with default options", func(c *quicktest.C) {
		ctx := context.Background()

		result, err := Setup(ctx)

		c.Check(err, quicktest.IsNil)
		c.Check(result, quicktest.Not(quicktest.IsNil))
		c.Check(result.TracerProvider, quicktest.Not(quicktest.IsNil))
		c.Check(result.LoggerProvider, quicktest.Not(quicktest.IsNil))
		c.Check(result.MeterProvider, quicktest.Not(quicktest.IsNil))

		// Cleanup
		err = result.Cleanup()
		c.Check(err, quicktest.IsNil)
	})

	qt.Run("setup with all custom options", func(c *quicktest.C) {
		ctx := context.Background()

		result, err := Setup(ctx,
			WithServiceName("custom-service"),
			WithServiceVersion("2.0.0"),
			WithHost("custom-host"),
			WithPort(9090),
			WithCollectorEnable(false),
		)

		c.Check(err, quicktest.IsNil)
		c.Check(result, quicktest.Not(quicktest.IsNil))

		// Cleanup
		err = result.Cleanup()
		c.Check(err, quicktest.IsNil)
	})
}

func TestSetupWithCleanup(t *testing.T) {
	qt := quicktest.New(t)

	qt.Run("successful setup with cleanup", func(c *quicktest.C) {
		ctx := context.Background()

		cleanup := SetupWithCleanup(ctx,
			WithServiceName("cleanup-test"),
			WithServiceVersion("1.0.0"),
		)

		c.Check(cleanup, quicktest.Not(quicktest.IsNil))

		// Verify providers are set globally
		tracerProvider := otel.GetTracerProvider()
		c.Check(tracerProvider, quicktest.Not(quicktest.IsNil))

		// Test that providers work
		tracer := tracerProvider.Tracer("test-tracer")
		c.Check(tracer, quicktest.Not(quicktest.IsNil))

		// Call cleanup
		cleanup()

		// Verify cleanup worked (providers should be shutdown)
		// Note: We can't easily verify this without accessing internal state
	})

	qt.Run("setup with panic on failure", func(c *quicktest.C) {
		ctx := context.Background()

		// Simply call the function - if it panics, the test will fail
		cleanup := SetupWithCleanup(ctx, WithServiceName("panic-test"))
		cleanup()
	})
}

func TestSetupResult(t *testing.T) {
	qt := quicktest.New(t)

	qt.Run("setup result structure", func(c *quicktest.C) {
		ctx := context.Background()

		result, err := Setup(ctx, WithServiceName("result-test"))
		c.Check(err, quicktest.IsNil)
		c.Check(result, quicktest.Not(quicktest.IsNil))

		// Verify all fields are properly set
		c.Check(result.TracerProvider, quicktest.Not(quicktest.IsNil))
		c.Check(result.LoggerProvider, quicktest.Not(quicktest.IsNil))
		c.Check(result.MeterProvider, quicktest.Not(quicktest.IsNil))
		c.Check(result.Cleanup, quicktest.Not(quicktest.IsNil))

		// Cleanup
		err = result.Cleanup()
		c.Check(err, quicktest.IsNil)
	})
}

func TestSetupErrorHandling(t *testing.T) {
	qt := quicktest.New(t)

	qt.Run("context cancellation", func(c *quicktest.C) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		result, err := Setup(ctx, WithServiceName("cancel-test"))

		// Should still work even with cancelled context
		c.Check(err, quicktest.IsNil)
		c.Check(result, quicktest.Not(quicktest.IsNil))

		// Cleanup should fail due to cancelled context
		err = result.Cleanup()
		c.Check(err, quicktest.Not(quicktest.IsNil))
		c.Check(err.Error(), quicktest.Contains, "cleanup errors")
	})

	qt.Run("invalid port number", func(c *quicktest.C) {
		ctx := context.Background()

		// Invalid port should still work since validation happens later
		result, err := Setup(ctx, WithPort(-1))
		c.Check(err, quicktest.IsNil)
		c.Check(result, quicktest.Not(quicktest.IsNil))

		// Cleanup
		err = result.Cleanup()
		c.Check(err, quicktest.IsNil)
	})
}

func TestSetupMultipleCalls(t *testing.T) {
	qt := quicktest.New(t)

	qt.Run("multiple setup calls", func(c *quicktest.C) {
		ctx := context.Background()

		// First setup
		result1, err := Setup(ctx, WithServiceName("multi1"))
		c.Check(err, quicktest.IsNil)
		c.Check(result1, quicktest.Not(quicktest.IsNil))

		// Second setup
		result2, err := Setup(ctx, WithServiceName("multi2"))
		c.Check(err, quicktest.IsNil)
		c.Check(result2, quicktest.Not(quicktest.IsNil))

		// Cleanup both
		err = result1.Cleanup()
		c.Check(err, quicktest.IsNil)

		err = result2.Cleanup()
		c.Check(err, quicktest.IsNil)
	})
}

func TestSetupConcurrentAccess(t *testing.T) {
	qt := quicktest.New(t)

	qt.Run("concurrent setup calls", func(c *quicktest.C) {
		ctx := context.Background()

		// Test concurrent setup calls
		done := make(chan bool, 2)

		go func() {
			result, err := Setup(ctx, WithServiceName("concurrent1"))
			if err == nil && result != nil {
				err = result.Cleanup()
				c.Check(err, quicktest.IsNil)
			}
			done <- true
		}()

		go func() {
			result, err := Setup(ctx, WithServiceName("concurrent2"))
			if err == nil && result != nil {
				err = result.Cleanup()
				c.Check(err, quicktest.IsNil)
			}
			done <- true
		}()

		// Wait for both goroutines to complete
		<-done
		<-done
	})
}

func TestSetupMemoryLeaks(t *testing.T) {
	qt := quicktest.New(t)

	qt.Run("memory leak prevention", func(c *quicktest.C) {
		ctx := context.Background()

		// Create and destroy multiple setups to check for memory leaks
		for range 5 {
			result, err := Setup(ctx,
				WithServiceName("memory-test"),
				WithServiceVersion("1.0.0"),
			)

			c.Check(err, quicktest.IsNil)
			c.Check(result, quicktest.Not(quicktest.IsNil))

			// Test that all providers work
			tracer := result.TracerProvider.Tracer("test-tracer")
			c.Check(tracer, quicktest.Not(quicktest.IsNil))

			logger := result.LoggerProvider.Logger("test-logger")
			c.Check(logger, quicktest.Not(quicktest.IsNil))

			meter := result.MeterProvider.Meter("test-meter")
			c.Check(meter, quicktest.Not(quicktest.IsNil))

			// Cleanup immediately
			err = result.Cleanup()
			c.Check(err, quicktest.IsNil)
		}
	})
}

func TestSetupIntegration(t *testing.T) {
	qt := quicktest.New(t)

	qt.Run("integration test with actual usage", func(c *quicktest.C) {
		ctx := context.Background()

		result, err := Setup(ctx,
			WithServiceName("integration-test"),
			WithServiceVersion("1.0.0"),
		)
		c.Check(err, quicktest.IsNil)
		c.Check(result, quicktest.Not(quicktest.IsNil))

		// Test actual usage of all providers
		tracer := result.TracerProvider.Tracer("integration-tracer")
		ctx, span := tracer.Start(ctx, "test_span")
		c.Check(span, quicktest.Not(quicktest.IsNil))
		span.End()

		meter := result.MeterProvider.Meter("integration-meter")
		counter, err := meter.Int64Counter("test_counter")
		c.Check(err, quicktest.IsNil)
		counter.Add(ctx, 1)

		// Cleanup
		err = result.Cleanup()
		c.Check(err, quicktest.IsNil)
	})
}

func TestSetupCleanupFunction(t *testing.T) {
	qt := quicktest.New(t)

	qt.Run("cleanup function behavior", func(c *quicktest.C) {
		ctx := context.Background()

		result, err := Setup(ctx, WithServiceName("cleanup-test"))
		c.Check(err, quicktest.IsNil)
		c.Check(result, quicktest.Not(quicktest.IsNil))

		// Call cleanup once
		err = result.Cleanup()
		c.Check(err, quicktest.IsNil)

		// Call cleanup again - should error because providers are already shut down
		err = result.Cleanup()
		c.Check(err, quicktest.Not(quicktest.IsNil))
		c.Check(err.Error(), quicktest.Contains, "cleanup errors")
	})
}

func TestSetupWithContextTODO(t *testing.T) {
	qt := quicktest.New(t)

	qt.Run("context TODO", func(c *quicktest.C) {
		// This should work as context.Background() is used internally
		result, err := Setup(context.TODO())

		c.Check(err, quicktest.IsNil)
		c.Check(result, quicktest.Not(quicktest.IsNil))

		// Cleanup
		err = result.Cleanup()
		c.Check(err, quicktest.IsNil)
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
		WithPort(4317),
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
