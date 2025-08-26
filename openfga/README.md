# OpenFGA Package

This package provides a unified interface for OpenFGA operations using the OpenFGA Go SDK.

## Features

- Simplified OpenFGA client configuration
- Unified interface for permission operations
- Generic permission checking and management
- Support for batch operations
- Direct SDK client access for migration compatibility

## Usage

### Creating a Client

```go
import (
    openfgax "github.com/instill-ai/x/openfga"
    "go.uber.org/zap"
)

// Create client
client, err := openfgax.NewClient(openfgax.ClientParams{
    Config: openfgax.Config{
        Host: "localhost",
        Port: 8080,
    },
    Logger: logger,
})
if err != nil {
    // handle error
}

// Set store and model IDs
err = client.SetStoreID("your-store-id")
if err != nil {
    // handle error
}

err = client.SetAuthorizationModelID("your-model-id")
if err != nil {
    // handle error
}
```

### Checking Permissions

```go
allowed, err := client.CheckPermission(ctx, openfgax.CheckPermissionRequest{
    User:     "user:alice",
    Relation: "viewer",
    Object:   "document:readme",
})
if err != nil {
    // handle error
}

if allowed {
    // user has permission
}
```

### Writing Permissions

```go
err = client.WritePermission(ctx, openfgax.WritePermissionRequest{
    Writes: []openfgax.TupleKey{
        {
            User:     "user:alice",
            Relation: "viewer",
            Object:   "document:readme",
        },
    },
})
if err != nil {
    // handle error
}
```

### Reading Tuples

```go
tuples, err := client.ReadTuples(ctx, openfgax.ReadTuplesRequest{
    Object: openfgax.PtrString("document:readme"),
})
if err != nil {
    // handle error
}

for _, tuple := range tuples {
    fmt.Printf("User: %s, Relation: %s, Object: %s\n", 
        tuple.Key.User, tuple.Key.Relation, tuple.Key.Object)
}
```

### Listing Objects

```go
objects, err := client.ListObjects(ctx, openfgax.ListObjectsRequest{
    User:     "user:alice",
    Relation: "viewer",
    Type:     "document",
})
if err != nil {
    // handle error
}

for _, object := range objects {
    fmt.Printf("Object: %s\n", object)
}
```

## Configuration

The package uses a simplified configuration structure:

```go
type Config struct {
    Host string `koanf:"host"`
    Port int    `koanf:"port"`
}
```

## Error Handling

The package defines specific errors for common scenarios:

- `ErrStoreNotFound`
- `ErrModelNotFound`
- `ErrInvalidUserType`
- `ErrClientNotSet`
- `ErrStoreIDNotSet`
- `ErrModelIDNotSet`
- `ErrInvalidRequest`

## Migration from Direct SDK Usage

If you're migrating from direct OpenFGA SDK usage, you can access the underlying SDK client:

```go
sdkClient := client.SDKClient()
// Use sdkClient for operations not yet supported by the wrapper
```

## Testing

Run tests with:

```bash
go test ./openfga
```

Note: Some tests require a running OpenFGA server for full integration testing.
