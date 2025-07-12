# client

A comprehensive gRPC client framework with built-in interceptors, TLS support, and type-safe client creation.

The `x/client` package provides a production-ready gRPC client setup with comprehensive middleware support, including metadata propagation, TLS/SSL support, and type-safe client creation for the Instill AI platform services. It's designed to work seamlessly with the Instill AI platform and follows best practices for client-side observability and error handling.

## Overview

The `x/client` package provides:

1. **Type-safe gRPC client creation** - Generic client factory with compile-time type safety
2. **Pre-configured client dial options** - Ready-to-use client configuration with sensible defaults
3. **Automatic metadata propagation** - Built-in interceptor for context metadata handling
4. **TLS/SSL support** - Secure communication with certificate-based authentication
5. **OpenTelemetry integration** - Automatic tracing and metrics collection
6. **Service configuration management** - Structured configuration for multi-service environments

## Core Components

### Configuration (`config.go`)

Core configuration structures for client setup and service connections.

#### `HTTPSConfig`

TLS/SSL configuration for secure connections.

```go
type HTTPSConfig struct {
    Cert string `koanf:"cert"` // Path to certificate file
    Key  string `koanf:"key"`  // Path to private key file
}
```

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

### Constants (`constant.go`)

Package-wide constants and configuration values.

```go
const (
    mb = 1024 * 1024 // number of bytes in a megabyte

    // MaxPayloadSize is the maximum size of the payload that gRPC clients allow.
    MaxPayloadSize = 256 * mb // 256MB default payload limit
)
```

### gRPC Client Options (`grpc/options.go`)

Configuration options and dial options creation for gRPC clients.

#### `NewClientDialOptionsAndCreds(options ...ClientOption) ([]grpc.DialOption, credentials.TransportCredentials, error)`

Creates a complete gRPC client configuration with:

- **Metadata propagation interceptor**: Automatic context metadata handling
- **TLS credentials**: Automatic certificate-based security
- **OpenTelemetry integration**: Built-in tracing and metrics
- **Message size limits**: Configurable payload size limits (256MB default)

```go
// Basic usage
dialOpts, creds, err := grpc.NewClientDialOptionsAndCreds(
    grpc.WithHostPort("localhost:8080"),
    grpc.WithOTELCollectorEnable(true),
)
if err != nil {
    log.Fatal(err)
}

conn, err := grpc.Dial("localhost:8080", dialOpts...)
```

#### Configuration Options

```go
// TLS configuration
grpc.WithHTTPSConfig(client.HTTPSConfig{
    Cert: "/path/to/cert.pem",
    Key:  "/path/to/key.pem",
})

// Connection details
grpc.WithHostPort("localhost:8080")

// Observability
grpc.WithOTELCollectorEnable(true)
```

### gRPC Client Factory (`grpc/backends.go`)

Type-safe client creation for Instill AI platform services.

#### `NewClient[T any](clientType ClientType, svc client.ServiceConfig) (T, func() error, error)`

Creates a type-safe gRPC client with automatic connection management.

**Supported Client Types:**

```go
const (
    PipelinePublic  ClientType = "pipeline_public"
    PipelinePrivate ClientType = "pipeline_private"
    ArtifactPublic  ClientType = "artifact_public"
    ArtifactPrivate ClientType = "artifact_private"
    ModelPublic     ClientType = "model_public"
    ModelPrivate    ClientType = "model_private"
    MgmtPublic      ClientType = "mgmt_public"
    MgmtPrivate     ClientType = "mgmt_private"
)
```

**Usage:**

```go
// Create a pipeline public service client
svc := client.ServiceConfig{
    Host:        "localhost",
    PublicPort:  8080,
    PrivatePort: 8081,
    HTTPS:       client.HTTPSConfig{},
}

client, closeFn, err := grpc.NewClient[pipelinepb.PipelinePublicServiceClient](
    grpc.PipelinePublic,
    svc,
)
if err != nil {
    log.Fatal(err)
}
defer closeFn()

// Use the client
response, err := client.CreatePipeline(ctx, request)
```

