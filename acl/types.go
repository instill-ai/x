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

// PermissionCachePrefix is the Redis key prefix for CheckPermission cache entries.
const PermissionCachePrefix = "acl:perm:"

// ListPermissionsCachePrefix is the Redis key prefix for ListPermissions cache entries.
// Stored separately from CheckPermission because the cache key shape differs:
// CheckPermission is keyed per (user, object, role); ListPermissions is keyed
// per (user, objectType, role) and stores a list of object UIDs.
const ListPermissionsCachePrefix = "acl:list:"

// ReadTupleFilter selects which tuples ReadTuples enumerates. The
// fields map 1:1 onto OpenFGA's ReadRequestTupleKey: any combination
// of object, relation, and user is valid, including a partial filter
// that names only one (e.g. Object="file:<uid>" returns every direct
// grant on that file across every user and relation).
//
// PageSize bounds the per-RPC fetch; ReadTuples internally pages
// through continuation tokens and returns the concatenated result, so
// callers do not have to deal with pagination. Zero / negative values
// fall back to DefaultReadPageSize.
type ReadTupleFilter struct {
	Object   string // FGA object string, e.g. "file:<uid>" or "" for any.
	Relation string // FGA relation, e.g. "viewer" or "" for any.
	User     string // FGA subject string, e.g. "user:<uid>" or "" for any.
	PageSize int32  // Per-RPC page size; 0 → DefaultReadPageSize.
}

// ReadTuple is the materialised representation of an FGA tuple
// returned by ReadTuples. The shape mirrors openfga.Tuple but lives
// in this package so callers do not need to import the OpenFGA proto
// directly.
type ReadTuple struct {
	Object   string
	Relation string
	User     string
}

// DefaultReadPageSize is the per-RPC page size used by ReadTuples
// when the caller leaves PageSize unset. Chosen to balance round-trip
// count against per-response payload size; the OpenFGA Read API
// permits up to 100 by default.
const DefaultReadPageSize int32 = 100

type contextKeyType string

// ContextKeyForceHigherConsistency is a context key that, when set to true,
// forces HIGHER_CONSISTENCY in OpenFGA Check requests regardless of whether
// the user is pinned. This is used when a resource's state recently changed
// (e.g., visibility toggle) and affects ALL callers — including anonymous
// visitors who have no persistent user UID to pin.
const ContextKeyForceHigherConsistency contextKeyType = "acl:force_higher_consistency"
