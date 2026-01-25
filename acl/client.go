package acl

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	openfga "github.com/openfga/api/proto/openfga/v1"

	"github.com/instill-ai/x/constant"
	"github.com/instill-ai/x/resource"

	errorsx "github.com/instill-ai/x/errors"
	logx "github.com/instill-ai/x/log"
)

// Client is the ACL client interface that all services should use.
type Client interface {
	// SetOwner sets the owner of an object.
	SetOwner(ctx context.Context, objectType string, objectUID uuid.UUID, ownerType string, ownerUID uuid.UUID) error
	// Purge deletes all permissions associated with an object.
	Purge(ctx context.Context, objectType string, objectUID uuid.UUID) error
	// CheckPermission verifies if the current user has a specific role for an object.
	CheckPermission(ctx context.Context, objectType string, objectUID uuid.UUID, role string) (bool, error)
	// CheckPublicExecutable checks if an object has public execute permission.
	CheckPublicExecutable(ctx context.Context, objectType string, objectUID uuid.UUID) (bool, error)
	// ListPermissions lists all objects of a type that the current user has a role for.
	ListPermissions(ctx context.Context, objectType string, role string, isPublic bool) ([]uuid.UUID, error)
	// SetResourcePermission sets a permission for a user on any resource type.
	SetResourcePermission(ctx context.Context, objectType string, objectUID uuid.UUID, user, role string, enable bool) error
	// DeleteResourcePermission deletes all permissions for a user on a resource.
	DeleteResourcePermission(ctx context.Context, objectType string, objectUID uuid.UUID, user string) error
	// SetPublicPermission sets public reader/executor permissions on a resource.
	SetPublicPermission(ctx context.Context, objectType string, objectUID uuid.UUID) error
	// DeletePublicPermission deletes public permissions from a resource.
	DeletePublicPermission(ctx context.Context, objectType string, objectUID uuid.UUID) error
	// GetOwner retrieves the owner of a given object.
	GetOwner(ctx context.Context, objectType string, objectUID uuid.UUID) (ownerType string, ownerUID string, err error)
	// CheckLinkPermission checks access through a shareable link/code.
	CheckLinkPermission(ctx context.Context, objectType string, objectUID uuid.UUID, role string, codeHeaderKey string) (bool, error)
	// CheckShareLinkPermission checks if a share link token has the specified permission.
	CheckShareLinkPermission(ctx context.Context, shareToken string, objectType string, objectUID uuid.UUID, relation string) (bool, error)
	// CheckRequesterPermission validates organization impersonation.
	CheckRequesterPermission(ctx context.Context) error
}

// ACLClient implements the Client interface with OpenFGA.
type ACLClient struct {
	writeClient  openfga.OpenFGAServiceClient
	readClient   openfga.OpenFGAServiceClient
	redisClient  *redis.Client
	storeID      string
	modelID      string // Cached authorization model ID - fetched once at startup
	cacheEnabled bool
	cacheTTL     time.Duration
	config       Config
}

