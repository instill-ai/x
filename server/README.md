# server

A comprehensive gRPC server framework with built-in interceptors, logging, tracing, and gateway support.

The `x/server` package provides a production-ready gRPC server setup with comprehensive middleware support, including logging, tracing, error handling, and gRPC-Gateway integration. It's designed to work seamlessly with the Instill AI platform and follows best practices for observability and error handling.

## 1. Overview

The `x/server` package provides:

1. **Pre-configured gRPC server options** - Ready-to-use server configuration with sensible defaults
2. **Comprehensive interceptor chain** - Built-in logging, tracing, error handling, and recovery
3. **Flexible logging and tracing control** - Configurable method exclusion patterns for selective logging and tracing
4. **OpenTelemetry integration** - Automatic tracing and metrics collection with custom filtering
5. **gRPC-Gateway support** - HTTP/JSON gateway with custom error handling
6. **TLS/SSL support** - Secure communication with certificate-based authentication
7. **Comprehensive testing** - Unit tests with minimock for all components

## 2. Core Components

### 2.1 gRPC Server Options (`grpc/options.go`)

The main entry point for creating gRPC server configurations with all necessary interceptors and options.

#### `NewServerOptionsAndCreds(options ...Option) ([]grpc.ServerOption, error)`

Creates a complete gRPC server configuration with:

- **Interceptor chain**: Metadata, logging, tracing, and recovery interceptors
- **TLS credentials**: Automatic certificate-based security
- **OpenTelemetry integration**: Built-in tracing and metrics with custom filtering
- **Message size limits**: Configurable payload size limits
- **Dual filtering**: Separate control for logging and tracing exclusions

```go
// Basic usage
opts, err := grpc.NewServerOptionsAndCreds(
    grpc.WithServiceName("my-service"),
    grpc.WithServiceVersion("v1.0.0"),
    grpc.WithSetOTELServerHandler(true),
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
grpc.WithSetOTELServerHandler(true)

// Logging control
grpc.WithMethodLogExcludePatterns([]string{
    ".*Health/.*",
    ".*Metrics/.*",
})

// Tracing control
grpc.WithMethodTraceExcludePatterns([]string{
    ".*TestService/.*",
    ".*DebugService/.*",
})
```

### 2.2 Filter Deciders

The server package now provides two types of filter deciders for fine-grained control over observability:

#### 2.2.1 Log Filter Decider

Controls which gRPC calls should be logged based on method patterns.

**Default Log Exclusions:**

```go
var defaultMethodLogExcludePatterns = []string{
    ".*PublicService/.*ness$",  // Health checks (liveness/readiness)
    ".*PrivateService/.*$",     // Private service calls
    ".*UsageService/.*$",       // Usage service calls
}
```

**Usage:**

```go
// Custom log exclusion patterns
opts, err := grpc.NewServerOptionsAndCreds(
    grpc.WithServiceName("my-service"),
    grpc.WithMethodLogExcludePatterns([]string{
        ".*Health/.*",
        ".*Metrics/.*",
        ".*Internal/.*",
    }),
)
```

#### 2.2.2 Trace Filter Decider

Controls which gRPC calls should be traced using OpenTelemetry.

**Default Trace Exclusions:**

```go
var defaultMethodTraceExcludePatterns = []string{
    ".*PublicService/.*ness$",  // Health checks (liveness/readiness)
    ".*PrivateService/.*$",     // Private service calls
    ".*UsageService/.*$",       // Usage service calls
}
```

**Usage:**

```go
// Custom trace exclusion patterns
opts, err := grpc.NewServerOptionsAndCreds(
    grpc.WithServiceName("my-service"),
    grpc.WithSetOTELServerHandler(true),
    grpc.WithMethodTraceExcludePatterns([]string{
        ".*TestService/.*",
        ".*DebugService/.*",
        ".*HealthService/.*",
    }),
)
```

**Important:** Method exclusion patterns use Go regex syntax. Use `.*` to match any characters (not `*`). For example:

- `".*PublicService/.*ness$"` - matches any service ending with "PublicService" and methods ending with "ness"
- `".*Health/.*"` - matches any service containing "Health" and any method
- `".*Internal/.*"` - matches any service containing "Internal" and any method

### 2.3 Interceptors

#### 2.3.1 Metadata Interceptor (`interceptor/metadata.go`)

Handles gRPC metadata preservation and error conversion for both unary and streaming RPCs.

**UnaryAppendMetadataInterceptor**

- Preserves incoming metadata in context by ensuring metadata is always present
- Creates empty metadata if none exists to ensure consistent behavior
- Converts errors to gRPC status errors using `x/errors.ConvertToGRPCError`

**StreamAppendMetadataInterceptor**

- Handles metadata for streaming RPCs
- Wraps stream context with metadata using `grpc_middleware.WrapServerStream`
- Ensures metadata is always present in stream context
- Converts errors to gRPC status errors

**Key Features:**

- **Consistent metadata handling**: Always ensures metadata is present in context
- **Error conversion**: Automatic conversion of domain errors to gRPC status codes
- **Stream support**: Full support for both unary and streaming RPCs
- **Graceful fallback**: Creates empty metadata when none exists

