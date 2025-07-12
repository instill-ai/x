# server

A comprehensive gRPC server framework with built-in interceptors, logging, tracing, and gateway support.

The `x/server` package provides a production-ready gRPC server setup with comprehensive middleware support, including logging, tracing, error handling, and gRPC-Gateway integration. It's designed to work seamlessly with the Instill AI platform and follows best practices for observability and error handling.

## Overview

The `x/server` package provides:

1. **Pre-configured gRPC server options** - Ready-to-use server configuration with sensible defaults
2. **Comprehensive interceptor chain** - Built-in logging, tracing, error handling, and recovery
3. **Flexible logging control** - Configurable method exclusion patterns for selective logging
4. **OpenTelemetry integration** - Automatic tracing and metrics collection
5. **gRPC-Gateway support** - HTTP/JSON gateway with custom error handling
6. **TLS/SSL support** - Secure communication with certificate-based authentication

## Core Components

### gRPC Server Options (`grpc/options.go`)

The main entry point for creating gRPC server configurations with all necessary interceptors and options.

#### `NewGRPCOptionsAndCreds(options ...Option) ([]grpc.ServerOption, credentials.TransportCredentials, error)`

Creates a complete gRPC server configuration with:

- **Interceptor chain**: Metadata, decider, tracing, and recovery interceptors
- **TLS credentials**: Automatic certificate-based security
- **OpenTelemetry integration**: Built-in tracing and metrics
- **Message size limits**: Configurable payload size limits

```go
// Basic usage
opts, creds, err := grpc.NewGRPCOptionsAndCreds(
    grpc.WithServiceName("my-service"),
    grpc.WithServiceVersion("v1.0.0"),
    grpc.WithOTELCollectorEnable(true),
)
if err != nil {
    log.Fatal(err)
}

server := grpc.NewServer(opts...)
```

#### Configuration Options

```go
// Service configuration
grpc.WithServiceConfig(client.HTTPSConfig{
    Cert: "/path/to/cert.pem",
    Key:  "/path/to/key.pem",
})

// Service metadata
grpc.WithServiceName("my-service")
grpc.WithServiceVersion("v1.0.0")

// Observability
grpc.WithOTELCollectorEnable(true)

// Logging control
grpc.WithMethodExcludePatterns([]string{
    "*.Health/*",
    "*.Metrics/*",
})
```

### Interceptors

#### 1. Metadata Interceptor (`metadata.go`)

Handles gRPC metadata propagation and error conversion.

**UnaryAppendMetadataAndErrorCodeInterceptor**

- Preserves incoming metadata in context
- Converts errors to gRPC status errors using `x/errors.ConvertToGRPCError`

**StreamAppendMetadataInterceptor**

- Handles metadata for streaming RPCs
- Wraps stream context with metadata

#### 2. Decider Interceptor (`decider.go`)

Controls which gRPC calls should be logged based on method patterns.

**Default Exclusions:**

```go
var DefaultMethodExcludePatterns = []string{
    "*PublicService/.*ness$",  // Health checks
    "*PrivateService/.*$",     // Private service calls
}
```

**Usage:**

```go
// Custom exclusion patterns
patterns := []string{
    "*.Health/*",
    "*.Metrics/*",
    "*.Internal/*",
}

interceptor := interceptor.DeciderUnaryServerInterceptor(patterns)
```

#### 3. Tracing Interceptor (`trace.go`)

Provides comprehensive request logging with trace context and OpenTelemetry integration.

**Features:**

- **Structured logging**: JSON-formatted logs with trace IDs
- **Performance metrics**: Request duration tracking
- **Error categorization**: Automatic log level selection based on gRPC codes
- **OpenTelemetry support**: Dual logging to both Zap and OTEL

**Log Levels by gRPC Code:**

- `codes.OK` → Info
- `codes.Canceled`, `codes.DeadlineExceeded`, etc. → Warn
- `codes.InvalidArgument`, `codes.NotFound`, etc. → Error

**Example Log Output:**

```json
{
  "level": "info",
  "msg": "finished unary call CreateUser (trace_id: 1234567890abcdef)",
  "timestamp": "2024-01-01T12:00:00Z",
  "duration": "15.2ms"
}
```