// NewClient creates a new ACL client with the given configuration.
// It auto-discovers the OpenFGA store and fetches the latest authorization model.
func NewClient(wc openfga.OpenFGAServiceClient, rc openfga.OpenFGAServiceClient, redisClient *redis.Client, cfg Config) *ACLClient {
	if rc == nil {
		rc = wc
	}

	// Auto-discover the store from OpenFGA
	storeResp, err := wc.ListStores(context.Background(), &openfga.ListStoresRequest{})
	if err != nil {
		panic(fmt.Sprintf("failed to list OpenFGA stores: %v", err))
	}
	if len(storeResp.Stores) == 0 {
		panic("no OpenFGA store found - mgmt-backend migration must run first to create the store")
	}
	if len(storeResp.Stores) > 1 {
		panic(fmt.Sprintf("multiple OpenFGA stores found (%d) - this indicates a configuration problem; there should be exactly one store", len(storeResp.Stores)))
	}

	storeID := storeResp.Stores[0].Id

	// Fetch and cache the authorization model ID at startup
	// This avoids fetching the entire model schema on every permission check
	modelResp, err := wc.ReadAuthorizationModels(context.Background(), &openfga.ReadAuthorizationModelsRequest{
		StoreId: storeID,
	})
	if err != nil {
		panic(fmt.Sprintf("failed to read OpenFGA authorization models: %v", err))
	}
	if len(modelResp.AuthorizationModels) == 0 {
		panic(fmt.Sprintf("no authorization model found in OpenFGA store %s", storeID))
	}
	modelID := modelResp.AuthorizationModels[0].Id

	// Configure permission caching
	cacheEnabled := cfg.Cache.Enabled
	cacheTTL := cfg.Cache.CacheTTLDuration()

	log, _ := logx.GetZapLogger(context.Background())
	log.Info("ACL client initialized",
		zap.String("storeID", storeID),
		zap.String("modelID", modelID),
		zap.Bool("cacheEnabled", cacheEnabled),
		zap.Duration("cacheTTL", cacheTTL),
	)

	return &ACLClient{
		writeClient:  wc,
		readClient:   rc,
		redisClient:  redisClient,
		storeID:      storeID,
		modelID:      modelID,
		cacheEnabled: cacheEnabled,
		cacheTTL:     cacheTTL,
		config:       cfg,
	}
}

// InitOpenFGAClient initializes gRPC connections to OpenFGA server.
func InitOpenFGAClient(ctx context.Context, host string, port int, maxDataSize int) (openfga.OpenFGAServiceClient, *grpc.ClientConn) {
	clientDialOpts := grpc.WithTransportCredentials(insecure.NewCredentials())

	const MB = 1024 * 1024
	clientConn, err := grpc.NewClient(
		fmt.Sprintf("%v:%v", host, port),
		clientDialOpts,
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(maxDataSize*MB),
			grpc.MaxCallSendMsgSize(maxDataSize*MB),
		),
	)
	if err != nil {
		panic(err)
	}

	return openfga.NewOpenFGAServiceClient(clientConn), clientConn
}

// getClient returns the appropriate OpenFGA client based on mode and read-after-write consistency.
func (c *ACLClient) getClient(ctx context.Context, mode Mode) openfga.OpenFGAServiceClient {
	userUID := resource.GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey)

	if c.redisClient == nil {
		return c.writeClient
	}

	if mode == WriteMode {
		// To solve the read-after-write inconsistency problem,
		// we direct the user to read from the primary database for a certain time frame
		if c.config.Replica.ReplicationTimeFrame > 0 {
			_ = c.redisClient.Set(ctx, fmt.Sprintf("db_pin_user:%s:openfga", userUID), time.Now(), time.Duration(c.config.Replica.ReplicationTimeFrame)*time.Second)
		}
		return c.writeClient
	}

	// Check if user is pinned to primary for read-after-write consistency
	if !errors.Is(c.redisClient.Get(ctx, fmt.Sprintf("db_pin_user:%s:openfga", userUID)).Err(), redis.Nil) {
		return c.writeClient // Primary
	}

	if mode == ReadMode && c.readClient != nil {
		return c.readClient // Replica
	}

	return c.writeClient
}

// getAuthorizationModelID returns the cached authorization model ID.
// The model ID is fetched once at startup and cached for the lifetime of the client.
func (c *ACLClient) getAuthorizationModelID(ctx context.Context) (string, error) {
	if c.modelID == "" {
		return "", fmt.Errorf("authorization model ID not initialized")
	}
	return c.modelID, nil
}

// GetModelID returns the cached authorization model ID.
// This is useful for external code that needs to make direct OpenFGA calls.
func (c *ACLClient) GetModelID() string {
	return c.modelID
}

// permissionCacheKey generates a cache key for permission checks.
func permissionCacheKey(userType, userUID, objectType, objectUID, role string) string {
	return fmt.Sprintf("%s%s:%s:%s:%s:%s", PermissionCachePrefix, userType, userUID, objectType, objectUID, role)
}

