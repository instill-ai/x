package acl

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	"google.golang.org/protobuf/types/known/wrapperspb"

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
	ListPermissions(ctx context.Context, objectType string, role string) ([]uuid.UUID, error)
	// ListPublicPermissions lists all objects of a type that are readable by
	// everyone (tuples keyed to the FGA wildcard `user:*`).
	ListPublicPermissions(ctx context.Context, objectType string, role string) ([]uuid.UUID, error)
	// ReadTuples enumerates direct-grant tuples that match the given
	// filter. Backed by the OpenFGA Read API, which is indexed,
	// deadline-free, and not subject to the listObjectsDeadline /
	// listObjectsMaxResults truncation that affects ListPermissions.
	// Use this for "list everyone with direct access to X" / "list
	// everything X has direct access to" lookups; use ListPermissions
	// when you need transitively reachable objects via FGA rewrites.
	ReadTuples(ctx context.Context, filter ReadTupleFilter) ([]ReadTuple, error)
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
	// CheckPermissionWithShareLink runs both an identity-scoped check
	// and, when shareToken is non-empty, a share_link-scoped check, and
	// returns true if EITHER grants access. See the method doc for the
	// full contract and rationale.
	CheckPermissionWithShareLink(ctx context.Context, objectType string, objectUID uuid.UUID, relation string, shareToken string) (bool, error)
	// CheckRequesterPermission validates organization impersonation.
	CheckRequesterPermission(ctx context.Context) error
	// IsUserPinned checks if the user is currently pinned to the primary database for read-after-write consistency.
	// This is used to bypass caches and use HIGHER_CONSISTENCY mode in OpenFGA queries.
	IsUserPinned(ctx context.Context) bool
}

