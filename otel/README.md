# otel

OpenTelemetry setup and configuration for distributed tracing, metrics, and logging.

`x/otel` provides a unified interface for setting up OpenTelemetry observability components including tracing, metrics, and logging. It supports both local development (stdout) and production (OTLP) exporters with automatic resource configuration and cleanup management.

## 1. Overview

The `x/otel` package provides a comprehensive OpenTelemetry setup solution that:

1. **Unified Configuration** - Single interface for setting up all OpenTelemetry components
2. **Flexible Exporters** - Support for both local development (stdout) and production (OTLP) exporters
3. **Automatic Resource Management** - Built-in service name, version, and resource configuration
4. **Graceful Cleanup** - Proper shutdown handling for all providers
5. **Global Integration** - Automatic setup of global providers and propagators
6. **Error Handling** - Comprehensive error handling with cleanup on failures

## 2. Core Concepts

### 2.1 Observability Pillars

The package supports all three pillars of observability:

- **Tracing**: Distributed request tracing with span correlation
- **Metrics**: Application and business metrics collection
- **Logging**: Structured logging with trace correlation

### 2.2 Exporter Modes

The package supports two exporter modes:

- **Local Development**: Uses stdout exporters for easy debugging
- **Production**: Uses OTLP exporters for centralized observability

### 2.3 Resource Configuration

All telemetry data is automatically tagged with:

- Service name
- Service version
- Host information

## 3. API Reference

### 3.1 Core Functions

#### `Setup(ctx context.Context, options ...Option) (*SetupResult, error)`

Initializes all OpenTelemetry components (tracing, logging, metrics) with the given options and returns a `SetupResult` containing the providers and a cleanup function.

```go
result, err := otel.Setup(ctx,
    otel.WithServiceName("my-service"),
    otel.WithServiceVersion("1.0.0"),
    otel.WithCollectorEnable(true),
)
if err != nil {
    log.Fatal(err)
}
defer result.Cleanup()
```

#### `SetupWithCleanup(ctx context.Context, options ...Option) func()`

Convenience function that sets up OpenTelemetry and returns a cleanup function. Panics if setup fails.

```go
cleanup := otel.SetupWithCleanup(ctx,
    otel.WithServiceName("my-service"),
    otel.WithServiceVersion("1.0.0"),
)
defer cleanup()
```

#### `SetupTracing(ctx context.Context, options ...Option) (*trace.TracerProvider, error)`

Sets up only the tracing system.

```go
tracerProvider, err := otel.SetupTracing(ctx,
    otel.WithServiceName("my-service"),
    otel.WithCollectorEnable(false), // Use stdout for local development
)
```

#### `SetupMetrics(ctx context.Context, options ...Option) (*metric.MeterProvider, error)`

Sets up only the metrics system.

```go
meterProvider, err := otel.SetupMetrics(ctx,
    otel.WithServiceName("my-service"),
    otel.WithCollectorEnable(true), // Use OTLP for production
)
```

#### `SetupLogging(ctx context.Context, options ...Option) (*log.LoggerProvider, error)`

Sets up only the logging system.

```go
loggerProvider, err := otel.SetupLogging(ctx,
    otel.WithServiceName("my-service"),
    otel.WithCollectorEnable(false), // Use stdout for local development
)
```

### 3.2 Configuration Options

#### `WithServiceName(name string) Option`

Sets the service name for all telemetry data.

```go
otel.WithServiceName("user-service")
```

#### `WithServiceVersion(version string) Option`

Sets the service version for all telemetry data.

```go
otel.WithServiceVersion("1.2.3")
```

#### `WithHost(host string) Option`

Sets the host address for OTLP exporters.

```go
otel.WithHost("collector.example.com")
```

#### `WithPort(port string) Option`

Sets the port number for OTLP exporters.

```go
otel.WithPort("4317")
```

#### `WithCollectorEnable(enable bool) Option`

Enables or disables OTLP collector integration. When disabled, stdout exporters are used.