// invalidateObjectCache invalidates all permission cache entries for a given object.
func (c *ACLClient) invalidateObjectCache(ctx context.Context, objectType string, objectUID string) {
	if !c.cacheEnabled || c.redisClient == nil {
		return
	}

	log, _ := logx.GetZapLogger(ctx)

	// Pattern to match all permission cache entries for this object
	// Cache key format: acl:perm:userType:userUID:objectType:objectUID:role
	// Pattern should match any user and any role for this specific object
	pattern := fmt.Sprintf("%s*:%s:%s:*", PermissionCachePrefix, objectType, objectUID)

	log.Debug("Invalidating permission cache",
		zap.String("objectType", objectType),
		zap.String("objectUID", objectUID),
		zap.String("pattern", pattern),
	)

	var cursor uint64
	var deletedCount int
	for {
		var keys []string
		var err error
		keys, cursor, err = c.redisClient.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			log.Warn("Failed to scan cache keys for invalidation", zap.Error(err), zap.String("pattern", pattern))
			return
		}

		if len(keys) > 0 {
			log.Debug("Found cache keys to delete", zap.Strings("keys", keys))
			if err := c.redisClient.Del(ctx, keys...).Err(); err != nil {
				log.Warn("Failed to delete cache keys", zap.Error(err), zap.Strings("keys", keys))
			} else {
				deletedCount += len(keys)
			}
		}

		if cursor == 0 {
			break
		}
	}

	log.Debug("Permission cache invalidation completed",
		zap.String("objectType", objectType),
		zap.String("objectUID", objectUID),
		zap.Int("deletedKeys", deletedCount),
	)
}

// invalidateUserObjectCache invalidates cache entries for a specific user on a specific object.
// This is used when setting permissions for a specific user to ensure their cached "denied" results
// are immediately invalidated.
func (c *ACLClient) invalidateUserObjectCache(ctx context.Context, user, objectType, objectUID string) {
	if !c.cacheEnabled || c.redisClient == nil {
		return
	}

	log, _ := logx.GetZapLogger(ctx)

	// Pattern to match all permission cache entries for this user on this object
	// Cache key format: acl:perm:userType:userUID:objectType:objectUID:role
	pattern := fmt.Sprintf("%s%s:%s:%s:*", PermissionCachePrefix, user, objectType, objectUID)

	log.Debug("Invalidating user-specific permission cache",
		zap.String("user", user),
		zap.String("objectType", objectType),
		zap.String("objectUID", objectUID),
		zap.String("pattern", pattern),
	)

	var cursor uint64
	var deletedCount int
	for {
		var keys []string
		var err error
		keys, cursor, err = c.redisClient.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			log.Warn("Failed to scan user cache keys for invalidation", zap.Error(err), zap.String("pattern", pattern))
			return
		}

		if len(keys) > 0 {
			log.Debug("Found user cache keys to delete", zap.Strings("keys", keys))
			if err := c.redisClient.Del(ctx, keys...).Err(); err != nil {
				log.Warn("Failed to delete user cache keys", zap.Error(err), zap.Strings("keys", keys))
			} else {
				deletedCount += len(keys)
			}
		}

		if cursor == 0 {
			break
		}
	}

	log.Debug("User permission cache invalidation completed",
		zap.String("user", user),
		zap.String("objectType", objectType),
		zap.String("objectUID", objectUID),
		zap.Int("deletedKeys", deletedCount),
	)
}

