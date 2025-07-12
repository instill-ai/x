# errors

Add end-user messages to errors.

`err.Error()` doesn't usually provide a human-friendly output. `x/errors` allows
errors to carry an (extendable) end-user message that can be used in e.g.
handlers.

## Overview

The `x/errors` package provides a comprehensive error handling solution that:

1. **Separates internal errors from user-facing messages** - Internal error details remain for debugging while users see friendly messages
2. **Supports error wrapping and chaining** - Works seamlessly with Go's error wrapping patterns
3. **Provides domain-specific error types** - Predefined errors for common scenarios across different layers
4. **Integrates with gRPC** - Automatic conversion to appropriate gRPC status codes
5. **Handles complex error scenarios** - Supports joined errors and multi-level error chains

## Core Concepts

### End-User Messages vs Internal Errors

The package distinguishes between:
- **Internal errors**: Technical details for developers and debugging
- **End-user messages**: Human-readable messages for API consumers

```go
// Internal error (for developers)
err := fmt.Errorf("failed to call connector vendor: %w", err)

// End-user message (for API consumers)
return errors.AddMessage(err, "Failed to call Vendor API.")
```

### Error Wrapping Support

The package fully supports Go's error wrapping patterns:

```go
// Basic wrapping
err := fmt.Errorf("operation failed: %w", originalErr)
err = errors.AddMessage(err, "Please try again later.")

// Chained wrapping
err := fmt.Errorf("step 1: %w",
    fmt.Errorf("step 2: %w",
        errors.AddMessage(originalErr, "Database connection failed.")
    )
)
```

## API Reference

### Core Functions

#### `AddMessage(err error, msg string) error`
Adds an end-user message to an error. If the error already has a message, the new message is prepended.

```go
err := errors.New("database connection failed")
err = errors.AddMessage(err, "Unable to process your request at this time.")
```

#### `Message(err error) string`
Extracts the end-user message from an error chain. Returns empty string if no message is found.

```go
msg := errors.Message(err)
if msg != "" {
    // Use the user-friendly message
    return msg
}
```

#### `MessageOrErr(err error) string`
Extracts the end-user message, falling back to `err.Error()` if no message exists.

```go
// Always returns a meaningful string
userMsg := errors.MessageOrErr(err)
```

### Domain Errors

The package provides predefined errors for common scenarios:

#### Domain Layer (`domain.go`)
```go
var (
    ErrInvalidArgument = errors.New("invalid")
    ErrNotFound        = errors.New("not found")
    ErrUnauthorized    = errors.New("unauthorized")
    ErrAlreadyExists   = AddMessage(errors.New("resource already exists"), "Resource already exists.")
)
```

#### Service Layer (`service.go`)
```go
var (
    ErrUnauthenticated = errors.New("unauthenticated")
    ErrRateLimiting    = errors.New("rate limiting")
    ErrExceedMaxBatchSize = errors.New("the batch size can not exceed 32")
    ErrTriggerFail     = errors.New("failed to trigger the pipeline")
    ErrCanNotUsePlaintextSecret = AddMessage(
        fmt.Errorf("%w: plaintext value in credential field", ErrInvalidArgument),
        "Plaintext values are forbidden in credential fields. You can create a secret and reference it with the syntax ${secret.my-secret}.",
    )
)
```

#### Repository Layer (`repository.go`)
```go
var (
    ErrOwnerTypeNotMatch = errors.New("owner type not match")
    ErrNoDataDeleted     = errors.New("no data deleted")
    ErrNoDataUpdated     = errors.New("no data updated")
)

func NewPageTokenErr(err error) error {
    return fmt.Errorf("%w: invalid page token: %w", ErrInvalidArgument, err)
}
```

#### Handler Layer (`handler.go`)
```go
var (
    ErrCheckUpdateImmutableFields = errors.New("update immutable fields error")
    ErrCheckOutputOnlyFields      = errors.New("can not contain output only fields")
    ErrCheckRequiredFields        = errors.New("required fields missing")
    ErrFieldMask                  = errors.New("field mask error")
    ErrSematicVersion             = errors.New("not a legal version, should be the format vX.Y.Z or vX.Y.Z-identifiers")
    ErrUpdateMask                 = errors.New("update mask error")
)
```

