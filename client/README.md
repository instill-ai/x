# client

A comprehensive gRPC client framework with built-in interceptors, TLS support, and type-safe client creation.

The `x/client` package provides a production-ready gRPC client setup with comprehensive middleware support, including metadata propagation, TLS/SSL support, and type-safe client creation for the Instill AI platform services. It's designed to work seamlessly with the Instill AI platform and follows best practices for client-side observability and error handling.

## Overview

The `x/client` package provides:

1. **Type-safe gRPC client creation** - Generic client factory with compile-time type safety using reflection
2. **Pre-configured client dial options** - Ready-to-use client configuration with sensible defaults
3. **Automatic metadata propagation** - Built-in interceptor for context metadata handling
4. **TLS/SSL support** - Secure communication with certificate-based authentication
5. **OpenTelemetry integration** - Automatic tracing and metrics collection
6. **Service configuration management** - Structured configuration for multi-service environments
7. **Options pattern** - Flexible and extensible client configuration

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

#### `NewClientOptionsAndCreds(options ...Option) ([]grpc.DialOption, error)`

Creates a complete gRPC client configuration with:

- **Metadata propagation interceptors**: Automatic context metadata handling for both unary and streaming calls
- **TLS credentials**: Automatic certificate-based security
- **OpenTelemetry integration**: Built-in tracing and metrics
- **Message size limits**: Configurable payload size limits (256MB default)

```go
// Basic usage
dialOpts, err := grpc.NewClientOptionsAndCreds(
    grpc.WithSetOTELClientHandler(true),
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

// Observability
grpc.WithSetOTELClientHandler(true)
```

### gRPC Client Factory (`grpc/clients.go`)

Type-safe client creation for Instill AI platform services using the options pattern and reflection-based type inference.

#### `NewClient[T any](options ...Option) (T, func() error, error)`

Creates a type-safe gRPC client with automatic connection management using the options pattern. The client type is automatically inferred from the generic type parameter using reflection.

**Supported Client Types:**

The client factory automatically supports all Instill AI platform service clients:

- `pipelinepb.PipelinePublicServiceClient`
- `pipelinepb.PipelinePrivateServiceClient`
- `artifactpb.ArtifactPublicServiceClient`
- `artifactpb.ArtifactPrivateServiceClient`
- `modelpb.ModelPublicServiceClient`
- `modelpb.ModelPrivateServiceClient`
- `mgmtpb.MgmtPublicServiceClient`
- `mgmtpb.MgmtPrivateServiceClient`
- `usagepb.UsageServiceClient`

**Client Options:**

```go
// WithServiceConfig sets the service configuration
func WithServiceConfig(svc client.ServiceConfig) Option

// WithHTTPSConfig sets the HTTPS configuration for TLS
func WithHTTPSConfig(config client.HTTPSConfig) Option

// WithSetOTELClientHandler enables or disables the OTEL client handler
func WithSetOTELClientHandler(enable bool) Option
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
    grpc.WithServiceConfig(svc),
    grpc.WithSetOTELClientHandler(false),
)
if err != nil {
    log.Fatal(err)
}
defer closeFn()

// Use the client
response, err := client.CreatePipeline(ctx, request)
```

### Metadata Interceptor (`grpc/interceptor/metadata.go`)

Handles metadata propagation from incoming to outgoing contexts for both unary and streaming gRPC calls.

#### `UnaryMetadataPropagatorInterceptor`

Automatically propagates metadata from incoming context to outgoing gRPC unary calls.

#### `StreamMetadataPropagatorInterceptor`

Automatically propagates metadata from incoming context to outgoing gRPC stream calls.

**Features:**

- **Automatic propagation**: Copies metadata from incoming to outgoing context
- **Smart handling**: Only propagates when outgoing context doesn't already have metadata
- **Graceful fallback**: Continues normally when no metadata is present
- **Dual support**: Works with both unary and streaming gRPC calls

```go
// Automatically applied in NewClientOptionsAndCreds
// No manual configuration required
```

## API Reference

### Core Functions

#### `grpc.NewClientOptionsAndCreds(options ...Option) ([]grpc.DialOption, error)`

Creates gRPC client dial options.

**Parameters:**

- `options`: Configuration options (see Options section)

**Returns:**

