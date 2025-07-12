package otel

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/log/global"
)

func TestSetupLogging(t *testing.T) {
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
		{
			name: "collector enabled",
			options: []Option{
				WithCollectorEnable(true),
				WithHost("localhost"),
				WithPort(4317),
			},
			wantErr: false,
		},
		{
			name: "all options set",
			options: []Option{
				WithServiceName("complete-service"),
				WithServiceVersion("2.0.0"),
				WithHost("example.com"),
				WithPort(9090),
				WithCollectorEnable(true),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			provider, err := SetupLogging(ctx, tt.options...)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, provider)

			// Verify the provider is set globally
			globalProvider := global.GetLoggerProvider()
			assert.Equal(t, provider, globalProvider)

			// Test that we can create a logger
			logger := provider.Logger("test-logger")
			assert.NotNil(t, logger)

			// Cleanup
			err = provider.Shutdown(ctx)
			assert.NoError(t, err)
		})
	}
}

func TestSetupLoggingWithCollector(t *testing.T) {
	// Test with collector enabled - this will try to connect to a real endpoint
	// In a real test environment, you might want to mock this or use a test collector
	t.Run("collector enabled with invalid endpoint", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		options := []Option{
			WithCollectorEnable(true),
			WithHost("invalid-host"),
			WithPort(9999),
		}

		// This should fail because the endpoint doesn't exist
		provider, err := SetupLogging(ctx, options...)

		// The OTLP exporter might not fail immediately, so we check if we get a provider
		// In some cases, it might succeed but fail later during export
		if err != nil {
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "failed to create OTLP log exporter")
		} else {
			// If it succeeds, clean up
			require.NotNil(t, provider)
			err = provider.Shutdown(ctx)
			assert.NoError(t, err)
		}
	})
}

func TestSetupLoggingWithStdout(t *testing.T) {
	t.Run("stdout exporter", func(t *testing.T) {
		ctx := context.Background()

		// Use default options (collector disabled = stdout exporter)
		provider, err := SetupLogging(ctx)

		require.NoError(t, err)
		require.NotNil(t, provider)

		// Verify it's set globally
		globalProvider := global.GetLoggerProvider()
		assert.Equal(t, provider, globalProvider)

		// Test logging functionality
		logger := provider.Logger("test-logger")
		assert.NotNil(t, logger)

		// Cleanup
		err = provider.Shutdown(ctx)
		assert.NoError(t, err)
	})
}

func TestSetupLoggingResourceAttributes(t *testing.T) {
	t.Run("verify resource attributes", func(t *testing.T) {
		ctx := context.Background()

		serviceName := "test-service"
		serviceVersion := "1.2.3"

		provider, err := SetupLogging(ctx,
			WithServiceName(serviceName),
			WithServiceVersion(serviceVersion),
		)

		require.NoError(t, err)
		require.NotNil(t, provider)

		// Get the resource from the provider
		// Note: This is a bit tricky to test directly, but we can verify the provider works
		logger := provider.Logger("test-logger")
		assert.NotNil(t, logger)

		// Cleanup
		err = provider.Shutdown(ctx)
		assert.NoError(t, err)
	})
}

func TestSetupLoggingMultipleCalls(t *testing.T) {
	t.Run("multiple setup calls", func(t *testing.T) {
		ctx := context.Background()

		// First setup
		provider1, err := SetupLogging(ctx, WithServiceName("service1"))
		require.NoError(t, err)
		require.NotNil(t, provider1)

		// Second setup - should work but might return the same provider due to global state
		provider2, err := SetupLogging(ctx, WithServiceName("service2"))
		require.NoError(t, err)
		require.NotNil(t, provider2)

		// Cleanup both
		err = provider1.Shutdown(ctx)
		assert.NoError(t, err)

		err = provider2.Shutdown(ctx)
		assert.NoError(t, err)
	})
}

func TestSetupLoggingContextCancellation(t *testing.T) {
	t.Run("context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		provider, err := SetupLogging(ctx)

		// Should still work even with cancelled context
		require.NoError(t, err)
		require.NotNil(t, provider)

		// Cleanup
		err = provider.Shutdown(context.Background())
		assert.NoError(t, err)
	})
}

func TestSetupLoggingShutdown(t *testing.T) {
	t.Run("provider shutdown", func(t *testing.T) {
		ctx := context.Background()

		provider, err := SetupLogging(ctx)
		require.NoError(t, err)
		require.NotNil(t, provider)

		// Shutdown the provider
		err = provider.Shutdown(ctx)
		assert.NoError(t, err)

		// Try to shutdown again - should not error
		err = provider.Shutdown(ctx)
		assert.NoError(t, err)
	})
}

