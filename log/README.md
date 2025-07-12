# log

Structured logging with Zap and OpenTelemetry integration.

`x/log` provides a comprehensive logging solution that combines the performance and flexibility of Zap with OpenTelemetry tracing integration. It offers colored JSON output, configurable log levels, and seamless integration with Temporal workflows.

## Overview

The `x/log` package provides a structured logging solution that:

1. **High-performance logging** - Built on Zap for optimal performance and structured output
2. **OpenTelemetry integration** - Automatic log injection into traces for observability
3. **Colored output** - Enhanced readability with color-coded log levels
4. **Configurable verbosity** - Debug and production modes with appropriate log levels
5. **Temporal compatibility** - Adapter for Temporal SDK logging interface
6. **Dual output streams** - Info/debug logs to stdout, warnings/errors to stderr

## Core Concepts

### Log Levels and Output Streams

The package uses different output streams based on log severity:

- **stdout**: Debug and Info levels (development mode), Info level only (production mode)
- **stderr**: Warn, Error, and Fatal levels (both modes)

```go
// Debug mode: Debug + Info → stdout, Warn + Error + Fatal → stderr
// Production mode: Info → stdout, Warn + Error + Fatal → stderr
```

### OpenTelemetry Integration

Logs are automatically injected into OpenTelemetry traces when a span is active:

```go
// Logs become span events with severity and message attributes
span.AddEvent("log", trace.WithAttributes(
    attribute.String("log.severity", entry.Level.String()),
    attribute.String("log.message", entry.Message),
))
```

### Colored JSON Output

Log entries are color-coded for enhanced readability:

- **Debug**: Blue (`\x1b[34m`)
- **Info**: Green (`\x1b[32m`)
- **Warn**: Yellow (`\x1b[33m`)
- **Error**: Red (`\x1b[31m`)
- **Fatal**: Magenta (`\x1b[35m`)

## API Reference

### Core Functions

#### `GetZapLogger(ctx context.Context) (*zap.Logger, error)`

Creates and configures a Zap logger with OpenTelemetry integration.

```go
logger, err := log.GetZapLogger(ctx)
if err != nil {
    // Handle error
}
logger.Info("Application started")
```

**Features:**

- Thread-safe singleton initialization
- Automatic OpenTelemetry trace integration
- Configurable log levels based on `Debug` flag
- Colored JSON output
- Caller information

#### `Debug bool`

Global flag that controls logger verbosity:

- `true`: Debug and Info levels enabled
- `false`: Info level only (production mode)

```go
log.Debug = true  // Enable debug logging
logger, _ := log.GetZapLogger(ctx)
logger.Debug("Debug information")  // Will be logged
```

### Encoder Components

#### `ColoredJSONEncoder`

A wrapper around Zap's JSON encoder that adds color to log output.

```go
encoder := log.NewColoredJSONEncoder(zapcore.NewJSONEncoder(config))
```

**Methods:**

- `Clone()` - Creates a copy of the encoder
- `EncodeEntry()` - Encodes log entries with color codes

#### `getJSONEncoderConfig(development bool) zapcore.EncoderConfig`

Creates encoder configuration based on environment:

```go
// Development mode: Full caller information
config := getJSONEncoderConfig(true)

// Production mode: Standard caller information
config := getJSONEncoderConfig(false)
```

### Temporal Integration

#### `ZapAdapter`

Implements Temporal's `log.Logger` interface using Zap.

```go
zapLogger, _ := log.GetZapLogger(ctx)
temporalLogger := log.NewZapAdapter(zapLogger)
```

**Methods:**

- `Debug(msg string, keyvals ...any)` - Log debug message
- `Info(msg string, keyvals ...any)` - Log info message
- `Warn(msg string, keyvals ...any)` - Log warning message
- `Error(msg string, keyvals ...any)` - Log error message
- `With(keyvals ...any) log.Logger` - Create logger with additional fields

## Usage Examples

### Basic Logging

```go
package main

import (
    "context"
    "github.com/instill-ai/x/log"
)

func main() {
    ctx := context.Background()

    // Enable debug mode
    log.Debug = true

    logger, err := log.GetZapLogger(ctx)
    if err != nil {
        panic(err)
    }

    // Different log levels
    logger.Debug("Debug information", zap.String("component", "main"))
    logger.Info("Application started", zap.Int("port", 8080))
    logger.Warn("Deprecated feature used", zap.String("feature", "old_api"))
    logger.Error("Failed to connect", zap.Error(err))
}
```

### OpenTelemetry Tracing and Logging Integration