### Metadata Interceptor (`grpc/interceptor/metadata.go`)

Handles metadata propagation from incoming to outgoing contexts.

#### `MetadataPropagatorInterceptor`

Automatically propagates metadata from incoming context to outgoing gRPC calls.

**Features:**

- **Automatic propagation**: Copies metadata from incoming to outgoing context
- **Smart handling**: Only propagates when outgoing context doesn't already have metadata
- **Graceful fallback**: Continues normally when no metadata is present

```go
// Automatically applied in NewClientDialOptionsAndCreds
// No manual configuration required
```

## API Reference

### Core Functions

#### `grpc.NewClientDialOptionsAndCreds(options ...ClientOption) ([]grpc.DialOption, credentials.TransportCredentials, error)`

Creates gRPC client dial options and credentials.

**Parameters:**

- `options`: Configuration options (see Options section)

**Returns:**

- `[]grpc.DialOption`: Dial options for `grpc.Dial()`
- `credentials.TransportCredentials`: TLS credentials (if configured)
- `error`: Any configuration errors

#### `grpc.NewClient[T any](clientType ClientType, svc client.ServiceConfig) (T, func() error, error)`

Creates a type-safe gRPC client.

**Parameters:**

- `T`: The gRPC client type (e.g., `pipelinepb.PipelinePublicServiceClient`)
- `clientType`: The service type (e.g., `grpc.PipelinePublic`)
- `svc`: Service configuration

**Returns:**

- `T`: The configured gRPC client
- `func() error`: Connection close function
- `error`: Any creation errors

### Configuration Options

#### `grpc.WithHTTPSConfig(config client.HTTPSConfig)`

Sets TLS/SSL configuration.

```go
config := client.HTTPSConfig{
    Cert: "/path/to/cert.pem",
    Key:  "/path/to/key.pem",
}
```

#### `grpc.WithHostPort(hostPort string)`

Sets the host and port for the connection.

```go
grpc.WithHostPort("localhost:8080")
```

#### `grpc.WithOTELCollectorEnable(enable bool)`

Enables or disables OpenTelemetry collector integration.

```go
grpc.WithOTELCollectorEnable(true)
```

## Usage Examples

### Basic Client Setup

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
    // Configure service
    svc := client.ServiceConfig{
        Host:        "localhost",
        PublicPort:  8080,
        PrivatePort: 8081,
        HTTPS:       client.HTTPSConfig{},
    }

    // Create client
    client, closeFn, err := grpc.NewClient[pipelinepb.PipelinePublicServiceClient](
        grpc.PipelinePublic,
        svc,
    )
    if err != nil {
        log.Fatal(err)
    }
    defer closeFn()

    // Use client
    ctx := context.Background()
    response, err := client.ListPipelines(ctx, &pipelinepb.ListPipelinesRequest{})
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("Found %d pipelines", len(response.Pipelines))
}
```

### Client with TLS

```go
// Configure service with TLS
svc := client.ServiceConfig{
    Host:        "secure.example.com",
    PublicPort:  443,
    PrivatePort: 8443,
    HTTPS: client.HTTPSConfig{
        Cert: "/path/to/client.crt",
        Key:  "/path/to/client.key",
    },
}

// Create secure client
client, closeFn, err := grpc.NewClient[pipelinepb.PipelinePublicServiceClient](
    grpc.PipelinePublic,
    svc,
)
if err != nil {
    log.Fatal(err)
}
defer closeFn()
```

### Multiple Service Clients

```go
// Configure multiple services
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

// Create multiple clients
pipelineClient, pipelineClose, err := grpc.NewClient[pipelinepb.PipelinePublicServiceClient](
    grpc.PipelinePublic,
    pipelineSvc,
)
if err != nil {
    log.Fatal(err)
}
defer pipelineClose()