```go
// Automatically applied in NewServerOptionsAndCreds
// No manual configuration required
```

#### 2.3.2 Recovery Interceptor (`interceptor/recovery.go`)

Handles panics and converts them to gRPC errors.

```go
// Automatically recovers from panics
// Converts panic values to gRPC status errors
// Prevents server crashes from unhandled panics
```

### 2.4 gRPC-Gateway Support (`gateway/misc.go`)

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

## 3. API Reference

### 3.1 Core Functions

#### `grpc.NewServerOptionsAndCreds(options ...Option) ([]grpc.ServerOption, error)`

Creates gRPC server options.

**Parameters:**

- `options`: Configuration options (see Options section)

**Returns:**

- `[]grpc.ServerOption`: Server options for `grpc.NewServer()`
- `error`: Any configuration errors

### 3.2 Configuration Options

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

#### `grpc.WithSetOTELServerHandler(enable bool)`

Enables or disables OpenTelemetry collector integration.

#### `grpc.WithMethodLogExcludePatterns(patterns []string)`

Sets custom method exclusion patterns for logging.

#### `grpc.WithMethodTraceExcludePatterns(patterns []string)`

Sets custom method exclusion patterns for tracing.

## 4. Usage Examples

### 4.1 Basic gRPC Server Setup

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
    opts, err := grpc.NewServerOptionsAndCreds(
        grpc.WithServiceName("user-service"),
        grpc.WithServiceVersion("v1.0.0"),
        grpc.WithSetOTELServerHandler(true),
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

### 4.2 gRPC Server with TLS

```go
// Create server options with TLS
opts, err := grpc.NewServerOptionsAndCreds(
    grpc.WithServiceName("secure-service"),
    grpc.WithServiceVersion("v1.0.0"),
    grpc.WithServiceConfig(client.HTTPSConfig{
        Cert: "/path/to/server.crt",
        Key:  "/path/to/server.key",
    }),
    grpc.WithSetOTELServerHandler(true),
)
if err != nil {
    log.Fatal(err)
}

// TLS credentials are automatically configured in the server options
log.Printf("TLS enabled: %v", opts != nil)
```

### 4.3 Custom Logging and Tracing Patterns

```go
// Exclude health checks and metrics from logging and tracing
opts, err := grpc.NewServerOptionsAndCreds(
    grpc.WithServiceName("api-service"),
    grpc.WithSetOTELServerHandler(true),
    grpc.WithMethodLogExcludePatterns([]string{
        ".*Health/.*",
        ".*Metrics/.*",
        ".*Internal/.*",
        ".*Debug/.*",
    }),
    grpc.WithMethodTraceExcludePatterns([]string{
        ".*TestService/.*",
        ".*DebugService/.*",
        ".*HealthService/.*",
    }),
)
```

### 4.4 gRPC-Gateway Setup

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

## 5. Testing

The server package includes comprehensive unit tests using **minimock** for all components.

### 5.1 Mock Generation

```bash
cd mock && go generate ./generator.go
```

Generates mocks for: `Logger`, `OTELLogger`, `ServerStream`, `Marshaler`, `Decoder`, `Encoder`, `ProtoMessage`.

### 5.2 Unit Testing Examples

```go
func TestMetadataInterceptor_WithMocks(t *testing.T) {
    qt := quicktest.New(t)
    mc := minimock.NewController(t)

    mockStream := mockserver.NewServerStreamMock(mc)

    // Set up stream context expectations
    mockStream.ContextMock.Expect().Return(context.Background())
    mockStream.ContextMock.Expect().Return(context.Background())

    // Test metadata handling
    interceptor := interceptor.StreamAppendMetadataInterceptor

    // ... test implementation
}
```

#### Best Practices

- **Use minimock for unit tests**: Isolated testing with generated mocks
- **Test interceptor behavior**: Verify logging and error handling
- **Test error scenarios**: Ensure proper error conversion and logging
- **Use quicktest assertions**: Consistent test assertions
- **Mock external dependencies**: Avoid external service dependencies in unit tests

## 6. Best Practices

### 6.1 Service Configuration

- **Use descriptive service names**: Help with log aggregation and tracing
- **Version your services**: Enable tracking of service versions in logs
- **Enable OpenTelemetry**: For production observability

```go
opts, err := grpc.NewServerOptionsAndCreds(
    grpc.WithServiceName("user-management-service"),
    grpc.WithServiceVersion("v2.1.0"),
    grpc.WithSetOTELServerHandler(true),
)
```

### 6.2 Logging and Tracing Strategy

- **Exclude noisy endpoints**: Health checks, metrics, and internal calls
- **Use consistent patterns**: Standardize method exclusion patterns across services
- **Monitor log volume**: Adjust patterns based on actual usage
- **Separate logging and tracing**: Use different patterns for logging vs tracing

```go
// Recommended exclusion patterns
logPatterns := []string{
    ".*Health/.*",           // Health checks
    ".*Metrics/.*",          // Metrics endpoints
    ".*Internal/.*",         // Internal service calls
    ".*Debug/.*",            // Debug endpoints
    ".*PublicService/.*ness$", // Liveness/readiness checks
}

tracePatterns := []string{
    ".*TestService/.*",      // Test services
    ".*DebugService/.*",     // Debug services
    ".*HealthService/.*",    // Health services
}
```

### 6.3 Error Handling

- **Use x/errors package**: Leverage the integrated error handling
- **Preserve error context**: Let interceptors handle error conversion
- **Monitor error rates**: Use the built-in error categorization
- **Consistent metadata**: Metadata interceptors ensure metadata is always present

```go
// In your service handlers, use x/errors
import "github.com/instill-ai/x/errors"

func (s *Service) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.CreateUserResponse, error) {
    if err := s.validateUser(req); err != nil {
        return nil, errors.AddMessage(err, "Please check your input and try again.")
    }

    // The interceptor will automatically convert this to gRPC status
    // and ensure metadata is preserved in context
    return s.repo.CreateUser(ctx, req)
}
```

### 6.4 Security

- **Use TLS in production**: Always configure certificates for production
- **Validate certificates**: Ensure proper certificate validation
- **Monitor security events**: Use the built-in logging for security monitoring

```go
// Production TLS configuration
config := client.HTTPSConfig{
    Cert: "/etc/ssl/certs/server.crt",
    Key:  "/etc/ssl/private/server.key",
}

opts, err := grpc.NewServerOptionsAndCreds(
    grpc.WithServiceConfig(config),
    // ... other options
)
```

### 6.5 Observability

- **Enable OpenTelemetry**: For comprehensive tracing and metrics
- **Monitor request patterns**: Use the built-in request logging
- **Track performance**: Leverage the automatic duration tracking
- **Use dual filtering**: Configure both logging and tracing exclusions

```go
// Enable full observability with custom filtering
opts, err := grpc.NewServerOptionsAndCreds(
    grpc.WithSetOTELServerHandler(true),
    grpc.WithServiceName("my-service"),
    grpc.WithServiceVersion("v1.0.0"),
    grpc.WithMethodLogExcludePatterns([]string{
        ".*Health/.*",
        ".*Metrics/.*",
    }),
    grpc.WithMethodTraceExcludePatterns([]string{
        ".*TestService/.*",
        ".*DebugService/.*",
    }),
)
```

## 7. Migration Guide

### 7.1 From Standard gRPC Server

**Before:**

```go
server := grpc.NewServer()
// Manual interceptor setup
// Manual error handling
// Manual logging
```

**After:**

```go
opts, err := grpc.NewServerOptionsAndCreds(
    grpc.WithServiceName("my-service"),
    grpc.WithServiceVersion("v1.0.0"),
    grpc.WithSetOTELServerHandler(true),
)
if err != nil {
    log.Fatal(err)
}

server := grpc.NewServer(opts...)
// Automatic interceptor chain
// Automatic error handling
// Automatic structured logging
```

### 7.2 Adding Custom Interceptors

```go
// Create base options
opts, err := grpc.NewServerOptionsAndCreds(
    grpc.WithServiceName("my-service"),
)

// Add custom interceptors
unaryInterceptors := append([]grpc.UnaryServerInterceptor{
    myCustomInterceptor,
}, opts...)

server := grpc.NewServer(grpc.UnaryInterceptor(unaryInterceptors...))
```

## 8. Performance Considerations

- **Minimal overhead**: Interceptors are optimized for performance
- **Selective logging and tracing**: Use exclusion patterns to reduce observability overhead
- **Efficient error handling**: Error conversion is optimized for common cases
- **Memory efficient**: Context propagation is designed for minimal allocations
- **Dual filtering**: Separate control for logging and tracing reduces unnecessary processing

## 9. Troubleshooting

### 9.1 Common Issues

1. **Missing TLS certificates**: Ensure certificate files exist and are readable
2. **Log volume too high**: Adjust method exclusion patterns
3. **OpenTelemetry not working**: Verify OTEL collector is running and accessible
4. **gRPC-Gateway errors**: Check header matcher configuration
5. **Tracing overhead**: Use trace exclusion patterns to reduce tracing costs

### 9.2 Debug Mode

Enable debug logging to troubleshoot interceptor behavior:

```go
// Set log level to debug
log.SetLevel(zap.DebugLevel)

// Check interceptor chain
opts, err := grpc.NewServerOptionsAndCreds(
    grpc.WithServiceName("debug-service"),
    grpc.WithSetOTELServerHandler(false), // Disable OTEL for debugging
)
```

## 10. Contributing

When adding new functionality:

1. **Follow existing patterns**: Use the established interceptor patterns
2. **Add comprehensive tests**: Include unit tests with minimock for new interceptors
3. **Update documentation**: Keep this README current with new features
4. **Consider performance**: Ensure new interceptors don't impact performance
5. **Add examples**: Provide usage examples for new features

## 11. License

This package is part of the Instill AI x library and follows the same licensing terms.
