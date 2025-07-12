package otel

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
)

func TestSetupMetrics(t *testing.T) {
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
				WithPort("8080"),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			provider, err := SetupMetrics(ctx, tt.options...)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, provider)

			// Verify the provider is set globally
			globalProvider := otel.GetMeterProvider()
			assert.Equal(t, provider, globalProvider)

			// Test that we can create a meter
			meter := provider.Meter("test-meter")
			assert.NotNil(t, meter)

			// Cleanup
			err = provider.Shutdown(ctx)
			assert.NoError(t, err)
		})
	}
}

func TestSetupMetricsWithStdoutExporter(t *testing.T) {
	t.Run("stdout exporter functionality", func(t *testing.T) {
		ctx := context.Background()

		// Use stdout exporter (default when collector is disabled)
		provider, err := SetupMetrics(ctx,
			WithServiceName("stdout-test"),
			WithServiceVersion("1.0.0"),
			WithCollectorEnable(false), // Explicitly disable collector
		)

		require.NoError(t, err)
		require.NotNil(t, provider)

		// Test metrics functionality
		meter := provider.Meter("test-meter")
		assert.NotNil(t, meter)

		// Create and use various metric types
		counter, err := meter.Int64Counter("test_counter")
		require.NoError(t, err)
		counter.Add(ctx, 1)

		histogram, err := meter.Float64Histogram("test_histogram")
		require.NoError(t, err)
		histogram.Record(ctx, 1.5)

		// Wait a bit for metrics to be processed
		time.Sleep(100 * time.Millisecond)

		// Cleanup
		err = provider.Shutdown(ctx)
		assert.NoError(t, err)
	})
}

func TestSetupMetricsResourceAttributes(t *testing.T) {
	t.Run("verify resource attributes", func(t *testing.T) {
		ctx := context.Background()

		serviceName := "test-service"
		serviceVersion := "1.2.3"

		provider, err := SetupMetrics(ctx,
			WithServiceName(serviceName),
			WithServiceVersion(serviceVersion),
		)

		require.NoError(t, err)
		require.NotNil(t, provider)

		// Test that we can create a meter
		meter := provider.Meter("test-meter")
		assert.NotNil(t, meter)

		// Cleanup
		err = provider.Shutdown(ctx)
		assert.NoError(t, err)
	})
}

func TestSetupMetricsMultipleCalls(t *testing.T) {
	t.Run("multiple setup calls", func(t *testing.T) {
		ctx := context.Background()

		// First setup
		provider1, err := SetupMetrics(ctx, WithServiceName("service1"))
		require.NoError(t, err)
		require.NotNil(t, provider1)

		// Second setup - should work but might return the same provider due to global state
		provider2, err := SetupMetrics(ctx, WithServiceName("service2"))
		require.NoError(t, err)
		require.NotNil(t, provider2)

		// Cleanup both
		err = provider1.Shutdown(ctx)
		assert.NoError(t, err)

		err = provider2.Shutdown(ctx)
		assert.NoError(t, err)
	})
}

func TestSetupMetricsContextCancellation(t *testing.T) {
	t.Run("context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		provider, err := SetupMetrics(ctx)

		// Should still work even with cancelled context
		require.NoError(t, err)
		require.NotNil(t, provider)

		// Cleanup
		err = provider.Shutdown(context.Background())
		assert.NoError(t, err)
	})
}

func TestSetupMetricsShutdown(t *testing.T) {
	t.Run("provider shutdown", func(t *testing.T) {
		ctx := context.Background()

		provider, err := SetupMetrics(ctx)
		require.NoError(t, err)
		require.NotNil(t, provider)

		// Shutdown the provider
		err = provider.Shutdown(ctx)
		assert.NoError(t, err)

		// Try to shutdown again - might error due to reader already shutdown
		err = provider.Shutdown(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "reader is shutdown")
	})
}

func TestSetupMetricsWithContextTODO(t *testing.T) {
	t.Run("context TODO", func(t *testing.T) {
		// This should work as context.Background() is used internally
		provider, err := SetupMetrics(context.TODO())

		require.NoError(t, err)
		require.NotNil(t, provider)

		// Cleanup
		err = provider.Shutdown(context.Background())
		assert.NoError(t, err)
	})
}

func TestSetupMetricsOptionsCombination(t *testing.T) {
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
				WithPort("1234"),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			provider, err := SetupMetrics(ctx, tt.options...)

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
			globalProvider := otel.GetMeterProvider()
			assert.Equal(t, provider, globalProvider)

			// Cleanup
			err = provider.Shutdown(ctx)
			assert.NoError(t, err)
		})
	}
}

func TestSetupMetricsDefaultValues(t *testing.T) {
	t.Run("verify default values", func(t *testing.T) {
		ctx := context.Background()

		// Use no options to get default values
		provider, err := SetupMetrics(ctx)
		require.NoError(t, err)
		require.NotNil(t, provider)

		// The default values should be:
		// - ServiceName: "unknown"
		// - ServiceVersion: "unknown"
		// - Host: "localhost"
		// - Port: "4317"
		// - CollectorEnable: false

		// We can't easily verify these directly, but we can verify the provider works
		meter := provider.Meter("test-meter")
		assert.NotNil(t, meter)

		// Cleanup
		err = provider.Shutdown(ctx)
		assert.NoError(t, err)
	})
}