```go
otel.WithCollectorEnable(true)  // Use OTLP for production
otel.WithCollectorEnable(false) // Use stdout for development
```

### 3.3 Data Structures

#### `SetupResult`

Contains the initialized providers and cleanup function.

```go
type SetupResult struct {
    TracerProvider *trace.TracerProvider
    LoggerProvider *log.LoggerProvider
    MeterProvider  *metric.MeterProvider
    Cleanup        func() error
}
```

#### `Options`

Internal configuration structure.

```go
type Options struct {
    ServiceName     string
    ServiceVersion  string
    Host            string
    Port            string
    CollectorEnable bool
}
```

## 4. Usage Examples

### 4.1 Basic Setup

```go
package main

import (
    "context"
    "log"
    "github.com/instill-ai/x/otel"
)

func main() {
    ctx := context.Background()

    // Setup all OpenTelemetry components
    result, err := otel.Setup(ctx,
        otel.WithServiceName("my-application"),
        otel.WithServiceVersion("1.0.0"),
        otel.WithCollectorEnable(false), // Use stdout for development
    )
    if err != nil {
        log.Fatal(err)
    }
    defer result.Cleanup()

    // Your application code here
}
```

### 4.2 Production Setup

```go
package main

import (
    "context"
    "log"
    "github.com/instill-ai/x/otel"
)

func main() {
    ctx := context.Background()

    // Setup for production with OTLP collector
    cleanup := otel.SetupWithCleanup(ctx,
        otel.WithServiceName("production-service"),
        otel.WithServiceVersion("2.1.0"),
        otel.WithHost("collector.production.com"),
        otel.WithPort("4317"),
        otel.WithCollectorEnable(true),
    )
    defer cleanup()

    // Your application code here
}
```

### 4.3 Individual Component Setup

```go
package main

import (
    "context"
    "log"
    "github.com/instill-ai/x/otel"
)

func main() {
    ctx := context.Background()

    // Setup only tracing
    tracerProvider, err := otel.SetupTracing(ctx,
        otel.WithServiceName("tracing-service"),
        otel.WithCollectorEnable(false),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer tracerProvider.Shutdown(ctx)

    // Setup only metrics
    meterProvider, err := otel.SetupMetrics(ctx,
        otel.WithServiceName("metrics-service"),
        otel.WithCollectorEnable(true),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer meterProvider.Shutdown(ctx)

    // Your application code here
}
```

### 4.4 Using the Providers

```go
package main

import (
    "context"
    "github.com/instill-ai/x/otel"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/metric"
    "go.opentelemetry.io/otel/trace"
)

func main() {
    ctx := context.Background()

    // Setup OpenTelemetry
    result, err := otel.Setup(ctx,
        otel.WithServiceName("example-service"),
        otel.WithServiceVersion("1.0.0"),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer result.Cleanup()

    // Use tracing
    tracer := otel.GetTracerProvider().Tracer("example-tracer")
    ctx, span := tracer.Start(ctx, "main-operation")
    defer span.End()

    // Use metrics
    meter := otel.GetMeterProvider().Meter("example-meter")
    counter, _ := meter.Int64Counter("requests_total")
    counter.Add(ctx, 1)

    // Use logging (if you have a logger implementation)
    // logger := result.LoggerProvider.Logger("example-logger")
}
```

### 4.5 Error Handling

```go
package main

import (
    "context"
    "fmt"
    "log"
    "github.com/instill-ai/x/otel"
)

func main() {
    ctx := context.Background()

    result, err := otel.Setup(ctx,
        otel.WithServiceName("error-handling-service"),
        otel.WithHost("invalid-host"),
        otel.WithCollectorEnable(true),
    )
    if err != nil {
        // Handle setup errors gracefully
        log.Printf("Failed to setup OpenTelemetry: %v", err)
        log.Println("Continuing without observability...")
        return
    }

    // Ensure cleanup even if setup partially succeeded
    if result != nil {
        defer func() {
            if cleanupErr := result.Cleanup(); cleanupErr != nil {
                log.Printf("Cleanup error: %v", cleanupErr)
            }
        }()
    }
}
```