#### 4. Recovery Interceptor (`recovery.go`)

Handles panics and converts them to gRPC errors.

```go
// Automatically recovers from panics
// Converts panic values to gRPC status errors
// Prevents server crashes from unhandled panics
```

### gRPC-Gateway Support (`gateway/`)

Provides HTTP/JSON gateway functionality with custom error handling and response modification.

#### HTTPResponseModifier

Modifies HTTP responses based on gRPC metadata:

- Sets custom HTTP status codes via `x-http-code` header
- Removes internal headers from responses

#### ErrorHandler

Custom error handling for HTTP responses:

- Converts gRPC errors to appropriate HTTP status codes
- Provides structured error responses
- Handles authentication headers (`WWW-Authenticate`)

#### CustomHeaderMatcher

Controls which HTTP headers are forwarded to gRPC:

- JWT headers (`jwt-*`)
- Instill headers (`instill-*`)
- GitHub headers (`x-github*`)
- Standard headers (`accept`, `request-id`, etc.)

## API Reference

### Core Functions

#### `grpc.NewGRPCOptionsAndCreds(options ...Option) ([]grpc.ServerOption, credentials.TransportCredentials, error)`

Creates gRPC server options and credentials.

**Parameters:**

- `options`: Configuration options (see Options section)

**Returns:**

- `[]grpc.ServerOption`: Server options for `grpc.NewServer()`
- `credentials.TransportCredentials`: TLS credentials (if configured)
- `error`: Any configuration errors

#### `interceptor.DeciderUnaryServerInterceptor(patterns []string) grpc.UnaryServerInterceptor`

Creates a unary interceptor that controls logging based on method patterns.

**Parameters:**

- `patterns`: Regex patterns for methods to exclude from logging

#### `interceptor.TracingUnaryServerInterceptor(serviceName, serviceVersion string, otelEnable bool) grpc.UnaryServerInterceptor`

Creates a unary interceptor for request tracing and logging.

**Parameters:**

- `serviceName`: Service identifier for logs
- `serviceVersion`: Service version for logs
- `otelEnable`: Enable OpenTelemetry logging

#### `gateway.HTTPResponseModifier(ctx context.Context, w http.ResponseWriter, p proto.Message) error`

Modifies HTTP responses based on gRPC metadata.

#### `gateway.ErrorHandler(ctx context.Context, mux *runtime.ServeMux, marshaler runtime.Marshaler, w http.ResponseWriter, r *http.Request, err error)`

Handles gRPC errors in HTTP responses.

### Configuration Options

#### `grpc.WithServiceConfig(config client.HTTPSConfig)`

Sets TLS/SSL configuration.

```go
config := client.HTTPSConfig{
    Cert: "/path/to/cert.pem",
    Key:  "/path/to/key.pem",
}
```

#### `grpc.WithServiceName(name string)`

Sets the service name for logging and tracing.

#### `grpc.WithServiceVersion(version string)`

Sets the service version for logging and tracing.

#### `grpc.WithOTELCollectorEnable(enable bool)`

Enables or disables OpenTelemetry collector integration.

#### `grpc.WithMethodExcludePatterns(patterns []string)`

Sets custom method exclusion patterns for logging.

## Usage Examples

### Basic gRPC Server Setup

```go
package main

import (
    "log"
    "net"

    "google.golang.org/grpc"
    "github.com/instill-ai/x/server/grpc"
)

func main() {
    // Create server options
    opts, creds, err := grpc.NewGRPCOptionsAndCreds(
        grpc.WithServiceName("user-service"),
        grpc.WithServiceVersion("v1.0.0"),
        grpc.WithOTELCollectorEnable(true),
    )
    if err != nil {
        log.Fatal(err)
    }

    // Create gRPC server
    server := grpc.NewServer(opts...)

    // Register your services
    // pb.RegisterUserServiceServer(server, &userService{})

    // Start server
    lis, err := net.Listen("tcp", ":50051")
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("Server listening on :50051")
    if err := server.Serve(lis); err != nil {
        log.Fatal(err)
    }
}
```

### gRPC Server with TLS