- `[]grpc.DialOption`: Dial options for `grpc.Dial()`
- `error`: Any configuration errors

#### `grpc.NewClient[T any](options ...Option) (T, func() error, error)`

Creates a type-safe gRPC client using the options pattern and reflection-based type inference.

**Parameters:**

- `T`: The gRPC client type (e.g., `pipelinepb.PipelinePublicServiceClient`)
- `options`: Client configuration options

**Returns:**

- `T`: The configured gRPC client
- `func() error`: Connection close function
- `error`: Any creation errors

### Configuration Options

#### `grpc.WithServiceConfig(svc client.ServiceConfig)`

Sets the service configuration.

```go
grpc.WithServiceConfig(client.ServiceConfig{
    Host:        "localhost",
    PublicPort:  8080,
    PrivatePort: 8081,
    HTTPS:       client.HTTPSConfig{},
})
```

#### `grpc.WithHTTPSConfig(config client.HTTPSConfig)`

Sets TLS/SSL configuration.

```go
config := client.HTTPSConfig{
    Cert: "/path/to/cert.pem",
    Key:  "/path/to/key.pem",
}
```

#### `grpc.WithSetOTELClientHandler(enable bool)`

Enables or disables OpenTelemetry collector integration.

```go
grpc.WithSetOTELClientHandler(true)
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

    // Create client using options pattern
    client, closeFn, err := grpc.NewClient[pipelinepb.PipelinePublicServiceClient](
        grpc.WithServiceConfig(svc),
        grpc.WithSetOTELClientHandler(false),
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
    HTTPS:       client.HTTPSConfig{},
}

// Create secure client
client, closeFn, err := grpc.NewClient[pipelinepb.PipelinePublicServiceClient](
    grpc.WithServiceConfig(svc),
    grpc.WithHTTPSConfig(client.HTTPSConfig{
        Cert: "/path/to/client.crt",
        Key:  "/path/to/client.key",
    }),
    grpc.WithSetOTELClientHandler(true),
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

### Usage Service Client

```go
// Create usage service client
svc := client.ServiceConfig{
    Host:        "usage.example.com",
    PublicPort:  8080,
    PrivatePort: 8081,
    HTTPS:       client.HTTPSConfig{},
}

usageClient, closeFn, err := grpc.NewClient[usagepb.UsageServiceClient](
    grpc.WithServiceConfig(svc),
    grpc.WithSetOTELClientHandler(false),
)
if err != nil {
    log.Fatal(err)
}
defer closeFn()
```

### Custom Dial Options

```go
// Create custom dial options
dialOpts, err := grpc.NewClientOptionsAndCreds(
    grpc.WithSetOTELClientHandler(true),
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
    grpc.WithServiceConfig(svc),
    grpc.WithSetOTELClientHandler(false),
)
if err != nil {
    log.Fatal(err)
}
defer closeFn()
```

### Minimal Configuration

```go
// Create client with minimal options (uses defaults)
client, closeFn, err := grpc.NewClient[pipelinepb.PipelinePublicServiceClient](
    grpc.WithServiceConfig(svc),
    // OTEL client handler defaults to false
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
    grpc.WithServiceConfig(svc),
    grpc.WithSetOTELClientHandler(false),
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
    HTTPS:       client.HTTPSConfig{},
}