```go
package service

import (
    "context"
    "github.com/instill-ai/x/log"
    "go.opentelemetry.io/otel"
)

func (s *Service) ProcessRequest(ctx context.Context, req *Request) error {
    // Create a span
    ctx, span := otel.Tracer("service").Start(ctx, "ProcessRequest")
    defer span.End()

    // Get logger with trace context
    logger, _ := log.GetZapLogger(ctx)

    // Logs will be automatically added to the span
    logger.Info("Processing request",
        zap.String("request_id", req.ID),
        zap.String("user_id", req.UserID),
    )

    // Process the request...
    if err := s.validate(req); err != nil {
        logger.Error("Validation failed", zap.Error(err))
        return err
    }

    logger.Info("Request processed successfully")
    return nil
}
```

### Temporal Workflow Integration

```go
package workflow

import (
    "time"
    "github.com/instill-ai/x/log"
    "go.temporal.io/sdk/workflow"
)

func ProcessOrderWorkflow(ctx workflow.Context, order Order) error {
    // Get Temporal logger
    logger := log.NewZapAdapter(log.GetZapLogger(ctx))

    logger.Info("Starting order processing",
        "order_id", order.ID,
        "customer_id", order.CustomerID,
    )

    // Process order steps
    if err := workflow.ExecuteActivity(ctx, ValidateOrder, order).Get(ctx, nil); err != nil {
        logger.Error("Order validation failed", "error", err)
        return err
    }

    logger.Info("Order validated successfully")

    // Continue processing...
    return nil
}
```

### Structured Logging with Fields

```go
func (h *Handler) HandleRequest(ctx context.Context, req *Request) (*Response, error) {
    logger, _ := log.GetZapLogger(ctx)

    // Log request details
    logger.Info("Handling request",
        zap.String("method", req.Method),
        zap.String("path", req.Path),
        zap.String("user_agent", req.UserAgent),
        zap.String("client_ip", req.ClientIP),
    )

    // Process request
    start := time.Now()
    resp, err := h.process(req)
    duration := time.Since(start)

    if err != nil {
        logger.Error("Request failed",
            zap.Error(err),
            zap.Duration("duration", duration),
            zap.String("method", req.Method),
            zap.String("path", req.Path),
        )
        return nil, err
    }

    logger.Info("Request completed",
        zap.Duration("duration", duration),
        zap.Int("status_code", resp.StatusCode),
        zap.String("method", req.Method),
        zap.String("path", req.Path),
    )

    return resp, nil
}
```

### Logger with Context Fields

```go
func (s *Service) ProcessUser(ctx context.Context, userID string) error {
    logger, _ := log.GetZapLogger(ctx)

    // Create logger with user context
    userLogger := logger.With(
        zap.String("user_id", userID),
        zap.String("service", "user_processor"),
    )

    userLogger.Info("Starting user processing")

    // All subsequent logs will include user_id and service fields
    if err := s.validateUser(userID); err != nil {
        userLogger.Error("User validation failed", zap.Error(err))
        return err
    }

    userLogger.Info("User processing completed")
    return nil
}
```

## Best Practices

### 1. Context Usage

- **Always pass context**: Use context to enable OpenTelemetry integration
- **Create spans appropriately**: Create spans for significant operations
- **Propagate context**: Pass context through function calls

```go
// Good
func (s *Service) Process(ctx context.Context, data []byte) error {
    logger, _ := log.GetZapLogger(ctx)
    logger.Info("Processing data", zap.Int("size", len(data)))
    // ...
}

// Avoid
func (s *Service) Process(data []byte) error {
    logger, _ := log.GetZapLogger(context.Background())
    logger.Info("Processing data", zap.Int("size", len(data)))
    // ...
}
```

### 2. Log Level Selection

- **Debug**: Detailed information for debugging
- **Info**: General application flow and state changes
- **Warn**: Unexpected but recoverable situations
- **Error**: Errors that need attention but don't stop the application
- **Fatal**: Critical errors that require immediate attention

```go
// Good log level usage
logger.Debug("Parsing configuration file", zap.String("path", configPath))
logger.Info("Server started", zap.Int("port", port))
logger.Warn("Database connection pool at 80% capacity")
logger.Error("Failed to send notification", zap.Error(err))
logger.Fatal("Cannot bind to port", zap.Int("port", port))
```

### 3. Structured Logging

- **Use structured fields**: Prefer structured fields over string interpolation
- **Include relevant context**: Add fields that help with debugging and monitoring
- **Be consistent**: Use consistent field names across your application