modelClient, modelClose, err := grpc.NewClient[modelpb.ModelPublicServiceClient](
    grpc.ModelPublic,
    modelSvc,
)
if err != nil {
    log.Fatal(err)
}
defer modelClose()
```

### Custom Dial Options

```go
// Create custom dial options
dialOpts, creds, err := grpc.NewClientDialOptionsAndCreds(
    grpc.WithHostPort("localhost:8080"),
    grpc.WithOTELCollectorEnable(true),
    grpc.WithHTTPSConfig(client.HTTPSConfig{
        Cert: "/path/to/cert.pem",
        Key:  "/path/to/key.pem",
    }),
)
if err != nil {
    log.Fatal(err)
}

// Use with custom connection
conn, err := grpc.Dial("localhost:8080", dialOpts...)
if err != nil {
    log.Fatal(err)
}
defer conn.Close()

// Create client manually
client := pipelinepb.NewPipelinePublicServiceClient(conn)
```

### Private Service Access

```go
// Access private service endpoints
svc := client.ServiceConfig{
    Host:        "internal.example.com",
    PublicPort:  8080,
    PrivatePort: 8081, // Use private port
    HTTPS:       client.HTTPSConfig{},
}

// Create private client
privateClient, closeFn, err := grpc.NewClient[pipelinepb.PipelinePrivateServiceClient](
    grpc.PipelinePrivate, // Use private client type
    svc,
)
if err != nil {
    log.Fatal(err)
}
defer closeFn()
```

## Best Practices

### 1. Connection Management

- **Always defer close**: Use the returned close function to properly close connections
- **Handle errors**: Check for connection errors and handle them appropriately
- **Use context**: Pass context to all gRPC calls for proper cancellation

```go
client, closeFn, err := grpc.NewClient[pipelinepb.PipelinePublicServiceClient](
    grpc.PipelinePublic,
    svc,
)
if err != nil {
    log.Fatal(err)
}
defer closeFn() // Always close the connection

ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

response, err := client.CreatePipeline(ctx, request)
```

### 2. Service Configuration

- **Use descriptive hosts**: Use meaningful hostnames for different environments
- **Configure ports properly**: Use public ports for external access, private for internal
- **Enable TLS in production**: Always use TLS for production environments

```go
// Development
svc := client.ServiceConfig{
    Host:        "localhost",
    PublicPort:  8080,
    PrivatePort: 8081,
    HTTPS:       client.HTTPSConfig{}, // No TLS for dev
}

// Production
svc := client.ServiceConfig{
    Host:        "api.production.example.com",
    PublicPort:  443,
    PrivatePort: 8443,
    HTTPS: client.HTTPSConfig{
        Cert: "/etc/ssl/certs/client.crt",
        Key:  "/etc/ssl/private/client.key",
    },
}
```

### 3. Error Handling

- **Check connection errors**: Always handle connection creation errors
- **Use appropriate timeouts**: Set context timeouts for gRPC calls
- **Handle service errors**: Check for service-specific error responses

```go
client, closeFn, err := grpc.NewClient[pipelinepb.PipelinePublicServiceClient](
    grpc.PipelinePublic,
    svc,
)
if err != nil {
    log.Printf("Failed to create client: %v", err)
    return err
}
defer closeFn()

ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

response, err := client.CreatePipeline(ctx, request)
if err != nil {
    log.Printf("Service call failed: %v", err)
    return err
}
```

### 4. Observability

- **Enable OpenTelemetry**: Use OTEL for comprehensive tracing and metrics
- **Monitor connection health**: Check connection state periodically
- **Log important operations**: Log client creation and major operations

```go
// Enable observability
dialOpts, _, err := grpc.NewClientDialOptionsAndCreds(
    grpc.WithOTELCollectorEnable(true),
    grpc.WithHostPort("localhost:8080"),
)
if err != nil {
    log.Fatal(err)
}

// Monitor connection
conn, err := grpc.Dial("localhost:8080", dialOpts...)
if err != nil {
    log.Fatal(err)
}