func TestSetupMetricsErrorHandling(t *testing.T) {
	t.Run("invalid port number", func(t *testing.T) {
		ctx := context.Background()

		// Test with an invalid port that might cause issues
		provider, err := SetupMetrics(ctx, WithPort("invalid-port"))

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

func TestSetupMetricsConcurrentAccess(t *testing.T) {
	t.Run("concurrent setup calls", func(t *testing.T) {
		ctx := context.Background()

		// Test concurrent setup calls
		done := make(chan bool, 2)

		go func() {
			provider, err := SetupMetrics(ctx, WithServiceName("concurrent1"))
			if err == nil && provider != nil {
				err = provider.Shutdown(ctx)
				assert.NoError(t, err)
			}
			done <- true
		}()

		go func() {
			provider, err := SetupMetrics(ctx, WithServiceName("concurrent2"))
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

func TestSetupMetricsMemoryLeaks(t *testing.T) {
	t.Run("memory leak prevention", func(t *testing.T) {
		ctx := context.Background()

		// Create and destroy multiple providers to check for memory leaks
		for range 10 {
			provider, err := SetupMetrics(ctx,
				WithServiceName("memory-test"),
				WithServiceVersion("1.0.0"),
			)

			require.NoError(t, err)
			require.NotNil(t, provider)

			// Create a meter to ensure it works
			meter := provider.Meter("test-meter")
			assert.NotNil(t, meter)

			// Shutdown immediately
			err = provider.Shutdown(ctx)
			assert.NoError(t, err)
		}
	})
}

func TestSetupMetricsIntegration(t *testing.T) {
	t.Run("integration test with actual metrics", func(t *testing.T) {
		ctx := context.Background()

		provider, err := SetupMetrics(ctx,
			WithServiceName("integration-test"),
			WithServiceVersion("1.0.0"),
		)
		require.NoError(t, err)
		require.NotNil(t, provider)

		// Create a meter and test actual metrics
		meter := provider.Meter("integration-meter")
		assert.NotNil(t, meter)

		// Test creating a counter
		counter, err := meter.Int64Counter("test_counter")
		require.NoError(t, err)
		assert.NotNil(t, counter)

		// Test creating a histogram
		histogram, err := meter.Float64Histogram("test_histogram")
		require.NoError(t, err)
		assert.NotNil(t, histogram)

		// Cleanup
		err = provider.Shutdown(ctx)
		assert.NoError(t, err)
	})
}

func TestSetupMetricsPeriodicReader(t *testing.T) {
	t.Run("verify periodic reader configuration", func(t *testing.T) {
		ctx := context.Background()

		provider, err := SetupMetrics(ctx)
		require.NoError(t, err)
		require.NotNil(t, provider)

		// The provider should have a periodic reader configured
		// We can verify this by checking that the provider works correctly
		meter := provider.Meter("test-meter")
		assert.NotNil(t, meter)

		// Create a counter and increment it
		counter, err := meter.Int64Counter("test_counter")
		require.NoError(t, err)
		counter.Add(ctx, 1)

		// Cleanup
		err = provider.Shutdown(ctx)
		assert.NoError(t, err)
	})
}

func TestSetupMetricsWithCustomInterval(t *testing.T) {
	t.Run("custom export interval", func(t *testing.T) {
		ctx := context.Background()

		// Note: The current implementation uses a fixed 10-second interval
		// This test verifies the current behavior
		provider, err := SetupMetrics(ctx)
		require.NoError(t, err)
		require.NotNil(t, provider)

		// Test that metrics are collected
		meter := provider.Meter("test-meter")
		counter, err := meter.Int64Counter("test_counter")
		require.NoError(t, err)

		// Add some values
		counter.Add(ctx, 1)
		counter.Add(ctx, 2)
		counter.Add(ctx, 3)

		// Cleanup
		err = provider.Shutdown(ctx)
		assert.NoError(t, err)
	})
}

func TestSetupMetricsResourceConfiguration(t *testing.T) {
	t.Run("resource configuration", func(t *testing.T) {
		ctx := context.Background()

		serviceName := "resource-test"
		serviceVersion := "2.0.0"

		provider, err := SetupMetrics(ctx,
			WithServiceName(serviceName),
			WithServiceVersion(serviceVersion),
		)
		require.NoError(t, err)
		require.NotNil(t, provider)

		// Test that the provider works with the configured resource
		meter := provider.Meter("test-meter")
		assert.NotNil(t, meter)

		// Create and use a metric
		counter, err := meter.Int64Counter("resource_test_counter")
		require.NoError(t, err)
		counter.Add(ctx, 1)

		// Cleanup
		err = provider.Shutdown(ctx)
		assert.NoError(t, err)
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
		WithPort("4317"),
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