// SetOwner sets the owner of a given object.
func (c *ACLClient) SetOwner(ctx context.Context, objectType string, objectUID uuid.UUID, ownerType string, ownerUID uuid.UUID) error {
	log, _ := logx.GetZapLogger(ctx)

	// Normalize ownerType
	ownerType = strings.TrimSuffix(ownerType, "s")

	// Check if the owner already exists
	data, err := c.getClient(ctx, ReadMode).Read(ctx, &openfga.ReadRequest{
		StoreId: c.storeID,
		TupleKey: &openfga.ReadRequestTupleKey{
			User:     fmt.Sprintf("%s:%s", ownerType, ownerUID.String()),
			Relation: "owner",
			Object:   fmt.Sprintf("%s:%s", objectType, objectUID.String()),
		},
	})
	if err != nil {
		return err
	}
	if len(data.Tuples) > 0 {
		return nil
	}

	// Get the latest authorization model ID
	modelID, err := c.getAuthorizationModelID(ctx)
	if err != nil {
		return fmt.Errorf("getting authorization model: %w", err)
	}

	log.Debug("SetOwner: writing new owner tuple",
		zap.String("modelID", modelID),
		zap.String("storeID", c.storeID),
		zap.String("user", fmt.Sprintf("%s:%s", ownerType, ownerUID.String())),
		zap.String("object", fmt.Sprintf("%s:%s", objectType, objectUID.String())),
	)

	// Write the new owner
	_, err = c.getClient(ctx, WriteMode).Write(ctx, &openfga.WriteRequest{
		StoreId:              c.storeID,
		AuthorizationModelId: modelID,
		Writes: &openfga.WriteRequestWrites{
			TupleKeys: []*openfga.TupleKey{
				{
					User:     fmt.Sprintf("%s:%s", ownerType, ownerUID.String()),
					Relation: "owner",
					Object:   fmt.Sprintf("%s:%s", objectType, objectUID.String()),
				},
			},
		},
	})
	if err != nil {
		log.Error("SetOwner: failed to write owner tuple", zap.Error(err))
		return err
	}

	log.Debug("SetOwner: successfully wrote owner tuple")

	// Invalidate permission cache for the object
	c.invalidateObjectCache(ctx, objectType, objectUID.String())

	return nil
}

// Purge deletes all permissions associated with the specified object.
func (c *ACLClient) Purge(ctx context.Context, objectType string, objectUID uuid.UUID) error {
	// Read all tuples related to the specified object
	data, err := c.getClient(ctx, ReadMode).Read(ctx, &openfga.ReadRequest{
		StoreId: c.storeID,
		TupleKey: &openfga.ReadRequestTupleKey{
			Object: fmt.Sprintf("%s:%s", objectType, objectUID),
		},
	})
	if err != nil {
		return err
	}

	// Invalidate permission cache for the object
	c.invalidateObjectCache(ctx, objectType, objectUID.String())

	if len(data.Tuples) == 0 {
		return nil
	}

	// Get the latest authorization model ID
	modelID, err := c.getAuthorizationModelID(ctx)
	if err != nil {
		return fmt.Errorf("getting authorization model: %w", err)
	}

	// Delete each tuple
	for _, tuple := range data.Tuples {
		_, err = c.getClient(ctx, WriteMode).Write(ctx, &openfga.WriteRequest{
			StoreId:              c.storeID,
			AuthorizationModelId: modelID,
			Deletes: &openfga.WriteRequestDeletes{
				TupleKeys: []*openfga.TupleKeyWithoutCondition{
					{
						User:     tuple.Key.User,
						Relation: tuple.Key.Relation,
						Object:   tuple.Key.Object,
					},
				},
			},
		})
		if err != nil {
			return err
		}
	}

	return nil
}