// ACLClient implements the Client interface with OpenFGA.
type ACLClient struct {
	writeClient            openfga.OpenFGAServiceClient
	readClient             openfga.OpenFGAServiceClient
	redisClient            *redis.Client
	storeID                string
	modelID                string // Cached authorization model ID - fetched once at startup
	cacheEnabled           bool   // Controls CheckPermission Redis cache
	listPermissionsCacheOn bool   // Controls ListPermissions / ListPublicPermissions Redis cache
	cacheTTL               time.Duration
	listObjectsCfg         ListObjectsConfig // Truncation-guard thresholds for StreamedListObjects
	config                 Config
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
	listPermissionsCacheOn := cfg.Cache.ListPermissionsEnabled
	cacheTTL := cfg.Cache.CacheTTLDuration()
	listObjectsCfg := cfg.ListObjects.resolved()

	log, _ := logx.GetZapLogger(context.Background())
	log.Info("ACL client initialized",
		zap.String("storeID", storeID),
		zap.String("modelID", modelID),
		zap.Bool("checkPermissionCacheEnabled", cacheEnabled),
		zap.Bool("listPermissionsCacheEnabled", listPermissionsCacheOn),
		zap.Duration("cacheTTL", cacheTTL),
		zap.Duration("listObjectsDeadline", listObjectsCfg.Deadline),
		zap.Int("listObjectsMaxResults", listObjectsCfg.MaxResults),
		zap.Duration("listObjectsSlack", listObjectsCfg.Slack),
	)

	return &ACLClient{
		writeClient:            wc,
		readClient:             rc,
		redisClient:            redisClient,
		storeID:                storeID,
		modelID:                modelID,
		cacheEnabled:           cacheEnabled,
		listPermissionsCacheOn: listPermissionsCacheOn,
		cacheTTL:               cacheTTL,
		listObjectsCfg:         listObjectsCfg,
		config:                 cfg,
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

// listPermissionsCacheKey generates a Redis key for caching ListPermissions results.
// The key encodes the caller identity (userType + UID), the object type, and the
// relation so that different users / roles never collide.
func listPermissionsCacheKey(userType, userUID, objectType, role string) string {
	return fmt.Sprintf("%s%s:%s:%s:%s", ListPermissionsCachePrefix, userType, userUID, objectType, role)
}

// invalidateListPermissionsCache deletes all ListPermissions cache entries that
// match the given pattern. This is called from every FGA write path so that
// cached object-UID lists are refreshed after permission changes.
//
// Pattern examples:
//
//	"acl:list:user:abc-123:*"       — all list entries for a specific user
//	"acl:list:*:file:*"             — all list entries for object type "file"
//	"acl:list:*"                    — everything (nuclear option, not used)
func (c *ACLClient) invalidateListPermissionsCache(ctx context.Context, pattern string) {
	if !c.listPermissionsCacheOn || c.redisClient == nil {
		return
	}

	log, _ := logx.GetZapLogger(ctx)
	log.Debug("Invalidating ListPermissions cache", zap.String("pattern", pattern))

	var cursor uint64
	var deletedCount int
	for {
		var keys []string
		var err error
		keys, cursor, err = c.redisClient.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			log.Warn("Failed to scan list-permissions cache keys", zap.Error(err), zap.String("pattern", pattern))
			return
		}
		if len(keys) > 0 {
			if err := c.redisClient.Del(ctx, keys...).Err(); err != nil {
				log.Warn("Failed to delete list-permissions cache keys", zap.Error(err), zap.Strings("keys", keys))
			} else {
				deletedCount += len(keys)
			}
		}
		if cursor == 0 {
			break
		}
	}

	if deletedCount > 0 {
		log.Debug("ListPermissions cache invalidation completed",
			zap.String("pattern", pattern),
			zap.Int("deletedKeys", deletedCount),
		)
	}
}

// invalidateListPermissionsCacheForUser invalidates all ListPermissions cache
// entries for the given FGA user string (e.g. "user:abc-123" or "user:*").
func (c *ACLClient) invalidateListPermissionsCacheForUser(ctx context.Context, user string) {
	pattern := fmt.Sprintf("%s%s:*", ListPermissionsCachePrefix, user)
	c.invalidateListPermissionsCache(ctx, pattern)
}

// invalidateListPermissionsCacheForObjectType invalidates all ListPermissions
// cache entries for a given object type across all users. This is used when a
// public permission change (user:* / visitor:*) alters the FGA graph for every
// caller.
func (c *ACLClient) invalidateListPermissionsCacheForObjectType(ctx context.Context, objectType string) {
	pattern := fmt.Sprintf("%s*:%s:*", ListPermissionsCachePrefix, objectType)
	c.invalidateListPermissionsCache(ctx, pattern)
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
	c.invalidateListPermissionsCacheForUser(ctx, fmt.Sprintf("%s:%s", ownerType, ownerUID.String()))

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
	c.invalidateListPermissionsCacheForObjectType(ctx, objectType)

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

// resolveACLSubject derives the OpenFGA subject (type, UID) from the
// incoming request headers. Every ACL read path should funnel through
// this helper so that the "who is asking?" decision is made in exactly
// one place.
//
// Why this exists
// ---------------
// The earlier in-line logic selected the subject by branching on
// `Instill-Auth-Type` alone:
//
//	if authType == "user" { uid = userUIDHeader } else { uid = visitorUIDHeader }
//
// That encodes an invariant that stopped holding once dual-auth (a
// signed-in reader viewing a `/r/{token}` share link) became real:
// the gateway stamps `Instill-Auth-Type: capability` together with a
// non-empty `Instill-User-Uid` (and no visitor UID, because visitor
// UIDs are only minted when the user UID is empty — see
// api-gateway-ee's authenticateCapabilityToken). The else-branch then
// read an empty visitor UID and every ACL check surfaced as
// `unauthenticated: userUID is empty in check permission`, 401'ing
// core flows such as `CreateChat` on `/r/{token}`.
//
// Selection rules
// ---------------
//
//  1. If `Instill-User-Uid` is non-empty, the caller is an
//     authenticated user. The FGA subject is `user:{uuid}` regardless
//     of `Instill-Auth-Type`, because all authenticated tuples in the
//     store are keyed that way — emitting `capability:{uuid}` would
//     never match a stored tuple and silently deny access.
//
//  2. Otherwise, if `Instill-Auth-Type` is a visitor-shaped label
//     (`visitor` or `capability`) and `Instill-Visitor-Uid` is set,
//     the caller is an anonymous visitor. Both labels collapse to
//     FGA subject type `visitor` because:
//     - The identity (the browser cookie visitor UID) is the same
//     in both cases — the only difference is whether the request
//     also carries a share-link capability token.
//     - Per-resource share-link grants live in `share_link:{token}`
//     tuples, not in `capability:{uid}` tuples; no backend code
//     writes `capability:<...>` tuples, so emitting the label
//     would produce an FGA subject with no possible match.
//     - `capability` is an auth-mechanism signal ("how did this
//     request authenticate?"), not an identity class ("who is
//     asking?"). Keeping the two orthogonal avoids conflating
//     token-holders with anonymous visitors in the FGA schema.
//     Callers that need to honour a capability token must read
//     `Instill-Capability-Token-Uid` and call CheckShareLinkPermission
//     explicitly.
//
//  3. Otherwise, no usable identity was found. Returning an error
//     here preserves the long-standing "empty subject = unauthenticated"
//     contract that callers rely on for 401 mapping.
//
// The returned `userType` is safe to use for both the FGA `User`
// tuple field and the permission cache key without further munging.
func resolveACLSubject(ctx context.Context) (userType, userUID string, err error) {
	if uid := resource.GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey); uid != "" {
		// Rule (1): authenticated user. Always FGA-subject-type "user",
		// even if the request also carries a capability/visitor label
		// (dual-auth on a shared-link reader, etc.).
		return "user", uid, nil
	}

	authType := resource.GetRequestSingleHeader(ctx, constant.HeaderAuthTypeKey)
	switch authType {
	case "visitor", "capability":
		visitorUID := resource.GetRequestSingleHeader(ctx, constant.HeaderVisitorUIDKey)
		if visitorUID == "" {
			return "", "", fmt.Errorf("%w: userUID is empty in check permission", errorsx.ErrUnauthenticated)
		}
		// Rule (2): collapse both labels to `visitor`. The FGA schema
		// only defines `user` and `visitor` identity types; capability
		// tokens are handled separately via CheckShareLinkPermission.
		return "visitor", visitorUID, nil
	}

	// Rule (3): no recognizable identity header. The upstream "user"
	// branch lost its user-UID header before reaching us (bug in the
	// gateway or a mis-forwarded gRPC-to-gRPC hop) — fail closed.
	return "", "", fmt.Errorf("%w: userUID is empty in check permission", errorsx.ErrUnauthenticated)
}

// CheckPermission verifies if the current user has a specific role for an object.
func (c *ACLClient) CheckPermission(ctx context.Context, objectType string, objectUID uuid.UUID, role string) (bool, error) {
	log, _ := logx.GetZapLogger(ctx)

	userType, userUID, err := resolveACLSubject(ctx)
	if err != nil {
		return false, err
	}

	// Determine whether to bypass caches and use HIGHER_CONSISTENCY.
	// Two triggers: (1) caller set ContextKeyForceHigherConsistency (object-level,
	// e.g. after a visibility toggle that affects all users including anonymous),
	// or (2) user is pinned via Redis (per-user read-after-write).
	var consistency openfga.ConsistencyPreference
	forceConsistency := false
	if forceHC, ok := ctx.Value(ContextKeyForceHigherConsistency).(bool); ok && forceHC {
		forceConsistency = true
		consistency = openfga.ConsistencyPreference_HIGHER_CONSISTENCY
	}
	if !forceConsistency && c.redisClient != nil {
		pinKey := fmt.Sprintf("db_pin_user:%s:openfga", userUID)
		if !errors.Is(c.redisClient.Get(ctx, pinKey).Err(), redis.Nil) {
			forceConsistency = true
			consistency = openfga.ConsistencyPreference_HIGHER_CONSISTENCY
		}
	}

	cacheKey := permissionCacheKey(userType, userUID, objectType, objectUID.String(), role)
	if c.cacheEnabled && c.redisClient != nil && !forceConsistency {
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

	log.Debug("CheckPermission",
		zap.String("userType", userType),
		zap.String("userUID", userUID),
		zap.String("objectType", objectType),
		zap.String("objectUID", objectUID.String()),
		zap.String("role", role),
		zap.String("modelID", modelID),
		zap.String("storeID", c.storeID),
		zap.Bool("forceConsistency", forceConsistency),
		zap.String("consistency", consistency.String()),
	)

	// Create a CheckRequest to verify the user's permission
	checkReq := &openfga.CheckRequest{
		StoreId:              c.storeID,
		AuthorizationModelId: modelID,
		TupleKey: &openfga.CheckRequestTupleKey{
			User:     fmt.Sprintf("%s:%s", userType, userUID),
			Relation: role,
			Object:   fmt.Sprintf("%s:%s", objectType, objectUID.String()),
		},
		Consistency: consistency,
	}
	data, err := c.getClient(ctx, ReadMode).Check(ctx, checkReq)
	if err != nil {
		log.Error("CheckPermission failed", zap.Error(err))
		return false, err
	}

	if c.cacheEnabled && c.redisClient != nil && !forceConsistency {
		cacheValue := "0"
		if data.Allowed {
			cacheValue = "1"
		}
		if err := c.redisClient.Set(ctx, cacheKey, cacheValue, c.cacheTTL).Err(); err != nil {
			log.Warn("CheckPermission failed to cache result", zap.Error(err))
		}
	}

	log.Debug("CheckPermission result", zap.Bool("allowed", data.Allowed), zap.Bool("forceConsistency", forceConsistency))

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

// IsUserPinned checks if the user is currently pinned to the primary database for read-after-write consistency.
// This is used to determine whether to use HIGHER_CONSISTENCY mode in OpenFGA queries
// and whether to bypass in-memory caches.
func (c *ACLClient) IsUserPinned(ctx context.Context) bool {
	if c.redisClient == nil {
		return false
	}
	userUID := resource.GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey)
	if userUID == "" {
		return false
	}
	pinKey := fmt.Sprintf("db_pin_user:%s:openfga", userUID)
	return !errors.Is(c.redisClient.Get(ctx, pinKey).Err(), redis.Nil)
}

// ListPermissions lists all objects of a type that the current user
// has the given role on. The caller's identity is resolved from the
// request context via resolveACLSubject (authenticated user UID takes
// precedence, with visitor/capability UIDs as the fallback).
//
// Use StreamedListObjects under the hood to avoid the 1000-result
// limit of the regular ListObjects API, which matters for roles like
// "reader" where a single user can be granted access to thousands of
// objects.
//
// Callers that want to enumerate globally public objects (tuples keyed
// to `user:*`) must use ListPublicPermissions instead; there is no
// flag here because a wildcard listing has no caller identity and
// conflating "list as me" with "list as anyone" has historically
// produced subtle bugs (e.g. silently returning the public set when
// the caller's identity failed to resolve).
func (c *ACLClient) ListPermissions(ctx context.Context, objectType string, role string) ([]uuid.UUID, error) {
	userType, userUIDStr, err := resolveACLSubject(ctx)
	if err != nil {
		return nil, err
	}
	return c.listObjectsForSubject(ctx, objectType, role, userType, userUIDStr)
}

// ListPublicPermissions lists all objects of a type that are readable
// by everyone — i.e. tuples whose subject is the FGA wildcard
// `user:*`. Separate from ListPermissions because the two have
// incompatible authorisation semantics: a wildcard query has no
// caller identity, must not return empty on an unauthenticated
// request, and is never subject to the per-user read-after-write
// pinning that ListPermissions honours.
func (c *ACLClient) ListPublicPermissions(ctx context.Context, objectType string, role string) ([]uuid.UUID, error) {
	return c.listObjectsForSubject(ctx, objectType, role, "user", "*")
}

// listObjectsForSubject issues a StreamedListObjects call for the
// given FGA subject. Factored out so ListPermissions (caller-scoped)
// and ListPublicPermissions (wildcard) can share the streaming,
// consistency, and error-translation logic without each growing its
// own copy that then drifts.
//
// Results are cached in Redis (when caching is enabled) to avoid
// repeating the expensive StreamedListObjects call on every request.
// The cache is bypassed when HIGHER_CONSISTENCY is required (context
// flag or user-pin).
func (c *ACLClient) listObjectsForSubject(ctx context.Context, objectType, role, userType, userUIDStr string) ([]uuid.UUID, error) {
	log, _ := logx.GetZapLogger(ctx)

	modelID, err := c.getAuthorizationModelID(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting authorization model: %w", err)
	}

	// Use HIGHER_CONSISTENCY when the caller forced it via context (object-level
	// visibility change) or when the user is pinned (per-user read-after-write).
	forceConsistency := false
	consistency := openfga.ConsistencyPreference_MINIMIZE_LATENCY
	if forceHC, ok := ctx.Value(ContextKeyForceHigherConsistency).(bool); ok && forceHC {
		forceConsistency = true
		consistency = openfga.ConsistencyPreference_HIGHER_CONSISTENCY
	} else if c.IsUserPinned(ctx) {
		forceConsistency = true
		consistency = openfga.ConsistencyPreference_HIGHER_CONSISTENCY
	}

	// --- Cache read ---
	cacheKey := listPermissionsCacheKey(userType, userUIDStr, objectType, role)
	if c.listPermissionsCacheOn && c.redisClient != nil && !forceConsistency {
		cached, redisErr := c.redisClient.Get(ctx, cacheKey).Result()
		if redisErr == nil {
			var uidStrs []string
			if jsonErr := json.Unmarshal([]byte(cached), &uidStrs); jsonErr == nil {
				uids := make([]uuid.UUID, 0, len(uidStrs))
				for _, s := range uidStrs {
					uids = append(uids, uuid.FromStringOrNil(s))
				}
				log.Debug("ListPermissions cache hit",
					zap.String("cacheKey", cacheKey),
					zap.Int("count", len(uids)),
				)
				return uids, nil
			} else {
				log.Warn("ListPermissions cache unmarshal error, falling through to FGA", zap.Error(jsonErr))
			}
		} else if !errors.Is(redisErr, redis.Nil) {
			log.Warn("ListPermissions cache read error", zap.Error(redisErr))
		}
	}

	// --- FGA call ---
	startedAt := time.Now()
	stream, err := c.getClient(ctx, ReadMode).StreamedListObjects(ctx, &openfga.StreamedListObjectsRequest{
		StoreId:              c.storeID,
		AuthorizationModelId: modelID,
		User:                 fmt.Sprintf("%s:%s", userType, userUIDStr),
		Relation:             role,
		Type:                 objectType,
		Consistency:          consistency,
	})
	if err != nil {
		if statusErr, ok := status.FromError(err); ok {
			if statusErr.Code() == codes.Code(openfga.ErrorCode_type_not_found) {
				return []uuid.UUID{}, nil
			}
		}
		return nil, fmt.Errorf("starting streamed list objects: %w", err)
	}

	objectUIDs := []uuid.UUID{}
	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("receiving from stream: %w", err)
		}
		objectUIDs = append(objectUIDs, uuid.FromStringOrNil(strings.Split(resp.GetObject(), ":")[1]))
	}
	elapsed := time.Since(startedAt)

	// --- Truncation guard ---
	//
	// OpenFGA's StreamedListObjects silently caps responses at the
	// smaller of `listObjectsDeadline` (3s default) and
	// `listObjectsMaxResults` (1000 default). When either ceiling is
	// hit the stream ends with EOF carrying no error or marker, so a
	// truncated reply is indistinguishable from a complete one at the
	// gRPC layer. Caching such a reply propagates the missing tuples
	// for the cache TTL and was the root cause of the 2026-04 "files
	// disappeared from the workspace" incident.
	//
	// The heuristic flags the result as suspect when EITHER:
	//   - elapsed time falls within Slack of the configured deadline
	//     (the deadline branch fires before the max-results branch on
	//     graphs with many transitively reachable objects), OR
	//   - the result count is at or above the configured max
	//
	// On a hit we refuse the cache write so the next read goes back
	// to FGA, and we emit a warn log carrying the stable token
	// `acl.list_objects_truncated` so on-call can grep / alert in
	// Cloud Logging without a separate metrics pipeline. The partial
	// result is still returned to the caller — the alternative would
	// break read paths that have legitimately small result sets but
	// ran slowly under FGA load.
	truncated := isLikelyTruncated(elapsed, len(objectUIDs), c.listObjectsCfg)
	if truncated {
		log.Warn("acl.list_objects_truncated: StreamedListObjects result is likely truncated; refusing to cache",
			zap.String("objectType", objectType),
			zap.String("role", role),
			zap.String("subject", fmt.Sprintf("%s:%s", userType, userUIDStr)),
			zap.Duration("elapsed", elapsed),
			zap.Duration("deadline", c.listObjectsCfg.Deadline),
			zap.Int("returned", len(objectUIDs)),
			zap.Int("maxResults", c.listObjectsCfg.MaxResults),
		)
	}

	// --- Cache write ---
	if c.listPermissionsCacheOn && c.redisClient != nil && !forceConsistency && !truncated {
		uidStrs := make([]string, len(objectUIDs))
		for i, u := range objectUIDs {
			uidStrs[i] = u.String()
		}
		data, jsonErr := json.Marshal(uidStrs)
		if jsonErr == nil {
			if err := c.redisClient.Set(ctx, cacheKey, data, c.cacheTTL).Err(); err != nil {
				log.Warn("ListPermissions failed to cache result", zap.Error(err))
			}
		}
	}

	return objectUIDs, nil
}