// Check connection state
state := conn.GetState()
log.Printf("Connection state: %v", state)
```

### 5. Type Safety

- **Use generic client creation**: Leverage the type-safe `NewClient[T]` function
- **Specify correct types**: Use the exact gRPC client type for compile-time safety
- **Handle type assertions**: The factory handles type safety automatically

```go
// Type-safe client creation
client, closeFn, err := grpc.NewClient[pipelinepb.PipelinePublicServiceClient](
    grpc.PipelinePublic,
    svc,
)
if err != nil {
    log.Fatal(err)
}
defer closeFn()

// Compile-time type safety
response, err := client.CreatePipeline(ctx, request) // Type-safe call
```

### 6. Testing

- **Mock external services**: Use gRPC testing utilities for unit tests
- **Test error scenarios**: Verify proper error handling
- **Test connection management**: Ensure connections are properly closed

```go
func TestClient_CreatePipeline(t *testing.T) {
    // Create test server
    server := grpc.NewServer()
    defer server.Stop()

    // Create test client
    client, closeFn, err := grpc.NewClient[pipelinepb.PipelinePublicServiceClient](
        grpc.PipelinePublic,
        testServiceConfig,
    )
    require.NoError(t, err)
    defer closeFn()

    // Test client operations
    response, err := client.CreatePipeline(ctx, testRequest)
    assert.NoError(t, err)
    assert.NotNil(t, response)
}
```

## Migration Guide

### From Manual gRPC Client Creation

**Before:**

```go
// Manual connection setup
conn, err := grpc.Dial("localhost:8080",
    grpc.WithTransportCredentials(insecure.NewCredentials()),
    grpc.WithUnaryInterceptor(myInterceptor),
)
if err != nil {
    log.Fatal(err)
}
defer conn.Close()

// Manual client creation
client := pipelinepb.NewPipelinePublicServiceClient(conn)
```

**After:**

```go
// Automated client creation
svc := client.ServiceConfig{
    Host:        "localhost",
    PublicPort:  8080,
    PrivatePort: 8081,
    HTTPS:       client.HTTPSConfig{},
}

client, closeFn, err := grpc.NewClient[pipelinepb.PipelinePublicServiceClient](
    grpc.PipelinePublic,
    svc,
)
if err != nil {
    log.Fatal(err)
}
defer closeFn()
```

### Adding Custom Interceptors

```go
// Create base dial options
dialOpts, _, err := grpc.NewClientDialOptionsAndCreds(
    grpc.WithHostPort("localhost:8080"),
)

// Add custom interceptors
dialOpts = append(dialOpts,
    grpc.WithUnaryInterceptor(myCustomInterceptor),
)

// Use with connection
conn, err := grpc.Dial("localhost:8080", dialOpts...)
```

## Performance Considerations

- **Connection pooling**: The client factory manages connections efficiently
- **Minimal overhead**: Interceptors are optimized for performance
- **Efficient metadata handling**: Metadata propagation is designed for minimal allocations
- **Memory efficient**: Type-safe client creation avoids runtime type assertions

## Troubleshooting

### Common Issues

1. **Connection failures**: Check host, port, and TLS configuration
2. **Type assertion errors**: Ensure correct client type is specified
3. **TLS certificate issues**: Verify certificate paths and permissions
4. **Metadata propagation**: Check context setup for metadata

### Debug Mode

Enable debug logging to troubleshoot client behavior:

```go
// Set log level to debug
log.SetLevel(zap.DebugLevel)

// Create client with debug information
client, closeFn, err := grpc.NewClient[pipelinepb.PipelinePublicServiceClient](
    grpc.PipelinePublic,
    svc,
)
if err != nil {
    log.Printf("Client creation failed: %v", err)
    return err
}
defer closeFn()
```

## Contributing

When adding new functionality:

1. **Follow existing patterns**: Use the established client factory patterns
2. **Add comprehensive tests**: Include unit tests for new client types
3. **Update documentation**: Keep this README current with new features
4. **Consider performance**: Ensure new features don't impact performance
5. **Add examples**: Provide usage examples for new features

## License

This package is part of the Instill AI x library and follows the same licensing terms.