// CheckPermission verifies if the current user has a specific role for an object.
func (c *ACLClient) CheckPermission(ctx context.Context, objectType string, objectUID uuid.UUID, role string) (bool, error) {
	log, _ := logx.GetZapLogger(ctx)

	// Retrieve the user type from the request context headers
	userType := resource.GetRequestSingleHeader(ctx, constant.HeaderAuthTypeKey)
	userUID := ""

	// Determine the user UID based on the user type
	if userType == "user" {
		userUID = resource.GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey)
	} else {
		userUID = resource.GetRequestSingleHeader(ctx, constant.HeaderVisitorUIDKey)
	}

	// Check if the user UID is empty and return an error if it is
	if userUID == "" {
		return false, fmt.Errorf("%w: userUID is empty in check permission", errorsx.ErrUnauthenticated)
	}

	// Check cache first if enabled
	cacheKey := permissionCacheKey(userType, userUID, objectType, objectUID.String(), role)
	if c.cacheEnabled && c.redisClient != nil {
		cachedResult, err := c.redisClient.Get(ctx, cacheKey).Result()
		if err == nil {
			// Cache hit
			allowed := cachedResult == "1"
			log.Debug("CheckPermission cache hit",
				zap.String("cacheKey", cacheKey),
				zap.Bool("allowed", allowed),
			)
			return allowed, nil
		} else if !errors.Is(err, redis.Nil) {
			// Log cache error but continue to OpenFGA
			log.Warn("CheckPermission cache error", zap.Error(err))
		}
	}

	// Get the latest authorization model ID
	modelID, err := c.getAuthorizationModelID(ctx)
	if err != nil {
		return false, fmt.Errorf("getting authorization model: %w", err)
	}

	// Check if user is pinned (recently had permissions changed)
	// If so, we need to use HIGHER_CONSISTENCY to bypass OpenFGA's check query cache
	var consistency openfga.ConsistencyPreference
	isPinned := false
	if c.redisClient != nil {
		pinKey := fmt.Sprintf("db_pin_user:%s:openfga", userUID)
		if !errors.Is(c.redisClient.Get(ctx, pinKey).Err(), redis.Nil) {
			isPinned = true
			consistency = openfga.ConsistencyPreference_HIGHER_CONSISTENCY
		}
	}

	log.Debug("CheckPermission",
		zap.String("userType", userType),
		zap.String("userUID", userUID),
		zap.String("objectType", objectType),
		zap.String("objectUID", objectUID.String()),
		zap.String("role", role),
		zap.String("modelID", modelID),
		zap.String("storeID", c.storeID),
		zap.Bool("isPinned", isPinned),
		zap.String("consistency", consistency.String()),
	)

	// Create a CheckRequest to verify the user's permission
	data, err := c.getClient(ctx, ReadMode).Check(ctx, &openfga.CheckRequest{
		StoreId:              c.storeID,
		AuthorizationModelId: modelID,
		TupleKey: &openfga.CheckRequestTupleKey{
			User:     fmt.Sprintf("%s:%s", userType, userUID),
			Relation: role,
			Object:   fmt.Sprintf("%s:%s", objectType, objectUID.String()),
		},
		Consistency: consistency,
	})
	if err != nil {
		log.Error("CheckPermission failed", zap.Error(err))
		return false, err
	}

	// Cache the result if caching is enabled
	if c.cacheEnabled && c.redisClient != nil {
		cacheValue := "0"
		if data.Allowed {
			cacheValue = "1"
		}
		if err := c.redisClient.Set(ctx, cacheKey, cacheValue, c.cacheTTL).Err(); err != nil {
			log.Warn("CheckPermission failed to cache result", zap.Error(err))
		}
	}

	log.Debug("CheckPermission result", zap.Bool("allowed", data.Allowed))

	return data.Allowed, nil
}

// CheckPublicExecutable checks if an object has public execute permission.
func (c *ACLClient) CheckPublicExecutable(ctx context.Context, objectType string, objectUID uuid.UUID) (bool, error) {
	log, _ := logx.GetZapLogger(ctx)

	// Check cache first if enabled
	cacheKey := permissionCacheKey("user", "*", objectType, objectUID.String(), "executor")
	if c.cacheEnabled && c.redisClient != nil {
		cachedResult, err := c.redisClient.Get(ctx, cacheKey).Result()
		if err == nil {
			return cachedResult == "1", nil
		} else if !errors.Is(err, redis.Nil) {
			log.Warn("CheckPublicExecutable cache error", zap.Error(err))
		}
	}

	// Get the latest authorization model ID
	modelID, err := c.getAuthorizationModelID(ctx)
	if err != nil {
		return false, fmt.Errorf("getting authorization model: %w", err)
	}

	data, err := c.getClient(ctx, ReadMode).Check(ctx, &openfga.CheckRequest{
		StoreId:              c.storeID,
		AuthorizationModelId: modelID,
		TupleKey: &openfga.CheckRequestTupleKey{
			User:     "user:*",
			Relation: "executor",
			Object:   fmt.Sprintf("%s:%s", objectType, objectUID.String()),
		},
	})
	if err != nil {
		return false, err
	}

	// Cache the result if caching is enabled
	if c.cacheEnabled && c.redisClient != nil {
		cacheValue := "0"
		if data.Allowed {
			cacheValue = "1"
		}
		if err := c.redisClient.Set(ctx, cacheKey, cacheValue, c.cacheTTL).Err(); err != nil {
			log.Warn("CheckPublicExecutable failed to cache result", zap.Error(err))
		}
	}

	return data.Allowed, nil
}

