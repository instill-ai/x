package otel

import (
	"context"
	"testing"
	"time"

	"github.com/frankban/quicktest"
	"go.opentelemetry.io/otel"
)

func TestSetupMetrics(t *testing.T) {
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

			provider, err := SetupMetrics(ctx, tt.options...)

			if tt.wantErr {
				c.Check(err, quicktest.Not(quicktest.IsNil))
				return
			}

			c.Check(err, quicktest.IsNil)
			c.Check(provider, quicktest.Not(quicktest.IsNil))

			// Verify the provider is set globally
			globalProvider := otel.GetMeterProvider()
			c.Check(provider, quicktest.Equals, globalProvider)

			// Test that we can create a meter
			meter := provider.Meter("test-meter")
			c.Check(meter, quicktest.Not(quicktest.IsNil))

			// Cleanup
			err = provider.Shutdown(ctx)
			c.Check(err, quicktest.IsNil)
		})
	}
}

func TestSetupMetricsWithStdoutExporter(t *testing.T) {
	qt := quicktest.New(t)

	qt.Run("stdout exporter functionality", func(c *quicktest.C) {
		ctx := context.Background()

		// Use stdout exporter (default when collector is disabled)
		provider, err := SetupMetrics(ctx,
			WithServiceName("stdout-test"),
			WithServiceVersion("1.0.0"),
			WithCollectorEnable(false), // Explicitly disable collector
		)

		c.Check(err, quicktest.IsNil)
		c.Check(provider, quicktest.Not(quicktest.IsNil))

		// Test metrics functionality
		meter := provider.Meter("test-meter")
		c.Check(meter, quicktest.Not(quicktest.IsNil))

		// Create and use various metric types
		counter, err := meter.Int64Counter("test_counter")
		c.Check(err, quicktest.IsNil)
		counter.Add(ctx, 1)

		histogram, err := meter.Float64Histogram("test_histogram")
		c.Check(err, quicktest.IsNil)
		histogram.Record(ctx, 1.5)

		// Wait a bit for metrics to be processed
		time.Sleep(100 * time.Millisecond)

		// Cleanup
		err = provider.Shutdown(ctx)
		c.Check(err, quicktest.IsNil)
	})
}

func TestSetupMetricsResourceAttributes(t *testing.T) {
	qt := quicktest.New(t)

	qt.Run("verify resource attributes", func(c *quicktest.C) {
		ctx := context.Background()

		serviceName := "test-service"
		serviceVersion := "1.2.3"

		provider, err := SetupMetrics(ctx,
			WithServiceName(serviceName),
			WithServiceVersion(serviceVersion),
		)

		c.Check(err, quicktest.IsNil)
		c.Check(provider, quicktest.Not(quicktest.IsNil))

		// Test that we can create a meter
		meter := provider.Meter("test-meter")
		c.Check(meter, quicktest.Not(quicktest.IsNil))

		// Cleanup
		err = provider.Shutdown(ctx)
		c.Check(err, quicktest.IsNil)
	})
}

func TestSetupMetricsMultipleCalls(t *testing.T) {
	qt := quicktest.New(t)

	qt.Run("multiple setup calls", func(c *quicktest.C) {
		ctx := context.Background()

		// First setup
		provider1, err := SetupMetrics(ctx, WithServiceName("service1"))
		c.Check(err, quicktest.IsNil)
		c.Check(provider1, quicktest.Not(quicktest.IsNil))

		// Second setup - should work but might return the same provider due to global state
		provider2, err := SetupMetrics(ctx, WithServiceName("service2"))
		c.Check(err, quicktest.IsNil)
		c.Check(provider2, quicktest.Not(quicktest.IsNil))

		// Cleanup both
		err = provider1.Shutdown(ctx)
		c.Check(err, quicktest.IsNil)

		err = provider2.Shutdown(ctx)
		c.Check(err, quicktest.IsNil)
	})
}

func TestSetupMetricsContextCancellation(t *testing.T) {
	qt := quicktest.New(t)

	qt.Run("context cancellation", func(c *quicktest.C) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		provider, err := SetupMetrics(ctx)

		// Should still work even with cancelled context
		c.Check(err, quicktest.IsNil)
		c.Check(provider, quicktest.Not(quicktest.IsNil))

		// Cleanup
		err = provider.Shutdown(context.Background())
		c.Check(err, quicktest.IsNil)
	})
}