```go
// Create server options with TLS
opts, creds, err := grpc.NewGRPCOptionsAndCreds(
    grpc.WithServiceName("secure-service"),
    grpc.WithServiceVersion("v1.0.0"),
    grpc.WithServiceConfig(client.HTTPSConfig{
        Cert: "/path/to/server.crt",
        Key:  "/path/to/server.key",
    }),
    grpc.WithOTELCollectorEnable(true),
)
if err != nil {
    log.Fatal(err)
}

// creds contains the TLS credentials
log.Printf("TLS enabled: %v", creds != nil)
```

### Custom Logging Patterns

```go
// Exclude health checks and metrics from logging
opts, _, err := grpc.NewGRPCOptionsAndCreds(
    grpc.WithServiceName("api-service"),
    grpc.WithMethodExcludePatterns([]string{
        "*.Health/*",
        "*.Metrics/*",
        "*.Internal/*",
        "*.Debug/*",
    }),
)
```

### gRPC-Gateway Setup

```go
package main

import (
    "context"
    "net/http"

    "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"

    "github.com/instill-ai/x/server/grpc/gateway"
    pb "your-project/proto"
)

func main() {
    // Create gRPC client connection
    conn, err := grpc.DialContext(
        context.Background(),
        "localhost:50051",
        grpc.WithTransportCredentials(insecure.NewCredentials()),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer conn.Close()

    // Create gRPC-Gateway mux
    mux := runtime.NewServeMux(
        runtime.WithForwardResponseOption(gateway.HTTPResponseModifier),
        runtime.WithErrorHandler(gateway.ErrorHandler),
        runtime.WithIncomingHeaderMatcher(gateway.CustomHeaderMatcher),
    )

    // Register your gRPC service
    if err := pb.RegisterUserServiceHandler(context.Background(), mux, conn); err != nil {
        log.Fatal(err)
    }

    // Start HTTP server
    log.Printf("Gateway listening on :8080")
    if err := http.ListenAndServe(":8080", mux); err != nil {
        log.Fatal(err)
    }
}
```

### Custom Interceptor Chain

```go
// Create custom interceptor chain
unaryInterceptors := grpcmiddleware.ChainUnaryServer(
    interceptor.UnaryAppendMetadataAndErrorCodeInterceptor,
    interceptor.DeciderUnaryServerInterceptor([]string{"*.Health/*"}),
    interceptor.TracingUnaryServerInterceptor("my-service", "v1.0.0", true),
    grpcrecovery.UnaryServerInterceptor(interceptor.RecoveryInterceptorOpt()),
    // Add your custom interceptors here
)

streamInterceptors := grpcmiddleware.ChainStreamServer(
    interceptor.StreamAppendMetadataInterceptor,
    interceptor.DeciderStreamServerInterceptor([]string{"*.Health/*"}),
    interceptor.TracingStreamServerInterceptor("my-service", "v1.0.0", true),
    grpcrecovery.StreamServerInterceptor(interceptor.RecoveryInterceptorOpt()),
    // Add your custom interceptors here
)

server := grpc.NewServer(
    grpc.UnaryInterceptor(unaryInterceptors),
    grpc.StreamInterceptor(streamInterceptors),
)
```

## Best Practices

### 1. Service Configuration

- **Use descriptive service names**: Help with log aggregation and tracing
- **Version your services**: Enable tracking of service versions in logs
- **Enable OpenTelemetry**: For production observability

```go
opts, _, err := grpc.NewGRPCOptionsAndCreds(
    grpc.WithServiceName("user-management-service"),
    grpc.WithServiceVersion("v2.1.0"),
    grpc.WithOTELCollectorEnable(true),
)
```

### 2. Logging Strategy

- **Exclude noisy endpoints**: Health checks, metrics, and internal calls
- **Use consistent patterns**: Standardize method exclusion patterns across services
- **Monitor log volume**: Adjust patterns based on actual usage

```go
// Recommended exclusion patterns
patterns := []string{
    "*.Health/*",           // Health checks
    "*.Metrics/*",          // Metrics endpoints
    "*.Internal/*",         // Internal service calls
    "*.Debug/*",            // Debug endpoints
    "*PublicService/.*ness$", // Liveness/readiness checks
}
```

### 3. Error Handling

- **Use x/errors package**: Leverage the integrated error handling
- **Preserve error context**: Let interceptors handle error conversion
- **Monitor error rates**: Use the built-in error categorization