// ListPermissions lists all objects of a type that the current user has a role for.
func (c *ACLClient) ListPermissions(ctx context.Context, objectType string, role string, isPublic bool) ([]uuid.UUID, error) {
	userType := resource.GetRequestSingleHeader(ctx, constant.HeaderAuthTypeKey)
	userUIDStr := ""
	if userType == "user" {
		userUIDStr = resource.GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey)
	} else {
		userUIDStr = resource.GetRequestSingleHeader(ctx, constant.HeaderVisitorUIDKey)
	}

	if isPublic {
		userUIDStr = "*"
	}

	// Get the latest authorization model ID
	modelID, err := c.getAuthorizationModelID(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting authorization model: %w", err)
	}

	listObjectsResult, err := c.getClient(ctx, ReadMode).ListObjects(ctx, &openfga.ListObjectsRequest{
		StoreId:              c.storeID,
		AuthorizationModelId: modelID,
		User:                 fmt.Sprintf("%s:%s", userType, userUIDStr),
		Relation:             role,
		Type:                 objectType,
	})
	if err != nil {
		if statusErr, ok := status.FromError(err); ok {
			if statusErr.Code() == codes.Code(openfga.ErrorCode_type_not_found) {
				return []uuid.UUID{}, nil
			}
		}
		return nil, err
	}

	objectUIDs := []uuid.UUID{}
	for _, object := range listObjectsResult.GetObjects() {
		objectUIDs = append(objectUIDs, uuid.FromStringOrNil(strings.Split(object, ":")[1]))
	}

	return objectUIDs, nil
}

// GetStoreID returns the OpenFGA store ID.
func (c *ACLClient) GetStoreID() string {
	return c.storeID
}

// PinUserForConsistency pins the current user to the primary database for read-after-write consistency.
// This ensures that subsequent permission checks use HIGHER_CONSISTENCY mode to bypass OpenFGA's
// check query cache, preventing stale "permission denied" results after permission changes.
// Should be called after any write operation that affects permissions.
func (c *ACLClient) PinUserForConsistency(ctx context.Context) {
	if c.redisClient == nil || c.config.Replica.ReplicationTimeFrame <= 0 {
		return
	}

	userUID := resource.GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey)
	if userUID == "" {
		return
	}

	pinKey := fmt.Sprintf("db_pin_user:%s:openfga", userUID)
	_ = c.redisClient.Set(ctx, pinKey, time.Now(), time.Duration(c.config.Replica.ReplicationTimeFrame)*time.Second)
}