#### ACL Layer (`acl.go`)
```go
var ErrMembershipNotFound = errors.New("membership not found")
```

### gRPC Integration

#### `ConvertGRPCCode(err error) codes.Code`
Maps domain errors to appropriate gRPC status codes:

```go
code := errors.ConvertGRPCCode(err)
// Returns codes.NotFound, codes.InvalidArgument, etc.
```

#### `ConvertToGRPCError(err error) error`
Converts an error to a gRPC status error with proper message handling:

```go
grpcErr := errors.ConvertToGRPCError(err)
return grpcErr
```

**gRPC Code Mapping:**
- `ErrAlreadyExists` → `codes.AlreadyExists`
- `ErrNotFound`, `ErrNoDataDeleted`, `ErrNoDataUpdated`, `ErrMembershipNotFound` → `codes.NotFound`
- `ErrInvalidArgument` and related validation errors → `codes.InvalidArgument`
- `ErrUnauthorized` → `codes.PermissionDenied`
- `ErrUnauthenticated` → `codes.Unauthenticated`
- `ErrRateLimiting` → `codes.ResourceExhausted`
- Unknown errors → `codes.Unknown`

## Usage Examples

### Basic Usage

```go
package connector

import (
    "fmt"
    "io"
    "github.com/instill-ai/x/errors"
)

func (c *Client) sendReq(reqURL, method, contentType string, data io.Reader) ([]byte, error) {
    res, err := c.HTTPClient.Do(req)
    if err != nil {
        err := fmt.Errorf("failed to call connector vendor: %w", err)
        return nil, errors.AddMessage(err, "Failed to call Vendor API.")
    }

    if res.StatusCode < 200 || res.StatusCode >= 300 {
        err := fmt.Errorf("vendor responded with status code %d", res.StatusCode)
        msg := fmt.Sprintf("Vendor responded with a %d status code.", res.StatusCode)
        return nil, errors.AddMessage(err, msg)
    }

    // ... rest of implementation
}
```

### gRPC Handler Integration

```go
package handler

import (
    "context"
    "github.com/instill-ai/x/errors"
    "google.golang.org/grpc/status"
)

func (h *PublicHandler) DoAction(ctx context.Context, req *pb.DoActionRequest) (*pb.DoActionResponse, error) {
    resp, err := h.triggerActionSteps(ctx, req)
    if err != nil {
        return nil, status.Error(errors.ConvertGRPCCode(err), errors.MessageOrErr(err))
    }

    return resp, nil
}
```

### Complex Error Scenarios

```go
// Handling joined errors
func processMultipleOperations() error {
    var errs []error

    if err := operation1(); err != nil {
        errs = append(errs, errors.AddMessage(err, "Operation 1 failed."))
    }

    if err := operation2(); err != nil {
        errs = append(errs, errors.AddMessage(err, "Operation 2 failed."))
    }

    if len(errs) > 0 {
        joinedErr := errors.Join(errs...)
        return errors.AddMessage(joinedErr, "Multiple operations failed.")
    }

    return nil
}

// The resulting error will have a user message like:
// "Multiple operations failed. Operation 1 failed. Operation 2 failed."
```

### Domain-Specific Error Handling

```go
func (s *Service) CreateUser(ctx context.Context, user *User) error {
    // Check if user already exists
    if exists, _ := s.repo.UserExists(ctx, user.Email); exists {
        return errors.ErrAlreadyExists // Already has user-friendly message
    }

    // Validate user data
    if err := s.validateUser(user); err != nil {
        return errors.AddMessage(err, "Please check your input and try again.")
    }

    // Create user
    if err := s.repo.CreateUser(ctx, user); err != nil {
        return fmt.Errorf("failed to create user: %w", err)
    }

    return nil
}
```

## Best Practices

### 1. Message Composition

- **Be specific but concise**: Provide enough detail for users to understand and act on the error
- **Use consistent language**: Maintain a consistent tone and terminology across your application
- **Avoid technical jargon**: Use user-friendly language instead of technical terms