// ReadTuples enumerates direct-grant tuples that match the given
// filter. Backed by OpenFGA's Read API rather than ListObjects /
// StreamedListObjects because:
//
//   - Read is indexed and not subject to listObjectsDeadline /
//     listObjectsMaxResults — there is no truncation guard to write.
//   - Read returns the concrete tuple (object, relation, user)
//     instead of just the resolved object set, which is what every
//     caller wanting "who has direct access to X" actually needs.
//   - Read paginates via continuation_token, which this method walks
//     transparently so the caller never has to deal with paging.
//
// Callers that need transitively reachable objects (via FGA rewrites
// such as `viewer ... or X from permission_parent`) must keep using
// ListPermissions; Read only sees the raw tuples.
func (c *ACLClient) ReadTuples(ctx context.Context, filter ReadTupleFilter) ([]ReadTuple, error) {
	pageSize := filter.PageSize
	if pageSize <= 0 {
		pageSize = DefaultReadPageSize
	}

	tupleKey := &openfga.ReadRequestTupleKey{
		Object:   filter.Object,
		Relation: filter.Relation,
		User:     filter.User,
	}

	var (
		out               []ReadTuple
		continuationToken string
	)
	for {
		req := &openfga.ReadRequest{
			StoreId:           c.storeID,
			TupleKey:          tupleKey,
			PageSize:          wrapperspb.Int32(pageSize),
			ContinuationToken: continuationToken,
		}
		resp, err := c.getClient(ctx, ReadMode).Read(ctx, req)
		if err != nil {
			if statusErr, ok := status.FromError(err); ok {
				if statusErr.Code() == codes.Code(openfga.ErrorCode_type_not_found) {
					return out, nil
				}
			}
			return nil, fmt.Errorf("reading tuples: %w", err)
		}

		for _, t := range resp.GetTuples() {
			k := t.GetKey()
			if k == nil {
				continue
			}
			out = append(out, ReadTuple{
				Object:   k.GetObject(),
				Relation: k.GetRelation(),
				User:     k.GetUser(),
			})
		}

		continuationToken = resp.GetContinuationToken()
		if continuationToken == "" {
			break
		}
	}
	return out, nil
}