// SetResourcePermission sets a permission for a user on any resource type.
// This is a generic method that can be used for any object type (pipeline, model, knowledgebase, etc.).
// It first deletes any existing permission for the user, then sets the new permission if enable is true.
func (c *ACLClient) SetResourcePermission(ctx context.Context, objectType string, objectUID uuid.UUID, user, role string, enable bool) error {
	// Delete existing permission first
	_ = c.DeleteResourcePermission(ctx, objectType, objectUID, user)

	if enable {
		modelID, err := c.getAuthorizationModelID(ctx)
		if err != nil {
			return fmt.Errorf("getting authorization model: %w", err)
		}

		_, err = c.getClient(ctx, WriteMode).Write(ctx, &openfga.WriteRequest{
			StoreId:              c.storeID,
			AuthorizationModelId: modelID,
			Writes: &openfga.WriteRequestWrites{
				TupleKeys: []*openfga.TupleKey{
					{
						User:     user,
						Relation: role,
						Object:   fmt.Sprintf("%s:%s", objectType, objectUID.String()),
					},
				},
			},
		})
		if err != nil {
			return err
		}
	}

	// Invalidate permission cache for the object (all users)
	c.invalidateObjectCache(ctx, objectType, objectUID.String())

	// Also invalidate user-specific cache to ensure any previously cached "denied"
	// results for this specific user are immediately cleared
	c.invalidateUserObjectCache(ctx, user, objectType, objectUID.String())

	// Pin the subject user to primary for read-after-write consistency.
	// When user A grants permission to user B, we need to pin user B to primary
	// so that user B's subsequent reads see the newly granted permission.
	// The user format is "user:<UUID>" or "group:<UUID>#member"
	if c.redisClient != nil && c.config.Replica.ReplicationTimeFrame > 0 {
		// Extract the UUID from the user string (format: "user:<UUID>" or "group:<UUID>#member")
		parts := strings.SplitN(user, ":", 2)
		if len(parts) == 2 {
			subjectUID := strings.TrimSuffix(parts[1], "#member")
			_ = c.redisClient.Set(ctx, fmt.Sprintf("db_pin_user:%s:openfga", subjectUID), time.Now(), time.Duration(c.config.Replica.ReplicationTimeFrame)*time.Second)
		}
	}

	return nil
}

// DeleteResourcePermission deletes all permissions for a user on any resource type.
// It removes all standard roles (admin, writer, executor, reader) for the specified user.
func (c *ACLClient) DeleteResourcePermission(ctx context.Context, objectType string, objectUID uuid.UUID, user string) error {
	modelID, err := c.getAuthorizationModelID(ctx)
	if err != nil {
		return fmt.Errorf("getting authorization model: %w", err)
	}

	for _, role := range []string{"admin", "writer", "executor", "reader"} {
		_, _ = c.getClient(ctx, WriteMode).Write(ctx, &openfga.WriteRequest{
			StoreId:              c.storeID,
			AuthorizationModelId: modelID,
			Deletes: &openfga.WriteRequestDeletes{
				TupleKeys: []*openfga.TupleKeyWithoutCondition{
					{
						User:     user,
						Relation: role,
						Object:   fmt.Sprintf("%s:%s", objectType, objectUID.String()),
					},
				},
			},
		})
	}

	// Invalidate permission cache for the object (all users)
	c.invalidateObjectCache(ctx, objectType, objectUID.String())

	// Also invalidate user-specific cache to ensure any cached results
	// for this specific user are immediately cleared
	c.invalidateUserObjectCache(ctx, user, objectType, objectUID.String())

	return nil
}

// SetPublicPermission sets public permissions on a resource.
// This grants reader permission to user:* and visitor:*, and executor permission to user:*.
func (c *ACLClient) SetPublicPermission(ctx context.Context, objectType string, objectUID uuid.UUID) error {
	// Set reader permission for user:* and visitor:*
	for _, t := range []string{"user", "visitor"} {
		err := c.SetResourcePermission(ctx, objectType, objectUID, fmt.Sprintf("%s:*", t), "reader", true)
		if err != nil {
			return err
		}
	}

	// Set executor permission for user:*
	err := c.SetResourcePermission(ctx, objectType, objectUID, "user:*", "executor", true)
	if err != nil {
		return err
	}

	return nil
}

// DeletePublicPermission deletes public permissions from a resource.
// This removes reader and executor permissions for user:* and visitor:*.
func (c *ACLClient) DeletePublicPermission(ctx context.Context, objectType string, objectUID uuid.UUID) error {
	for _, t := range []string{"user", "visitor"} {
		err := c.DeleteResourcePermission(ctx, objectType, objectUID, fmt.Sprintf("%s:*", t))
		if err != nil {
			return err
		}
	}
	return nil
}