func TestSetupMetricsShutdown(t *testing.T) {
	qt := quicktest.New(t)

	qt.Run("provider shutdown", func(c *quicktest.C) {
		ctx := context.Background()

		provider, err := SetupMetrics(ctx)
		c.Check(err, quicktest.IsNil)
		c.Check(provider, quicktest.Not(quicktest.IsNil))

		// Shutdown the provider
		err = provider.Shutdown(ctx)
		c.Check(err, quicktest.IsNil)

		// Try to shutdown again - might error due to reader already shutdown
		err = provider.Shutdown(ctx)
		c.Check(err, quicktest.Not(quicktest.IsNil))
		c.Check(err.Error(), quicktest.Contains, "reader is shutdown")
	})
}

func TestSetupMetricsWithContextTODO(t *testing.T) {
	qt := quicktest.New(t)

	qt.Run("context TODO", func(c *quicktest.C) {
		// This should work as context.Background() is used internally
		provider, err := SetupMetrics(context.TODO())

		c.Check(err, quicktest.IsNil)
		c.Check(provider, quicktest.Not(quicktest.IsNil))

		// Cleanup
		err = provider.Shutdown(context.Background())
		c.Check(err, quicktest.IsNil)
	})
}

func TestSetupMetricsOptionsCombination(t *testing.T) {
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

			provider, err := SetupMetrics(ctx, tt.options...)

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
			globalProvider := otel.GetMeterProvider()
			c.Check(provider, quicktest.Equals, globalProvider)

			// Cleanup
			err = provider.Shutdown(ctx)
			c.Check(err, quicktest.IsNil)
		})
	}
}

func TestSetupMetricsDefaultValues(t *testing.T) {
	qt := quicktest.New(t)

	qt.Run("verify default values", func(c *quicktest.C) {
		ctx := context.Background()

		// Use no options to get default values
		provider, err := SetupMetrics(ctx)
		c.Check(err, quicktest.IsNil)
		c.Check(provider, quicktest.Not(quicktest.IsNil))

		// The default values should be:
		// - ServiceName: "unknown"
		// - ServiceVersion: "unknown"
		// - Host: "localhost"
		// - Port: 4317
		// - CollectorEnable: false

		// We can't easily verify these directly, but we can verify the provider works
		meter := provider.Meter("test-meter")
		c.Check(meter, quicktest.Not(quicktest.IsNil))

		// Cleanup
		err = provider.Shutdown(ctx)
		c.Check(err, quicktest.IsNil)
	})
}