```go
// Good
errors.AddMessage(err, "Unable to process your request. Please try again in a few minutes.")

// Avoid
errors.AddMessage(err, "HTTP 503 Service Unavailable")
```

### 2. Error Wrapping

- **Wrap errors at each layer**: Add context without losing the original error
- **Use descriptive prefixes**: Help with debugging and error tracing

```go
// Good - adds context at each layer
func (s *Service) ProcessData(data []byte) error {
    if err := s.validateData(data); err != nil {
        return fmt.Errorf("data validation failed: %w", err)
    }

    if err := s.storeData(data); err != nil {
        return fmt.Errorf("data storage failed: %w", err)
    }

    return nil
}
```

### 3. Domain Error Usage

- **Use predefined domain errors**: Leverage the built-in error types for consistency
- **Create custom domain errors**: Add application-specific errors when needed

```go
// Use predefined errors
if user == nil {
    return errors.ErrNotFound
}

if !user.HasPermission(permission) {
    return errors.ErrUnauthorized
}

// Create custom domain errors when needed
var ErrInvalidEmailFormat = errors.AddMessage(
    errors.New("invalid email format"),
    "Please provide a valid email address.",
)
```

### 4. gRPC Integration

- **Use ConvertToGRPCError for handlers**: Ensures proper status codes and messages
- **Handle status errors appropriately**: Check if errors are already gRPC status errors

```go
func (h *Handler) HandleRequest(ctx context.Context, req *Request) (*Response, error) {
    result, err := h.service.Process(ctx, req)
    if err != nil {
        // Automatically handles status code mapping and message extraction
        return nil, errors.ConvertToGRPCError(err)
    }

    return result, nil
}
```

### 5. Testing

- **Test error scenarios**: Ensure your error handling works correctly
- **Verify user messages**: Check that end-user messages are appropriate

```go
func TestService_ProcessData(t *testing.T) {
    c := qt.New(t)

    service := &Service{}

    // Test with invalid data
    err := service.ProcessData(nil)
    c.Assert(err, qt.IsNotNil)

    // Verify user message
    msg := errors.Message(err)
    c.Assert(msg, qt.Contains, "Please check your input")

    // Verify gRPC code
    code := errors.ConvertGRPCCode(err)
    c.Assert(code, qt.Equals, codes.InvalidArgument)
}
```

### 6. Error Propagation

- **Don't lose error context**: Always wrap errors with meaningful context
- **Preserve original errors**: Use `%w` verb for error wrapping

```go
// Good - preserves original error
return fmt.Errorf("failed to process request: %w", err)

// Avoid - loses original error
return errors.New("failed to process request")
```

## Migration Guide

### From Standard Error Handling

**Before:**

```go
func (h *Handler) HandleRequest(req *Request) (*Response, error) {
    result, err := h.service.Process(req)
    if err != nil {
        return nil, status.Error(codes.Internal, err.Error())
    }
    return result, nil
}
```

**After:**

```go
func (h *Handler) HandleRequest(req *Request) (*Response, error) {
    result, err := h.service.Process(req)
    if err != nil {
        return nil, errors.ConvertToGRPCError(err)
    }
    return result, nil
}
```

### Adding User Messages to Existing Code

**Before:**

```go
if err := validateInput(input); err != nil {
    return fmt.Errorf("validation failed: %w", err)
}
```

**After:**

```go
if err := validateInput(input); err != nil {
    err := fmt.Errorf("validation failed: %w", err)
    return errors.AddMessage(err, "Please check your input and try again.")
}
```

## Performance Considerations

- **Minimal overhead**: The package adds minimal runtime overhead
- **Memory efficient**: Error wrapping doesn't create unnecessary allocations
- **Fast message extraction**: Message extraction is optimized for common cases

## Contributing

When adding new error types or functionality:

1. **Follow existing patterns**: Use the established conventions for error definitions
2. **Add appropriate gRPC mappings**: Update `ConvertGRPCCode` for new domain errors
3. **Include tests**: Add comprehensive tests for new functionality
4. **Update documentation**: Keep this README current with new features

## License

This package is part of the Instill AI x library and follows the same licensing terms.