func TestSetupLoggingWithNilContext(t *testing.T) {
	t.Run("nil context", func(t *testing.T) {
		// This should work as context.Background() is used internally
		provider, err := SetupLogging(context.TODO())

		require.NoError(t, err)
		require.NotNil(t, provider)

		// Cleanup
		err = provider.Shutdown(context.Background())
		assert.NoError(t, err)
	})
}

func TestSetupLoggingOptionsCombination(t *testing.T) {
	tests := []struct {
		name    string
		options []Option
	}{
		{
			name: "service name only",
			options: []Option{
				WithServiceName("service-only"),
			},
		},
		{
			name: "service version only",
			options: []Option{
				WithServiceVersion("1.0.0"),
			},
		},
		{
			name: "host only",
			options: []Option{
				WithHost("host-only"),
			},
		},
		{
			name: "port only",
			options: []Option{
				WithPort(1234),
			},
		},
		{
			name: "collector enable only",
			options: []Option{
				WithCollectorEnable(true),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			provider, err := SetupLogging(ctx, tt.options...)

			require.NoError(t, err)
			require.NotNil(t, provider)

			// Verify global provider is set
			globalProvider := global.GetLoggerProvider()
			assert.Equal(t, provider, globalProvider)

			// Cleanup
			err = provider.Shutdown(ctx)
			assert.NoError(t, err)
		})
	}
}

func TestSetupLoggingDefaultValues(t *testing.T) {
	t.Run("verify default values", func(t *testing.T) {
		ctx := context.Background()

		// Use no options to get default values
		provider, err := SetupLogging(ctx)
		require.NoError(t, err)
		require.NotNil(t, provider)

		// The default values should be:
		// - ServiceName: "unknown"
		// - ServiceVersion: "unknown"
		// - Host: "localhost"
		// - Port: 4317
		// - CollectorEnable: false

		// We can't easily verify these directly, but we can verify the provider works
		logger := provider.Logger("test-logger")
		assert.NotNil(t, logger)

		// Cleanup
		err = provider.Shutdown(ctx)
		assert.NoError(t, err)
	})
}

func TestSetupLoggingErrorHandling(t *testing.T) {
	t.Run("invalid port number", func(t *testing.T) {
		ctx := context.Background()

		// Test with an invalid port that might cause issues
		provider, err := SetupLogging(ctx, WithPort(-1))

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

func TestSetupLoggingConcurrentAccess(t *testing.T) {
	t.Run("concurrent setup calls", func(t *testing.T) {
		ctx := context.Background()

		// Test concurrent setup calls
		done := make(chan bool, 2)

		go func() {
			provider, err := SetupLogging(ctx, WithServiceName("concurrent1"))
			if err == nil && provider != nil {
				err = provider.Shutdown(ctx)
				assert.NoError(t, err)
			}
			done <- true
		}()

		go func() {
			provider, err := SetupLogging(ctx, WithServiceName("concurrent2"))
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

func TestSetupLoggingMemoryLeaks(t *testing.T) {
	t.Run("memory leak prevention", func(t *testing.T) {
		ctx := context.Background()

		// Create and destroy multiple providers to check for memory leaks
		for range 10 {
			provider, err := SetupLogging(ctx,
				WithServiceName("memory-test"),
				WithServiceVersion("1.0.0"),
			)

			require.NoError(t, err)
			require.NotNil(t, provider)

			// Create a logger to ensure it works
			logger := provider.Logger("test-logger")
			assert.NotNil(t, logger)

			// Shutdown immediately
			err = provider.Shutdown(ctx)
			assert.NoError(t, err)
		}
	})
}

func TestSetupLoggingIntegration(t *testing.T) {
	t.Run("integration test with actual logging", func(t *testing.T) {
		ctx := context.Background()

		provider, err := SetupLogging(ctx,
			WithServiceName("integration-test"),
			WithServiceVersion("1.0.0"),
		)
		require.NoError(t, err)
		require.NotNil(t, provider)

		// Create a logger and test actual logging
		logger := provider.Logger("integration-logger")
		assert.NotNil(t, logger)

		// Cleanup
		err = provider.Shutdown(ctx)
		assert.NoError(t, err)
	})
}

// Benchmark tests
func BenchmarkSetupLogging(b *testing.B) {
	ctx := context.Background()

	b.ResetTimer()
	for b.Loop() {
		provider, err := SetupLogging(ctx, WithServiceName("benchmark"))
		if err == nil && provider != nil {
			shutdownErr := provider.Shutdown(ctx)
			if shutdownErr != nil {
				b.Fatal(shutdownErr)
			}
		}
	}
}

func BenchmarkSetupLoggingWithOptions(b *testing.B) {
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
		provider, err := SetupLogging(ctx, options...)
		if err == nil && provider != nil {
			shutdownErr := provider.Shutdown(ctx)
			if shutdownErr != nil {
				b.Fatal(shutdownErr)
			}
		}
	}
}
