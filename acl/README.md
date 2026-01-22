# ACL Package

A shared ACL (Access Control List) client for Instill services, providing permission management using OpenFGA.

## Features

- **OpenFGA Integration**: Built on OpenFGA for fine-grained authorization
- **Permission Caching**: Redis-based caching with configurable TTL
- **Read/Write Separation**: Supports separate read and write clients for scalability
- **Read-After-Write Consistency**: Redis-based pinning to ensure consistency
- **Auto-Discovery**: Automatically discovers OpenFGA store and authorization model

## Installation

```go
import "github.com/instill-ai/x/acl"
```

## Usage

### Initialize the Client

```go
// Initialize OpenFGA gRPC clients
writeClient, writeConn := acl.InitOpenFGAClient(ctx, "openfga", 8081, 32)
readClient, readConn := acl.InitOpenFGAClient(ctx, "openfga-replica", 8081, 32)
defer writeConn.Close()
defer readConn.Close()

// Configure the ACL client
cfg := acl.Config{
    Host: "openfga",
    Port: 8081,
    Replica: acl.ReplicaConfig{
        Host:                 "openfga-replica",
        Port:                 8081,
        ReplicationTimeFrame: 5, // seconds
    },
    Cache: acl.CacheConfig{
        Enabled: true,
        TTL:     60, // seconds
    },
}

// Create the ACL client
client := acl.NewClient(writeClient, readClient, redisClient, cfg)
```

### Check Permissions

```go
// Check if current user has reader permission
allowed, err := client.CheckPermission(ctx, "pipeline", pipelineUID, "reader")
if err != nil {
    return err
}
if !allowed {
    return errors.ErrPermissionDenied
}
```

### Set Owner

```go
// Set owner for a resource
err := client.SetOwner(ctx, "pipeline", pipelineUID, "user", userUID)
```

### List Permissions

```go
// List all pipelines the current user can read
pipelineUIDs, err := client.ListPermissions(ctx, "pipeline", "reader", false)
```

### Purge Permissions

```go
// Delete all permissions for a resource (e.g., when deleting the resource)
err := client.Purge(ctx, "pipeline", pipelineUID)
```

## Configuration

### YAML Configuration Example

```yaml
openfga:
  host: openfga
  port: 8081
  replica:
    host: openfga-replica
    port: 8081
    replicationtimeframe: 5
  cache:
    enabled: true
    ttl: 60
```

### Cache Configuration

| Field     | Type | Default | Description               |
| --------- | ---- | ------- | ------------------------- |
| `enabled` | bool | `false` | Enable permission caching |
| `ttl`     | int  | `60`    | Cache TTL in seconds      |

## Cache Key Format

```text
acl:perm:{userType}:{userUID}:{objectType}:{objectUID}:{role}
```

Example: `acl:perm:user:abc123:pipeline:def456:reader`

## Cache Invalidation

The cache is automatically invalidated when:

- `SetOwner` is called
- `Purge` is called
- Any write operation modifies permissions

## Object Types

### CE (Community Edition) - `github.com/instill-ai/x/acl`

Core object types:

- `ObjectTypePipeline` - Pipeline resources
- `ObjectTypeModel` - Model resources
- `ObjectTypeKnowledgeBase` - Knowledge base resources

### EE (Enterprise Edition) - `github.com/instill-ai/x-ee/acl`

Agent-specific object types (in addition to CE types):

- `ObjectTypeChat` - Chat resources
- `ObjectTypeCollection` - Collection resources
- `ObjectTypeProject` - Project resources
- `ObjectTypeRow` - Row resources
- `ObjectTypeColumn` - Column resources
- `ObjectTypeCell` - Cell resources
- `ObjectTypeFile` - File resources
- `ObjectTypeGroup` - Group resources

The EE package re-exports CE types for convenience.

## Roles

Common roles:

- `RoleOwner` - Full ownership
- `RoleAdmin` - Administrative access
- `RoleWriter` - Write access
- `RoleExecutor` - Execute access
- `RoleReader` - Read access
- `RoleMember` - Membership
