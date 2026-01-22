package acl

import (
	"github.com/gofrs/uuid"
)

// Mode represents the database operation mode.
type Mode string

const (
	// ReadMode indicates a read operation.
	ReadMode Mode = "read"
	// WriteMode indicates a write operation.
	WriteMode Mode = "write"
)

// Relation represents a permission relation between a user and an object.
type Relation struct {
	UID      uuid.UUID
	Relation string
}

// OwnerType represents the type of owner (user or organization).
type OwnerType string

const (
	// OwnerTypeUser indicates the owner is a user.
	OwnerTypeUser OwnerType = "user"
	// OwnerTypeOrganization indicates the owner is an organization.
	OwnerTypeOrganization OwnerType = "organization"
)

// ObjectType represents the type of object being protected by ACL.
type ObjectType string

// CE (Community Edition) object types - core resources
const (
	// ObjectTypePipeline represents a pipeline resource.
	ObjectTypePipeline ObjectType = "pipeline"
	// ObjectTypeModel represents a model resource.
	ObjectTypeModel ObjectType = "model"
	// ObjectTypeKnowledgeBase represents a knowledge base resource.
	ObjectTypeKnowledgeBase ObjectType = "knowledgebase"
)

// Role represents a permission role.
type Role string

const (
	// RoleOwner represents owner permission.
	RoleOwner Role = "owner"
	// RoleAdmin represents admin permission.
	RoleAdmin Role = "admin"
	// RoleWriter represents write permission.
	RoleWriter Role = "writer"
	// RoleExecutor represents execute permission.
	RoleExecutor Role = "executor"
	// RoleReader represents read permission.
	RoleReader Role = "reader"
	// RoleMember represents membership.
	RoleMember Role = "member"
)

// PermissionCachePrefix is the Redis key prefix for permission cache entries.
const PermissionCachePrefix = "acl:perm:"