```go
// Good - structured logging
logger.Info("User logged in",
    zap.String("user_id", user.ID),
    zap.String("email", user.Email),
    zap.String("ip_address", req.RemoteAddr),
    zap.String("user_agent", req.UserAgent()),
)

// Avoid - string interpolation
logger.Info(fmt.Sprintf("User %s logged in from %s", user.ID, req.RemoteAddr))
```

### 4. Error Logging

- **Include error details**: Always include the error object
- **Add context**: Provide additional context about the error
- **Use appropriate level**: Use Error level for actual errors, not warnings

```go
// Good error logging
if err := db.Query(query).Scan(&result); err != nil {
    logger.Error("Database query failed",
        zap.Error(err),
        zap.String("query", query),
        zap.String("table", "users"),
        zap.String("operation", "select"),
    )
    return err
}
```

### 5. Performance Considerations

- **Use field types appropriately**: Use specific field types (zap.String, zap.Int) instead of zap.Any when possible
- **Avoid expensive operations**: Don't perform expensive operations in log statements
- **Use conditional logging**: Use log level checks for expensive operations

```go
// Good - conditional logging
if logger.Core().Enabled(zapcore.DebugLevel) {
    logger.Debug("Expensive debug info", zap.String("data", expensiveOperation()))
}

// Avoid - always executed
logger.Debug("Expensive debug info", zap.String("data", expensiveOperation()))
```

### 6. Temporal Integration

- **Use ZapAdapter**: Use the provided adapter for Temporal workflows
- **Include workflow context**: Log workflow-specific information
- **Handle errors appropriately**: Log errors but let Temporal handle retries

```go
func MyWorkflow(ctx workflow.Context, input Input) error {
    logger := log.NewZapAdapter(log.GetZapLogger(ctx))

    logger.Info("Workflow started", "input", input)

    // Workflow logic...
    if err := someOperation(); err != nil {
        logger.Error("Operation failed", "error", err)
        return err // Let Temporal handle retry
    }

    logger.Info("Workflow completed successfully")
    return nil
}
```

## Configuration

### Environment-Based Configuration

```go
// Development environment
log.Debug = true
logger, _ := log.GetZapLogger(ctx)

// Production environment
log.Debug = false
logger, _ := log.GetZapLogger(ctx)
```

### Custom Encoder Configuration

```go
// Custom encoder configuration
config := zap.NewProductionEncoderConfig()
config.EncodeTime = zapcore.ISO8601TimeEncoder
config.EncodeLevel = zapcore.CapitalLevelEncoder

encoder := log.NewColoredJSONEncoder(zapcore.NewJSONEncoder(config))
```

## Migration Guide

### From Standard Logging

**Before:**

```go
import "log"

log.Printf("Processing request: %s", requestID)
log.Printf("Error: %v", err)
```

**After:**

```go
import "github.com/instill-ai/x/log"

logger, _ := log.GetZapLogger(ctx)
logger.Info("Processing request", zap.String("request_id", requestID))
logger.Error("Operation failed", zap.Error(err))
```

### From Other Structured Loggers

**Before:**

```go
import "github.com/sirupsen/logrus"

logrus.WithFields(logrus.Fields{
    "user_id": userID,
    "action":  "login",
}).Info("User logged in")
```

**After:**

```go
import "github.com/instill-ai/x/log"

logger, _ := log.GetZapLogger(ctx)
logger.Info("User logged in",
    zap.String("user_id", userID),
    zap.String("action", "login"),
)
```

### Adding OpenTelemetry Integration

**Before:**

```go
logger.Info("Processing request")
```

**After:**

```go
// Create span
ctx, span := tracer.Start(ctx, "ProcessRequest")
defer span.End()

// Get logger with trace context
logger, _ := log.GetZapLogger(ctx)
logger.Info("Processing request") // Automatically added to span
```

## Performance Considerations

- **Minimal overhead**: Zap provides high-performance logging with minimal overhead
- **Structured output**: JSON output is efficient and machine-readable
- **Color codes**: Color encoding adds minimal overhead
- **OpenTelemetry integration**: Trace integration is lightweight and conditional
- **Memory efficient**: Buffer pooling reduces memory allocations

## Contributing

When adding new features or modifications:

1. **Maintain performance**: Ensure new features don't significantly impact performance
2. **Add tests**: Include comprehensive tests for new functionality
3. **Update documentation**: Keep this README current with new features
4. **Follow patterns**: Use established patterns for consistency
5. **Consider backward compatibility**: Maintain compatibility with existing usage

## License

This package is part of the Instill AI x library and follows the same licensing terms.