// Use WithHTTPSConfig for TLS
client, closeFn, err := grpc.NewClient[pipelinepb.PipelinePublicServiceClient](
    grpc.WithServiceConfig(svc),
    grpc.WithHTTPSConfig(client.HTTPSConfig{
        Cert: "/etc/ssl/certs/client.crt",
        Key:  "/etc/ssl/private/client.key",
    }),
    grpc.WithSetOTELClientHandler(true),
)
```

### 3. Error Handling

- **Check connection errors**: Always handle connection creation errors
- **Use appropriate timeouts**: Set context timeouts for gRPC calls
- **Handle service errors**: Check for service-specific error responses

```go
client, closeFn, err := grpc.NewClient[pipelinepb.PipelinePublicServiceClient](
    grpc.WithServiceConfig(svc),
    grpc.WithSetOTELClientHandler(false),
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
client, closeFn, err := grpc.NewClient[pipelinepb.PipelinePublicServiceClient](
    grpc.WithServiceConfig(svc),
    grpc.WithSetOTELClientHandler(true), // Enable OTEL
)
if err != nil {
    log.Fatal(err)
}
defer closeFn()

// Monitor connection
// The client factory handles connection state internally
```

### 5. Type Safety

- **Use generic client creation**: Leverage the type-safe `NewClient[T]` function
- **Specify correct types**: Use the exact gRPC client type for compile-time safety
- **Automatic type inference**: The factory automatically determines client type from the generic parameter

```go
// Type-safe client creation
client, closeFn, err := grpc.NewClient[pipelinepb.PipelinePublicServiceClient](
    grpc.WithServiceConfig(svc),
    grpc.WithSetOTELClientHandler(false),
)
if err != nil {
    log.Fatal(err)
}
defer closeFn()

// Compile-time type safety
response, err := client.CreatePipeline(ctx, request) // Type-safe call
```

### 6. Options Pattern

- **Use required options**: Always provide `WithServiceConfig`
- **Leverage defaults**: Use default values for optional configurations
- **Group related options**: Keep related options together for readability

```go
// Good: Clear and readable
client, closeFn, err := grpc.NewClient[pipelinepb.PipelinePublicServiceClient](
    grpc.WithServiceConfig(svc),
    grpc.WithSetOTELClientHandler(true),
)

// Avoid: Missing required options
client, closeFn, err := grpc.NewClient[pipelinepb.PipelinePublicServiceClient](
    grpc.WithSetOTELClientHandler(true), // Missing service config
)
```

### 7. Testing

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
        grpc.WithServiceConfig(testServiceConfig),
        grpc.WithSetOTELClientHandler(false),
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
// Automated client creation with options pattern
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

### From Old Client Factory API

**Before:**

```go
client, closeFn, err := grpc.NewClient[pipelinepb.PipelinePublicServiceClient](
    grpc.PipelinePublic,
    svc,
    false,
)
```

**After:**

```go
client, closeFn, err := grpc.NewClient[pipelinepb.PipelinePublicServiceClient](
    grpc.WithServiceConfig(svc),
    grpc.WithSetOTELClientHandler(false),
)
```

### Adding Custom Interceptors

```go
// Create base dial options
dialOpts, err := grpc.NewClientOptionsAndCreds(
    grpc.WithSetOTELClientHandler(true),
)
if err != nil {
    log.Fatal(err)
}

// Add custom interceptors
dialOpts = append(dialOpts,
    grpc.WithUnaryInterceptor(myCustomUnaryInterceptor),
    grpc.WithStreamInterceptor(myCustomStreamInterceptor),
)

// Use with connection
conn, err := grpc.Dial("localhost:8080", dialOpts...)
```

## Performance Considerations

- **Connection pooling**: The client factory manages connections efficiently
- **Minimal overhead**: Interceptors are optimized for performance
- **Efficient metadata handling**: Metadata propagation is designed for minimal allocations
- **Memory efficient**: Type-safe client creation avoids runtime type assertions
- **Options validation**: Efficient validation of required options
- **Reflection optimization**: Type inference is optimized for minimal runtime overhead

## Troubleshooting

### Common Issues

1. **Connection failures**: Check host, port, and TLS configuration
2. **Type inference errors**: Ensure correct client type is specified in generic parameter
3. **TLS certificate issues**: Verify certificate paths and permissions
4. **Metadata propagation**: Check context setup for metadata
5. **Missing required options**: Ensure `WithServiceConfig` is provided

### Debug Mode

Enable debug logging to troubleshoot client behavior:

```go
// Set log level to debug
log.SetLevel(zap.DebugLevel)

// Create client with debug information
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

### Validation Errors

The new options pattern provides better error messages for missing or invalid configurations:

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

## Contributing

When adding new functionality:

1. **Follow existing patterns**: Use the established options pattern for new features
2. **Add comprehensive tests**: Include unit tests for new client types and options
3. **Update documentation**: Keep this README current with new features
4. **Consider performance**: Ensure new features don't impact performance
5. **Add examples**: Provide usage examples for new features
6. **Validate options**: Add proper validation for new required options
7. **Update client registry**: Add new client types to the client registry in `clients.go`

## License

This package is part of the Instill AI x library and follows the same licensing terms.
