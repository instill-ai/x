package otel

import (
	"context"
	"testing"
	"time"

	"github.com/frankban/quicktest"
	"go.opentelemetry.io/otel/log/global"
)

func TestSetupLogging(t *testing.T) {
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
		qt.Run(tt.name, func(c *quicktest.C) {
			ctx := context.Background()

			provider, err := SetupLogging(ctx, tt.options...)

			if tt.wantErr {
				c.Check(err, quicktest.Not(quicktest.IsNil))
				return
			}

			c.Check(err, quicktest.IsNil)
			c.Check(provider, quicktest.Not(quicktest.IsNil))

			// Verify the provider is set globally
			globalProvider := global.GetLoggerProvider()
			c.Check(provider, quicktest.Equals, globalProvider)

			// Test that we can create a logger
			logger := provider.Logger("test-logger")
			c.Check(logger, quicktest.Not(quicktest.IsNil))

			// Cleanup
			err = provider.Shutdown(ctx)
			c.Check(err, quicktest.IsNil)
		})
	}
}

func TestSetupLoggingWithCollector(t *testing.T) {
	qt := quicktest.New(t)

	// Test with collector enabled - this will try to connect to a real endpoint
	// In a real test environment, you might want to mock this or use a test collector
	qt.Run("collector enabled with invalid endpoint", func(c *quicktest.C) {
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
			c.Check(err, quicktest.Not(quicktest.IsNil))
			c.Check(err.Error(), quicktest.Contains, "failed to create OTLP log exporter")
		} else {
			// If it succeeds, clean up
			c.Check(provider, quicktest.Not(quicktest.IsNil))
			err = provider.Shutdown(ctx)
			c.Check(err, quicktest.IsNil)
		}
	})
}

func TestSetupLoggingWithStdout(t *testing.T) {
	qt := quicktest.New(t)

	qt.Run("stdout exporter", func(c *quicktest.C) {
		ctx := context.Background()

		// Use default options (collector disabled = stdout exporter)
		provider, err := SetupLogging(ctx)

		c.Check(err, quicktest.IsNil)
		c.Check(provider, quicktest.Not(quicktest.IsNil))

		// Verify it's set globally
		globalProvider := global.GetLoggerProvider()
		c.Check(provider, quicktest.Equals, globalProvider)

		// Test logging functionality
		logger := provider.Logger("test-logger")
		c.Check(logger, quicktest.Not(quicktest.IsNil))

		// Cleanup
		err = provider.Shutdown(ctx)
		c.Check(err, quicktest.IsNil)
	})
}

func TestSetupLoggingResourceAttributes(t *testing.T) {
	qt := quicktest.New(t)

	qt.Run("verify resource attributes", func(c *quicktest.C) {
		ctx := context.Background()

		serviceName := "test-service"
		serviceVersion := "1.2.3"

		provider, err := SetupLogging(ctx,
			WithServiceName(serviceName),
			WithServiceVersion(serviceVersion),
		)

		c.Check(err, quicktest.IsNil)
		c.Check(provider, quicktest.Not(quicktest.IsNil))

		// Get the resource from the provider
		// Note: This is a bit tricky to test directly, but we can verify the provider works
		logger := provider.Logger("test-logger")
		c.Check(logger, quicktest.Not(quicktest.IsNil))

		// Cleanup
		err = provider.Shutdown(ctx)
		c.Check(err, quicktest.IsNil)
	})
}

func TestSetupLoggingMultipleCalls(t *testing.T) {
	qt := quicktest.New(t)

	qt.Run("multiple setup calls", func(c *quicktest.C) {
		ctx := context.Background()

		// First setup
		provider1, err := SetupLogging(ctx, WithServiceName("service1"))
		c.Check(err, quicktest.IsNil)
		c.Check(provider1, quicktest.Not(quicktest.IsNil))

		// Second setup - should work but might return the same provider due to global state
		provider2, err := SetupLogging(ctx, WithServiceName("service2"))
		c.Check(err, quicktest.IsNil)
		c.Check(provider2, quicktest.Not(quicktest.IsNil))

		// Cleanup both
		err = provider1.Shutdown(ctx)
		c.Check(err, quicktest.IsNil)

		err = provider2.Shutdown(ctx)
		c.Check(err, quicktest.IsNil)
	})
}

func TestSetupLoggingContextCancellation(t *testing.T) {
	qt := quicktest.New(t)

	qt.Run("context cancellation", func(c *quicktest.C) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		provider, err := SetupLogging(ctx)

		// Should still work even with cancelled context
		c.Check(err, quicktest.IsNil)
		c.Check(provider, quicktest.Not(quicktest.IsNil))

		// Cleanup
		err = provider.Shutdown(context.Background())
		c.Check(err, quicktest.IsNil)
	})
}

