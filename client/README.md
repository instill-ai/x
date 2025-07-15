# client

A comprehensive gRPC client framework with built-in interceptors, TLS support, and type-safe client creation.

The `x/client` package provides a production-ready gRPC client setup with comprehensive middleware support, including metadata propagation, TLS/SSL support, and type-safe client creation for the Instill AI platform services.

## 1. Overview

The `x/client` package provides:

1. **Type-safe gRPC client creation** - Generic client factory with compile-time type safety
2. **Pre-configured client dial options** - Ready-to-use client configuration with sensible defaults
3. **Automatic metadata propagation** - Built-in interceptor for context metadata handling
4. **TLS/SSL support** - Secure communication with certificate-based authentication
5. **OpenTelemetry integration** - Automatic tracing and metrics collection with filtering
6. **Service configuration management** - Structured configuration for multi-service environments
7. **Options pattern** - Flexible and extensible client configuration

## 2. Core Components

### 2.1 Configuration (`config.go`)

Core configuration structures for client setup and service connections.

#### `ServiceConfig`

Complete service configuration including host, ports, and TLS settings.

```go
type ServiceConfig struct {
    Host        string      `koanf:"host"`        // Service hostname
    PublicPort  int         `koanf:"publicport"`  // Public service port
    PrivatePort int         `koanf:"privateport"` // Private service port
    HTTPS       HTTPSConfig `koanf:"https"`       // TLS configuration
}
```

### 2.2 gRPC Client Options (`grpc/options.go`)

Configuration options and dial options creation for gRPC clients.

#### `NewClientOptionsAndCreds(options ...Option) ([]grpc.DialOption, error)`

Creates a complete gRPC client configuration with:

- **Metadata propagation interceptors**: Automatic context metadata handling
- **TLS credentials**: Automatic certificate-based security
- **OpenTelemetry integration**: Built-in tracing and metrics with filtering
- **Message size limits**: Configurable payload size limits (256MB default)

```go
// Basic usage
dialOpts, err := grpc.NewClientOptionsAndCreds(
    grpc.WithSetOTELClientHandler(true),
)
if err != nil {
    log.Fatal(err)
}
```

#### Configuration Options

```go
// Service configuration
grpc.WithServiceConfig(client.ServiceConfig{
    Host:        "localhost",
    PublicPort:  8080,
    PrivatePort: 8081,
    HTTPS:       client.HTTPSConfig{},
})

// Observability
grpc.WithSetOTELClientHandler(true)

// Tracing control
grpc.WithMethodTraceExcludePatterns([]string{
    ".*TestService/.*",
    ".*DebugService/.*",
})
```

### 2.3 Trace Filter Decider

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
opts, err := grpc.NewClientOptionsAndCreds(
    grpc.WithSetOTELClientHandler(true),
    grpc.WithMethodTraceExcludePatterns([]string{
        ".*TestService/.*",
        ".*DebugService/.*",
        ".*HealthService/.*",
    }),
)
```

**Important:** Method exclusion patterns use Go regex syntax. Use `.*` to match any characters (not `*`).

### 2.4 gRPC Client Factory (`grpc/clients.go`)

Type-safe client creation for Instill AI platform services using reflection-based type inference.

#### `NewClient[T any](options ...Option) (T, func() error, error)`

Creates a type-safe gRPC client with automatic connection management.

**Supported Client Types:**

- `pipelinepb.PipelinePublicServiceClient`
- `pipelinepb.PipelinePrivateServiceClient`
- `artifactpb.ArtifactPublicServiceClient`
- `artifactpb.ArtifactPrivateServiceClient`
- `modelpb.ModelPublicServiceClient`
- `modelpb.ModelPrivateServiceClient`
- `mgmtpb.MgmtPublicServiceClient`
- `mgmtpb.MgmtPrivateServiceClient`
- `usagepb.UsageServiceClient`

**Usage:**

```go
svc := client.ServiceConfig{
    Host:        "localhost",
    PublicPort:  8080,
    PrivatePort: 8081,
    HTTPS:       client.HTTPSConfig{},
}

client, closeFn, err := grpc.NewClient[pipelinepb.PipelinePublicServiceClient](
    grpc.WithServiceConfig(svc),
    grpc.WithSetOTELClientHandler(false),
)
if err != nil {
    log.Fatal(err)
}
defer closeFn()
```

### 2.5 Metadata Interceptor (`grpc/interceptor/metadata.go`)

Handles metadata propagation from incoming to outgoing contexts for both unary and streaming gRPC calls.

**Features:**

- **Automatic propagation**: Copies metadata from incoming to outgoing context
- **Smart handling**: Only propagates when outgoing context doesn't already have metadata
- **Dual support**: Works with both unary and streaming gRPC calls

## 3. API Reference

### 3.1 Core Functions

#### `grpc.NewClientOptionsAndCreds(options ...Option) ([]grpc.DialOption, error)`

Creates gRPC client dial options.

#### `grpc.NewClient[T any](options ...Option) (T, func() error, error)`

Creates a type-safe gRPC client using reflection-based type inference.

### 3.2 Configuration Options

#### `grpc.WithServiceConfig(svc client.ServiceConfig)`

Sets the service configuration.

#### `grpc.WithSetOTELClientHandler(enable bool)`

Enables or disables OpenTelemetry collector integration.

#### `grpc.WithMethodTraceExcludePatterns(patterns []string)`

Sets custom method exclusion patterns for tracing.

## 4. Usage Examples

### 4.1 Basic Client Setup

```go
package main