```go
// In your service handlers, use x/errors
import "github.com/instill-ai/x/errors"

func (s *Service) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.CreateUserResponse, error) {
    if err := s.validateUser(req); err != nil {
        return nil, errors.AddMessage(err, "Please check your input and try again.")
    }

    // The interceptor will automatically convert this to gRPC status
    return s.repo.CreateUser(ctx, req)
}
```

### 4. Security

- **Use TLS in production**: Always configure certificates for production
- **Validate certificates**: Ensure proper certificate validation
- **Monitor security events**: Use the built-in logging for security monitoring

```go
// Production TLS configuration
config := client.HTTPSConfig{
    Cert: "/etc/ssl/certs/server.crt",
    Key:  "/etc/ssl/private/server.key",
}

opts, creds, err := grpc.NewGRPCOptionsAndCreds(
    grpc.WithServiceConfig(config),
    // ... other options
)
```

### 5. Observability

- **Enable OpenTelemetry**: For comprehensive tracing and metrics
- **Monitor request patterns**: Use the built-in request logging
- **Track performance**: Leverage the automatic duration tracking

```go
// Enable full observability
opts, _, err := grpc.NewGRPCOptionsAndCreds(
    grpc.WithOTELCollectorEnable(true),
    grpc.WithServiceName("my-service"),
    grpc.WithServiceVersion("v1.0.0"),
)
```

### 6. Testing

- **Test interceptor behavior**: Verify logging and error handling
- **Mock external dependencies**: Use the provided mock utilities
- **Test error scenarios**: Ensure proper error conversion and logging

```go
func TestService_WithInterceptors(t *testing.T) {
    // Use the provided mock utilities
    mockLogger := &interceptor.MockLogger{}

    // Test your service with interceptors
    // Verify logging behavior and error handling
}
```

## Migration Guide

### From Standard gRPC Server

**Before:**

```go
server := grpc.NewServer()
// Manual interceptor setup
// Manual error handling
// Manual logging
```

**After:**

```go
opts, _, err := grpc.NewGRPCOptionsAndCreds(
    grpc.WithServiceName("my-service"),
    grpc.WithServiceVersion("v1.0.0"),
    grpc.WithOTELCollectorEnable(true),
)
if err != nil {
    log.Fatal(err)
}

server := grpc.NewServer(opts...)
// Automatic interceptor chain
// Automatic error handling
// Automatic structured logging
```

### Adding Custom Interceptors

```go
// Create base options
opts, _, err := grpc.NewGRPCOptionsAndCreds(
    grpc.WithServiceName("my-service"),
)

// Add custom interceptors
unaryInterceptors := append([]grpc.UnaryServerInterceptor{
    myCustomInterceptor,
}, opts...)

server := grpc.NewServer(grpc.UnaryInterceptor(unaryInterceptors...))
```

## Performance Considerations

- **Minimal overhead**: Interceptors are optimized for performance
- **Selective logging**: Use exclusion patterns to reduce log volume
- **Efficient error handling**: Error conversion is optimized for common cases
- **Memory efficient**: Context propagation is designed for minimal allocations

## Troubleshooting

### Common Issues

1. **Missing TLS certificates**: Ensure certificate files exist and are readable
2. **Log volume too high**: Adjust method exclusion patterns
3. **OpenTelemetry not working**: Verify OTEL collector is running and accessible
4. **gRPC-Gateway errors**: Check header matcher configuration

### Debug Mode

Enable debug logging to troubleshoot interceptor behavior:

```go
// Set log level to debug
log.SetLevel(zap.DebugLevel)

// Check interceptor chain
opts, _, err := grpc.NewGRPCOptionsAndCreds(
    grpc.WithServiceName("debug-service"),
    grpc.WithOTELCollectorEnable(false), // Disable OTEL for debugging
)
```

## Contributing

When adding new functionality:

1. **Follow existing patterns**: Use the established interceptor patterns
2. **Add comprehensive tests**: Include unit tests for new interceptors
3. **Update documentation**: Keep this README current with new features
4. **Consider performance**: Ensure new interceptors don't impact performance
5. **Add examples**: Provide usage examples for new features

## License

This package is part of the Instill AI x library and follows the same licensing terms.