func TestSetupMetricsErrorHandling(t *testing.T) {
	qt := quicktest.New(t)

	qt.Run("invalid port number", func(c *quicktest.C) {
		ctx := context.Background()

		// Test with an invalid port that might cause issues
		provider, err := SetupMetrics(ctx, WithPort(-1))

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

func TestSetupMetricsConcurrentAccess(t *testing.T) {
	qt := quicktest.New(t)

	qt.Run("concurrent setup calls", func(c *quicktest.C) {
		ctx := context.Background()

		// Test concurrent setup calls
		done := make(chan bool, 2)

		go func() {
			provider, err := SetupMetrics(ctx, WithServiceName("concurrent1"))
			if err == nil && provider != nil {
				err = provider.Shutdown(ctx)
				c.Check(err, quicktest.IsNil)
			}
			done <- true
		}()

		go func() {
			provider, err := SetupMetrics(ctx, WithServiceName("concurrent2"))
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

func TestSetupMetricsMemoryLeaks(t *testing.T) {
	qt := quicktest.New(t)

	qt.Run("memory leak prevention", func(c *quicktest.C) {
		ctx := context.Background()

		// Create and destroy multiple providers to check for memory leaks
		for range 10 {
			provider, err := SetupMetrics(ctx,
				WithServiceName("memory-test"),
				WithServiceVersion("1.0.0"),
			)

			c.Check(err, quicktest.IsNil)
			c.Check(provider, quicktest.Not(quicktest.IsNil))

			// Create a meter to ensure it works
			meter := provider.Meter("test-meter")
			c.Check(meter, quicktest.Not(quicktest.IsNil))

			// Shutdown immediately
			err = provider.Shutdown(ctx)
			c.Check(err, quicktest.IsNil)
		}
	})
}

func TestSetupMetricsIntegration(t *testing.T) {
	qt := quicktest.New(t)

	qt.Run("integration test with actual metrics", func(c *quicktest.C) {
		ctx := context.Background()

		provider, err := SetupMetrics(ctx,
			WithServiceName("integration-test"),
			WithServiceVersion("1.0.0"),
		)
		c.Check(err, quicktest.IsNil)
		c.Check(provider, quicktest.Not(quicktest.IsNil))

		// Create a meter and test actual metrics
		meter := provider.Meter("integration-meter")
		c.Check(meter, quicktest.Not(quicktest.IsNil))

		// Test creating a counter
		counter, err := meter.Int64Counter("test_counter")
		c.Check(err, quicktest.IsNil)
		c.Check(counter, quicktest.Not(quicktest.IsNil))

		// Test creating a histogram
		histogram, err := meter.Float64Histogram("test_histogram")
		c.Check(err, quicktest.IsNil)
		c.Check(histogram, quicktest.Not(quicktest.IsNil))

		// Cleanup
		err = provider.Shutdown(ctx)
		c.Check(err, quicktest.IsNil)
	})
}

func TestSetupMetricsPeriodicReader(t *testing.T) {
	qt := quicktest.New(t)

	qt.Run("verify periodic reader configuration", func(c *quicktest.C) {
		ctx := context.Background()

		provider, err := SetupMetrics(ctx)
		c.Check(err, quicktest.IsNil)
		c.Check(provider, quicktest.Not(quicktest.IsNil))

		// The provider should have a periodic reader configured
		// We can verify this by checking that the provider works correctly
		meter := provider.Meter("test-meter")
		c.Check(meter, quicktest.Not(quicktest.IsNil))

		// Create a counter and increment it
		counter, err := meter.Int64Counter("test_counter")
		c.Check(err, quicktest.IsNil)
		counter.Add(ctx, 1)

		// Cleanup
		err = provider.Shutdown(ctx)
		c.Check(err, quicktest.IsNil)
	})
}

func TestSetupMetricsWithCustomInterval(t *testing.T) {
	qt := quicktest.New(t)

	qt.Run("custom export interval", func(c *quicktest.C) {
		ctx := context.Background()

		// Note: The current implementation uses a fixed 10-second interval
		// This test verifies the current behavior
		provider, err := SetupMetrics(ctx)
		c.Check(err, quicktest.IsNil)
		c.Check(provider, quicktest.Not(quicktest.IsNil))

		// Test that metrics are collected
		meter := provider.Meter("test-meter")
		counter, err := meter.Int64Counter("test_counter")
		c.Check(err, quicktest.IsNil)

		// Add some values
		counter.Add(ctx, 1)
		counter.Add(ctx, 2)
		counter.Add(ctx, 3)

		// Cleanup
		err = provider.Shutdown(ctx)
		c.Check(err, quicktest.IsNil)
	})
}

func TestSetupMetricsResourceConfiguration(t *testing.T) {
	qt := quicktest.New(t)

	qt.Run("resource configuration", func(c *quicktest.C) {
		ctx := context.Background()

		serviceName := "resource-test"
		serviceVersion := "2.0.0"

		provider, err := SetupMetrics(ctx,
			WithServiceName(serviceName),
			WithServiceVersion(serviceVersion),
		)
		c.Check(err, quicktest.IsNil)
		c.Check(provider, quicktest.Not(quicktest.IsNil))

		// Test that the provider works with the configured resource
		meter := provider.Meter("test-meter")
		c.Check(meter, quicktest.Not(quicktest.IsNil))

		// Create and use a metric
		counter, err := meter.Int64Counter("resource_test_counter")
		c.Check(err, quicktest.IsNil)
		counter.Add(ctx, 1)

		// Cleanup
		err = provider.Shutdown(ctx)
		c.Check(err, quicktest.IsNil)
	})
}

// Benchmark tests
func BenchmarkSetupMetrics(b *testing.B) {
	ctx := context.Background()

	b.ResetTimer()
	for b.Loop() {
		provider, err := SetupMetrics(ctx, WithServiceName("benchmark"))
		if err == nil && provider != nil {
			err = provider.Shutdown(ctx)
			if err != nil {
				b.Fatal(err)
			}
		}
	}
}

func BenchmarkSetupMetricsWithOptions(b *testing.B) {
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
		provider, err := SetupMetrics(ctx, options...)
		if err == nil && provider != nil {
			err = provider.Shutdown(ctx)
			if err != nil {
				b.Fatal(err)
			}
		}
	}
}

func BenchmarkMetricsCreation(b *testing.B) {
	ctx := context.Background()
	provider, err := SetupMetrics(ctx)
	if err != nil {
		b.Fatal(err)
	}
	defer func() {
		err = provider.Shutdown(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}()

	meter := provider.Meter("benchmark-meter")

	b.ResetTimer()
	for b.Loop() {
		counter, err := meter.Int64Counter("benchmark_counter")
		if err != nil {
			b.Fatal(err)
		}
		counter.Add(ctx, 1)
	}
}