## 5. Best Practices

### 5.1 Service Configuration

- **Use meaningful service names**: Choose descriptive names that identify your service
- **Version your services**: Include version information for better debugging
- **Consistent naming**: Use consistent naming conventions across your services

```go
// Good
otel.WithServiceName("user-authentication-service")
otel.WithServiceVersion("2.1.0")

// Avoid
otel.WithServiceName("service1")
otel.WithServiceVersion("latest")
```

### 5.2 Environment-Specific Configuration

- **Development**: Use stdout exporters for easy debugging
- **Production**: Use OTLP exporters for centralized observability
- **Configuration management**: Use environment variables for dynamic configuration

```go
func getOtelConfig() []otel.Option {
    serviceName := os.Getenv("SERVICE_NAME")
    if serviceName == "" {
        serviceName = "unknown-service"
    }

    collectorEnabled := os.Getenv("COLLECTOR_ENABLED") == "true"

    return []otel.Option{
        otel.WithServiceName(serviceName),
        otel.WithServiceVersion(os.Getenv("SERVICE_VERSION")),
        otel.WithHost(os.Getenv("COLLECTOR_HOST")),
        otel.WithPort(os.Getenv("COLLECTOR_PORT")),
        otel.WithCollectorEnable(collectorEnabled),
    }
}
```

### 5.3 Resource Management

- **Always cleanup**: Use defer statements to ensure proper cleanup
- **Handle cleanup errors**: Log cleanup errors but don't fail the application
- **Graceful shutdown**: Allow time for telemetry data to be exported

```go
func main() {
    ctx := context.Background()

    result, err := otel.Setup(ctx, getOtelConfig()...)
    if err != nil {
        log.Fatal(err)
    }

    // Ensure cleanup on exit
    defer func() {
        if cleanupErr := result.Cleanup(); cleanupErr != nil {
            log.Printf("Cleanup warning: %v", cleanupErr)
        }
    }()

    // Your application code
}
```

### 5.4 Error Handling

- **Graceful degradation**: Continue running even if OpenTelemetry setup fails
- **Partial failures**: Handle cases where only some components fail to initialize
- **Logging**: Provide meaningful error messages for debugging

```go
func setupObservability(ctx context.Context) (*otel.SetupResult, error) {
    result, err := otel.Setup(ctx, getOtelConfig()...)
    if err != nil {
        // Log the error but don't fail the application
        log.Printf("Observability setup failed: %v", err)
        log.Println("Application will continue without observability")
        return nil, err
    }

    log.Println("Observability setup completed successfully")
    return result, nil
}
```

### 5.5 Performance Considerations

- **Batch processing**: Use batch processors for better performance
- **Sampling**: Implement sampling strategies for high-traffic services
- **Resource limits**: Monitor memory and CPU usage of telemetry components

```go
// The package automatically uses batch processors for better performance
// For custom sampling, you can configure the tracer provider after setup
func configureSampling(tracerProvider *trace.TracerProvider) {
    // Custom sampling configuration if needed
}
```

### 5.6 Testing

- **Mock providers**: Use mock providers for unit tests
- **Integration tests**: Test the full setup and cleanup cycle
- **Error scenarios**: Test error handling and recovery

```go
func TestObservabilitySetup(t *testing.T) {
    ctx := context.Background()

    result, err := otel.Setup(ctx,
        otel.WithServiceName("test-service"),
        otel.WithCollectorEnable(false), // Use stdout for tests
    )
    require.NoError(t, err)
    require.NotNil(t, result)

    // Test that providers work
    tracer := result.TracerProvider.Tracer("test-tracer")
    assert.NotNil(t, tracer)

    // Cleanup
    err = result.Cleanup()
    assert.NoError(t, err)
}
```

## 6. Migration Guide

### 6.1 From Manual OpenTelemetry Setup