import (
    "context"
    "log"

    "github.com/instill-ai/x/client"
    "github.com/instill-ai/x/client/grpc"
    pipelinepb "github.com/instill-ai/protogen-go/pipeline/pipeline/v1beta"
)

func main() {
    svc := client.ServiceConfig{
        Host:        "localhost",
        PublicPort:  8080,
        PrivatePort: 8081,
        HTTPS:       client.HTTPSConfig{},
    }

    client, closeFn, err := grpc.NewClient[pipelinepb.PipelinePublicServiceClient](
        grpc.WithServiceConfig(svc),
        grpc.WithSetOTELClientHandler(false),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer closeFn()

    ctx := context.Background()
    response, err := client.ListPipelines(ctx, &pipelinepb.ListPipelinesRequest{})
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("Found %d pipelines", len(response.Pipelines))
}
```

### 4.2 Client with TLS and Tracing

```go
svc := client.ServiceConfig{
    Host:        "secure.example.com",
    PublicPort:  443,
    PrivatePort: 8443,
    HTTPS:       client.HTTPSConfig{
        Cert: "/path/to/cert.pem",
        Key:  "/path/to/key.pem",
    },
}

client, closeFn, err := grpc.NewClient[pipelinepb.PipelinePublicServiceClient](
    grpc.WithServiceConfig(svc),
    grpc.WithSetOTELClientHandler(true),
    grpc.WithMethodTraceExcludePatterns([]string{
        ".*TestService/.*",
        ".*DebugService/.*",
    }),
)
if err != nil {
    log.Fatal(err)
}
defer closeFn()
```

### 4.3 Multiple Service Clients

```go
pipelineSvc := client.ServiceConfig{
    Host:        "pipeline.example.com",
    PublicPort:  8080,
    PrivatePort: 8081,
    HTTPS:       client.HTTPSConfig{},
}

modelSvc := client.ServiceConfig{
    Host:        "model.example.com",
    PublicPort:  8080,
    PrivatePort: 8081,
    HTTPS:       client.HTTPSConfig{},
}

pipelineClient, pipelineClose, err := grpc.NewClient[pipelinepb.PipelinePublicServiceClient](
    grpc.WithServiceConfig(pipelineSvc),
    grpc.WithSetOTELClientHandler(false),
)
if err != nil {
    log.Fatal(err)
}
defer pipelineClose()

modelClient, modelClose, err := grpc.NewClient[modelpb.ModelPublicServiceClient](
    grpc.WithServiceConfig(modelSvc),
    grpc.WithSetOTELClientHandler(false),
)
if err != nil {
    log.Fatal(err)
}
defer modelClose()
```

### 4.4 Custom Dial Options

```go
dialOpts, err := grpc.NewClientOptionsAndCreds(
    grpc.WithSetOTELClientHandler(true),
    grpc.WithMethodTraceExcludePatterns([]string{
        ".*Health/.*",
        ".*Metrics/.*",
    }),
)
if err != nil {
    log.Fatal(err)
}

conn, err := grpc.Dial("localhost:8080", dialOpts...)
if err != nil {
    log.Fatal(err)
}
defer conn.Close()

client := pipelinepb.NewPipelinePublicServiceClient(conn)
```

## 5. Best Practices

### 5.1 Connection Management

- **Always defer close**: Use the returned close function to properly close connections
- **Handle errors**: Check for connection errors and handle them appropriately
- **Use context**: Pass context to all gRPC calls for proper cancellation

### 5.2 Service Configuration

- **Use descriptive hosts**: Use meaningful hostnames for different environments
- **Configure ports properly**: Use public ports for external access, private for internal
- **Enable TLS in production**: Always use TLS for production environments

### 5.3 Observability

- **Enable OpenTelemetry**: Use OTEL for comprehensive tracing and metrics
- **Use trace filtering**: Configure trace exclusion patterns to reduce overhead
- **Monitor connection health**: Check connection state periodically

```go
// Enable observability with filtering
client, closeFn, err := grpc.NewClient[pipelinepb.PipelinePublicServiceClient](
    grpc.WithServiceConfig(svc),
    grpc.WithSetOTELClientHandler(true),
    grpc.WithMethodTraceExcludePatterns([]string{
        ".*Health/.*",
        ".*Metrics/.*",
        ".*TestService/.*",
    }),
)
```

### 5.4 Type Safety

- **Use generic client creation**: Leverage the type-safe `NewClient[T]` function
- **Specify correct types**: Use the exact gRPC client type for compile-time safety
- **Automatic type inference**: The factory automatically determines client type from the generic parameter

### 5.5 Options Pattern

- **Use required options**: Always provide `WithServiceConfig`
- **Leverage defaults**: Use default values for optional configurations
- **Group related options**: Keep related options together for readability

## 6. Testing

The client package uses **minimock** for unit testing with generated mocks.

### 6.1 Mock Generation

```bash
cd mock && go generate ./generator.go
```

### 6.2 Unit Testing

```go
func TestNewClient_WithMocks(t *testing.T) {
    qt := quicktest.New(t)
    mc := minimock.NewController(t)

    mockConnManager := mockclient.NewConnectionManagerMock(mc)
    mockConnManager.NewConnectionMock.Expect("localhost", 8080, mockclient.HTTPSConfig{}, false).Return(nil, nil)

    client, closeFn, err := grpc.NewClient[pipelinepb.PipelinePublicServiceClient](
        grpc.WithServiceConfig(testServiceConfig),
    )

    qt.Check(err, quicktest.IsNil)
    defer closeFn()
}
```

### 6.3 Integration Testing

```go
func TestNewClient_Integration(t *testing.T) {
    qt := quicktest.New(t)

    server, port := createTestGRPCServer(t)
    defer server.Stop()

    client, closeFn, err := grpc.NewClient[pipelinepb.PipelinePublicServiceClient](
        grpc.WithServiceConfig(client.ServiceConfig{Host: "localhost", PublicPort: port}),
    )

    qt.Check(err, quicktest.IsNil)
    defer closeFn()
}
```

## 7. Migration Guide

### 7.1 From Manual gRPC Client Creation

**Before:**

```go
conn, err := grpc.Dial("localhost:8080",
    grpc.WithTransportCredentials(insecure.NewCredentials()),
    grpc.WithUnaryInterceptor(myInterceptor),
)
if err != nil {
    log.Fatal(err)
}
defer conn.Close()

client := pipelinepb.NewPipelinePublicServiceClient(conn)
```

**After:**

```go
svc := client.ServiceConfig{
    Host:        "localhost",
    PublicPort:  8080,
    PrivatePort: 8081,
    HTTPS:       client.HTTPSConfig{},
}

client, closeFn, err := grpc.NewClient[pipelinepb.PipelinePublicServiceClient](
    grpc.WithServiceConfig(svc),
    grpc.WithSetOTELClientHandler(false),
)
if err != nil {
    log.Fatal(err)
}
defer closeFn()
```

### 7.2 Adding Custom Interceptors

```go
dialOpts, err := grpc.NewClientOptionsAndCreds(
    grpc.WithSetOTELClientHandler(true),
)
if err != nil {
    log.Fatal(err)
}

dialOpts = append(dialOpts,
    grpc.WithUnaryInterceptor(myCustomUnaryInterceptor),
    grpc.WithStreamInterceptor(myCustomStreamInterceptor),
)

conn, err := grpc.Dial("localhost:8080", dialOpts...)
```

## 8. Performance Considerations

- **Connection pooling**: The client factory manages connections efficiently
- **Minimal overhead**: Interceptors are optimized for performance
- **Efficient metadata handling**: Metadata propagation is designed for minimal allocations
- **Memory efficient**: Type-safe client creation avoids runtime type assertions
- **Options validation**: Efficient validation of required options
- **Reflection optimization**: Type inference is optimized for minimal runtime overhead

## 9. Troubleshooting

### 9.1 Common Issues

1. **Connection failures**: Check host, port, and TLS configuration
2. **Type inference errors**: Ensure correct client type is specified in generic parameter
3. **TLS certificate issues**: Verify certificate paths and permissions
4. **Missing required options**: Ensure `WithServiceConfig` is provided

### 9.2 Debug Mode

```go
log.SetLevel(zap.DebugLevel)

client, closeFn, err := grpc.NewClient[pipelinepb.PipelinePublicServiceClient](
    grpc.WithServiceConfig(svc),
    grpc.WithSetOTELClientHandler(false),
)
if err != nil {
    log.Printf("Client creation failed: %v", err)
    return err
}
defer closeFn()
```

### 9.3 Validation Errors

```go
// Missing service config
client, closeFn, err := grpc.NewClient[pipelinepb.PipelinePublicServiceClient](
    grpc.WithSetOTELClientHandler(false),
)
// Error: "service config is required"

// Unsupported client type
type UnsupportedClient interface {
    SomeMethod()
}

client, closeFn, err := grpc.NewClient[UnsupportedClient](
    grpc.WithServiceConfig(svc),
    grpc.WithSetOTELClientHandler(false),
)
// Error: "unsupported client type: UnsupportedClient"
```

## 10. Contributing

When adding new functionality:

1. **Follow existing patterns**: Use the established options pattern for new features
2. **Add comprehensive tests**: Include unit tests for new client types and options
3. **Update documentation**: Keep this README current with new features
4. **Consider performance**: Ensure new features don't impact performance
5. **Add examples**: Provide usage examples for new features
6. **Validate options**: Add proper validation for new required options
7. **Update client registry**: Add new client types to the client registry in `clients.go`

## 11. License

This package is part of the Instill AI x library and follows the same licensing terms.