// isLikelyTruncated implements the heuristic documented inline at the
// caller. Pulled out so unit tests can exercise the threshold matrix
// without spinning a full mock stream.
func isLikelyTruncated(elapsed time.Duration, returned int, cfg ListObjectsConfig) bool {
	cfg = cfg.resolved()
	if cfg.Deadline > 0 && elapsed >= cfg.Deadline-cfg.Slack {
		return true
	}
	if cfg.MaxResults > 0 && returned >= cfg.MaxResults {
		return true
	}
	return false
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

	// Invalidate ListPermissions cache for the affected user.
	// For wildcard subjects (user:* / visitor:*), invalidate by object type
	// since every caller's list result may change.
	if user == "user:*" || user == "visitor:*" {
		c.invalidateListPermissionsCacheForObjectType(ctx, objectType)
	} else {
		c.invalidateListPermissionsCacheForUser(ctx, user)
	}

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

	// Invalidate ListPermissions cache for the affected user.
	if user == "user:*" || user == "visitor:*" {
		c.invalidateListPermissionsCacheForObjectType(ctx, objectType)
	} else {
		c.invalidateListPermissionsCacheForUser(ctx, user)
	}

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

// CheckPermissionWithShareLink evaluates a permission using BOTH the
// caller's identity AND (optionally) a share-link capability token,
// returning true if either subject grants the requested relation.
//
// Why two subjects
// ----------------
// The FGA schema models shareable resources as a single authoritative
// relation graph: e.g. `file.viewer` admits both direct identity
// grants (`user`, `user:*`, `visitor:*`) and share-link grants
// (`share_link`), and composite resources (collection, project,
// chat, ...) delegate inheritance (e.g. `file.viewer = ... or viewer
// from parent_collection`). A single CheckPermission call targets
// exactly one FGA subject, so a request that authenticates with BOTH
// a visitor cookie AND a share-link cap-token can only exercise ONE
// side of the graph per call. Making callers do two checks by hand
// has caused real bugs — most visibly artifact-backend-ee's linked
// -file download path, which checked only the visitor subject and
// silently 404'd every valid share-link download even though the
// FGA `file.viewer → viewer from parent_collection → share_link:<T>`
// chain resolved true.
//
// Semantics
// ---------
//  1. Always run the identity check first (same as CheckPermission).
//     If it grants, return (true, nil) immediately — no extra FGA
//     round-trip.
//  2. If shareToken is empty, return the identity result as-is
//     (including its error). This makes the helper a drop-in
//     replacement for CheckPermission on code paths that may or may
//     not have a cap-token: callers don't have to branch on
//     "do I have a token?" before choosing which ACL method to call.
//  3. Otherwise, run CheckShareLinkPermission. If the share-link
//     check errors, return that error (it represents an actual FGA
//     failure, not a "no grant" — which is already a clean
//     `(false, nil)`). If it grants, return (true, nil). Otherwise
//     return `(false, nil)` — a clean "denied" regardless of whether
//     the identity check earlier produced an ErrUnauthenticated,
//     because for a cap-token-bearing request the absence of a
//     visitor UID is not a 401 condition; the cap-token itself IS
//     the authentication.
//
// Scope isolation guarantee
// -------------------------
// Per-resource cap-token isolation is preserved for free: a
// share_link tuple is written only for the specific resource the
// token was minted for (e.g. `share_link:<TA> viewer collection:A`),
// so `CheckPermission(file:F, share_link:<TA>)` resolves true only
// if `file:F` inherits from `collection:A` (via its own
// `parent_collection` tuples). A cap-token for A cannot unlock
// files parented to B.
//
// Caller responsibility
// ---------------------
// Callers must extract `shareToken` from their domain-specific
// header (x-ee's HeaderCapabilityTokenUIDKey) and pass it here. The
// x library deliberately avoids the EE-header dependency so the
// two packages do not cycle.
func (c *ACLClient) CheckPermissionWithShareLink(
	ctx context.Context, objectType string, objectUID uuid.UUID, relation string, shareToken string,
) (bool, error) {
	granted, identityErr := c.CheckPermission(ctx, objectType, objectUID, relation)
	if identityErr == nil && granted {
		return true, nil
	}
	if shareToken == "" {
		return false, identityErr
	}
	granted2, shareErr := c.CheckShareLinkPermission(ctx, shareToken, objectType, objectUID, relation)
	if shareErr != nil {
		return false, shareErr
	}
	return granted2, nil
}

// CheckRequesterPermission validates namespace delegation: when a user
// (Instill-User-Uid) operates within an organization workspace, the frontend
// sets Instill-Requester-Uid to the org UID. This check ensures the
// authenticated user is a member of that organization.
//
// Visitors skip this check: they have no namespace membership to validate.
// Their per-resource access is gated by CheckPermission using the
// visitor:{uid} FGA identity.
func (c *ACLClient) CheckRequesterPermission(ctx context.Context) error {
	authType := resource.GetRequestSingleHeader(ctx, constant.HeaderAuthTypeKey)
	if authType == "visitor" {
		return nil
	}
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