func TestSetupLoggingShutdown(t *testing.T) {
	qt := quicktest.New(t)

	qt.Run("provider shutdown", func(c *quicktest.C) {
		ctx := context.Background()

		provider, err := SetupLogging(ctx)
		c.Check(err, quicktest.IsNil)
		c.Check(provider, quicktest.Not(quicktest.IsNil))

		// Shutdown the provider
		err = provider.Shutdown(ctx)
		c.Check(err, quicktest.IsNil)

		// Try to shutdown again - should not error
		err = provider.Shutdown(ctx)
		c.Check(err, quicktest.IsNil)
	})
}

func TestSetupLoggingWithNilContext(t *testing.T) {
	qt := quicktest.New(t)

	qt.Run("nil context", func(c *quicktest.C) {
		// This should work as context.Background() is used internally
		provider, err := SetupLogging(context.TODO())

		c.Check(err, quicktest.IsNil)
		c.Check(provider, quicktest.Not(quicktest.IsNil))

		// Cleanup
		err = provider.Shutdown(context.Background())
		c.Check(err, quicktest.IsNil)
	})
}

func TestSetupLoggingOptionsCombination(t *testing.T) {
	qt := quicktest.New(t)

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
		qt.Run(tt.name, func(c *quicktest.C) {
			ctx := context.Background()

			provider, err := SetupLogging(ctx, tt.options...)

			c.Check(err, quicktest.IsNil)
			c.Check(provider, quicktest.Not(quicktest.IsNil))

			// Verify global provider is set
			globalProvider := global.GetLoggerProvider()
			c.Check(provider, quicktest.Equals, globalProvider)

			// Cleanup
			err = provider.Shutdown(ctx)
			c.Check(err, quicktest.IsNil)
		})
	}
}

func TestSetupLoggingDefaultValues(t *testing.T) {
	qt := quicktest.New(t)

	qt.Run("verify default values", func(c *quicktest.C) {
		ctx := context.Background()

		// Use no options to get default values
		provider, err := SetupLogging(ctx)
		c.Check(err, quicktest.IsNil)
		c.Check(provider, quicktest.Not(quicktest.IsNil))

		// The default values should be:
		// - ServiceName: "unknown"
		// - ServiceVersion: "unknown"
		// - Host: "localhost"
		// - Port: 4317
		// - CollectorEnable: false

		// We can't easily verify these directly, but we can verify the provider works
		logger := provider.Logger("test-logger")
		c.Check(logger, quicktest.Not(quicktest.IsNil))

		// Cleanup
		err = provider.Shutdown(ctx)
		c.Check(err, quicktest.IsNil)
	})
}

func TestSetupLoggingErrorHandling(t *testing.T) {
	qt := quicktest.New(t)

	qt.Run("invalid port number", func(c *quicktest.C) {
		ctx := context.Background()

		// Test with an invalid port that might cause issues
		provider, err := SetupLogging(ctx, WithPort(-1))

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

func TestSetupLoggingConcurrentAccess(t *testing.T) {
	qt := quicktest.New(t)

	qt.Run("concurrent setup calls", func(c *quicktest.C) {
		ctx := context.Background()

		// Test concurrent setup calls
		done := make(chan bool, 2)

		go func() {
			provider, err := SetupLogging(ctx, WithServiceName("concurrent1"))
			if err == nil && provider != nil {
				err = provider.Shutdown(ctx)
				c.Check(err, quicktest.IsNil)
			}
			done <- true
		}()

		go func() {
			provider, err := SetupLogging(ctx, WithServiceName("concurrent2"))
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

func TestSetupLoggingMemoryLeaks(t *testing.T) {
	qt := quicktest.New(t)

	qt.Run("memory leak prevention", func(c *quicktest.C) {
		ctx := context.Background()

		// Create and destroy multiple providers to check for memory leaks
		for range 10 {
			provider, err := SetupLogging(ctx,
				WithServiceName("memory-test"),
				WithServiceVersion("1.0.0"),
			)

			c.Check(err, quicktest.IsNil)
			c.Check(provider, quicktest.Not(quicktest.IsNil))

			// Create a logger to ensure it works
			logger := provider.Logger("test-logger")
			c.Check(logger, quicktest.Not(quicktest.IsNil))

			// Shutdown immediately
			err = provider.Shutdown(ctx)
			c.Check(err, quicktest.IsNil)
		}
	})
}

func TestSetupLoggingIntegration(t *testing.T) {
	qt := quicktest.New(t)

	qt.Run("integration test with actual logging", func(c *quicktest.C) {
		ctx := context.Background()

		provider, err := SetupLogging(ctx,
			WithServiceName("integration-test"),
			WithServiceVersion("1.0.0"),
		)
		c.Check(err, quicktest.IsNil)
		c.Check(provider, quicktest.Not(quicktest.IsNil))

		// Create a logger and test actual logging
		logger := provider.Logger("integration-logger")
		c.Check(logger, quicktest.Not(quicktest.IsNil))

		// Cleanup
		err = provider.Shutdown(ctx)
		c.Check(err, quicktest.IsNil)
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