**Before:**

```go
func setupOpenTelemetry() error {
    // Manual setup of each component
    exporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithEndpoint("localhost:4317"))
    if err != nil {
        return err
    }

    resource := resource.NewWithAttributes(
        semconv.SchemaURL,
        semconv.ServiceNameKey.String("my-service"),
    )

    provider := trace.NewTracerProvider(
        trace.WithBatcher(exporter),
        trace.WithResource(resource),
    )

    otel.SetTracerProvider(provider)
    return nil
}
```

**After:**

```go
func setupOpenTelemetry() error {
    result, err := otel.Setup(ctx,
        otel.WithServiceName("my-service"),
        otel.WithCollectorEnable(true),
    )
    if err != nil {
        return err
    }

    // Store result for cleanup
    defer result.Cleanup()
    return nil
}
```

### 6.2 From Different Observability Libraries

**Before:**

```go
// Using a different observability library
import "github.com/other/observability"

func main() {
    observability.Init(observability.Config{
        ServiceName: "my-service",
        Endpoint:    "localhost:4317",
    })
}
```

**After:**

```go
// Using x/otel
import "github.com/instill-ai/x/otel"

func main() {
    cleanup := otel.SetupWithCleanup(ctx,
        otel.WithServiceName("my-service"),
        otel.WithCollectorEnable(true),
    )
    defer cleanup()
}
```

## 7. Configuration Examples

### 7.1 Development Environment

```go
func getDevConfig() []otel.Option {
    return []otel.Option{
        otel.WithServiceName("dev-service"),
        otel.WithServiceVersion("dev"),
        otel.WithCollectorEnable(false), // Use stdout
    }
}
```

### 7.2 Staging Environment

```go
func getStagingConfig() []otel.Option {
    return []otel.Option{
        otel.WithServiceName("staging-service"),
        otel.WithServiceVersion("1.0.0"),
        otel.WithHost("staging-collector.example.com"),
        otel.WithPort("4317"),
        otel.WithCollectorEnable(true),
    }
}
```

### 7.3 Production Environment

```go
func getProductionConfig() []otel.Option {
    return []otel.Option{
        otel.WithServiceName("production-service"),
        otel.WithServiceVersion("2.1.0"),
        otel.WithHost("collector.production.com"),
        otel.WithPort("4317"),
        otel.WithCollectorEnable(true),
    }
}
```

## 8. Performance Considerations

- **Minimal overhead**: The package is designed for minimal runtime overhead
- **Efficient exporters**: Uses batch processors and efficient encoding
- **Resource management**: Proper cleanup prevents memory leaks
- **Configurable intervals**: Metrics are exported at configurable intervals (default: 10 seconds)

## 9. Troubleshooting

### 9.1 Common Issues

1. **Connection refused errors**: Check collector endpoint and network connectivity
2. **Memory leaks**: Ensure proper cleanup with defer statements
3. **Missing telemetry data**: Verify service name and version configuration
4. **Performance impact**: Monitor resource usage and adjust sampling if needed

### 9.2 Debug Mode

For debugging, use stdout exporters:

```go
result, err := otel.Setup(ctx,
    otel.WithServiceName("debug-service"),
    otel.WithCollectorEnable(false), // Use stdout
)
```

### 9.3 Health Checks

Monitor the health of your observability setup:

```go
func checkObservabilityHealth(result *otel.SetupResult) error {
    // Check if providers are working
    tracer := result.TracerProvider.Tracer("health-check")
    ctx, span := tracer.Start(context.Background(), "health-check")
    defer span.End()

    return nil
}
```

## 10. Contributing

When adding new features or functionality:

1. **Follow existing patterns**: Use the established conventions for option functions
2. **Add comprehensive tests**: Include unit tests for new functionality
3. **Update documentation**: Keep this README current with new features
4. **Consider backward compatibility**: Maintain compatibility with existing APIs

## 11. License

This package is part of the Instill AI x library and follows the same licensing terms.