// GetOwner retrieves the owner of a given object.
// Returns the owner type (user or organization) and owner UID.
func (c *ACLClient) GetOwner(ctx context.Context, objectType string, objectUID uuid.UUID) (ownerType string, ownerUID string, err error) {
	data, err := c.getClient(ctx, ReadMode).Read(ctx, &openfga.ReadRequest{
		StoreId: c.storeID,
		TupleKey: &openfga.ReadRequestTupleKey{
			Relation: "owner",
			Object:   fmt.Sprintf("%s:%s", objectType, objectUID.String()),
		},
	})
	if err != nil {
		return "", "", err
	}

	if len(data.Tuples) == 0 {
		return "", "", fmt.Errorf("no owner found for %s:%s", objectType, objectUID.String())
	}

	// Parse the owner from the tuple (format: "user:uid" or "organization:uid")
	parts := strings.Split(data.Tuples[0].Key.User, ":")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid owner format: %s", data.Tuples[0].Key.User)
	}

	return parts[0], parts[1], nil
}

// CheckLinkPermission checks the access over a resource through a shareable link/code.
// The codeHeaderKey parameter specifies which header to read the share code from.
func (c *ACLClient) CheckLinkPermission(ctx context.Context, objectType string, objectUID uuid.UUID, role string, codeHeaderKey string) (bool, error) {
	code := resource.GetRequestSingleHeader(ctx, codeHeaderKey)
	if code == "" {
		return false, nil
	}

	modelID, err := c.getAuthorizationModelID(ctx)
	if err != nil {
		return false, fmt.Errorf("getting authorization model: %w", err)
	}

	data, err := c.getClient(ctx, ReadMode).Check(ctx, &openfga.CheckRequest{
		StoreId:              c.storeID,
		AuthorizationModelId: modelID,
		TupleKey: &openfga.CheckRequestTupleKey{
			User:     fmt.Sprintf("code:%s", code),
			Relation: role,
			Object:   fmt.Sprintf("%s:%s", objectType, objectUID.String()),
		},
	})
	if err != nil {
		return false, fmt.Errorf("requesting permissions from ACL service: %w", err)
	}

	return data.Allowed, nil
}

// CheckShareLinkPermission checks if a share link token has the specified permission for a resource.
// This is used to authorize anonymous access via share links.
func (c *ACLClient) CheckShareLinkPermission(ctx context.Context, shareToken string, objectType string, objectUID uuid.UUID, relation string) (bool, error) {
	modelID, err := c.getAuthorizationModelID(ctx)
	if err != nil {
		return false, fmt.Errorf("getting authorization model: %w", err)
	}

	data, err := c.getClient(ctx, ReadMode).Check(ctx, &openfga.CheckRequest{
		StoreId:              c.storeID,
		AuthorizationModelId: modelID,
		TupleKey: &openfga.CheckRequestTupleKey{
			User:     fmt.Sprintf("share_link:%s", shareToken),
			Relation: relation,
			Object:   fmt.Sprintf("%s:%s", objectType, objectUID.String()),
		},
	})
	if err != nil {
		if statusErr, ok := status.FromError(err); ok {
			if statusErr.Code() == codes.Code(openfga.ErrorCode_type_not_found) {
				return false, nil
			}
		}
		return false, err
	}

	return data.Allowed, nil
}

// CheckRequesterPermission validates that the authenticated user can make
// requests on behalf of the resource identified by the requester UID.
// This is used for organization impersonation.
func (c *ACLClient) CheckRequesterPermission(ctx context.Context) error {
	authType := resource.GetRequestSingleHeader(ctx, constant.HeaderAuthTypeKey)
	if authType != "user" {
		return fmt.Errorf("%w: unauthenticated user", errorsx.ErrUnauthenticated)
	}

	requester := resource.GetRequestSingleHeader(ctx, constant.HeaderRequesterUIDKey)
	authenticatedUser := resource.GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey)
	if requester == "" || authenticatedUser == requester {
		return nil
	}

	// The only impersonation that's currently implemented is switching to an organization namespace
	isMember, err := c.CheckPermission(ctx, "organization", uuid.FromStringOrNil(requester), "member")
	if err != nil {
		return fmt.Errorf("checking organization membership: %w", err)
	}

	if !isMember {
		return fmt.Errorf("%w: authenticated user doesn't belong to requester organization", errorsx.ErrPermissionDenied)
	}

	return nil
}
