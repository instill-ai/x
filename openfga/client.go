package openfga

import (
	"context"
	"fmt"
	"strings"

	"github.com/gofrs/uuid"
	"go.uber.org/zap"

	openfga "github.com/openfga/go-sdk"
	openfgaclient "github.com/openfga/go-sdk/client"

	"github.com/instill-ai/x/constant"
	"github.com/instill-ai/x/errors"
	"github.com/instill-ai/x/resource"
)

type ObjectType string
type OwnerType string
type UserType string

const (
	OwnerTypeUser         OwnerType = "user"
	OwnerTypeOrganization OwnerType = "organization"
)

const (
	UserTypeUser    UserType = "user"
	UserTypeVisitor UserType = "visitor"
	UserTypeCode    UserType = "code"
)

// Client defines the interface for OpenFGA operations using the SDK.
type Client interface {
	// Store and Model configuration
	SetStoreID(storeID string) error
	SetAuthorizationModelID(modelID string) error
	GetStoreID() string
	GetAuthorizationModelID() string

	// Common ACL operations
	CheckPermission(ctx context.Context, objectType ObjectType, objectUID uuid.UUID, role string) (bool, error)
	CheckPermissionByUser(ctx context.Context, objectType ObjectType, objectUID uuid.UUID, userType UserType, userUID string, role string) (bool, error)
	SetOwner(ctx context.Context, objectType ObjectType, objectUID uuid.UUID, ownerType OwnerType, ownerUID uuid.UUID) error
	Purge(ctx context.Context, objectType ObjectType, objectUID uuid.UUID) error
	ListPermissions(ctx context.Context, objectType ObjectType, role string, isPublic bool) ([]uuid.UUID, error)

	// Direct SDK client access (for migration compatibility)
	SDKClient() *openfgaclient.OpenFgaClient

	// Utility methods
	Close() error
}

// ClientParams contains parameters for creating a new OpenFGA client.
type ClientParams struct {
	Config Config
	Logger *zap.Logger
}

// client implements the Client interface
type client struct {
	sdkClient            *openfgaclient.OpenFgaClient
	logger               *zap.Logger
	storeID              string
	authorizationModelID string
}

// NewClient creates a new OpenFGA SDK client.
func NewClient(params ClientParams) (Client, error) {
	cfg := &openfgaclient.ClientConfiguration{
		ApiScheme: "http",
		ApiHost:   fmt.Sprintf("%s:%d", params.Config.Host, params.Config.Port),
	}

	sdkClient, err := openfgaclient.NewSdkClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create OpenFGA SDK client: %w", err)
	}

	return &client{
		sdkClient: sdkClient,
		logger:    params.Logger,
	}, nil
}

// SetStoreID sets the store ID for the client
func (c *client) SetStoreID(storeID string) error {
	if storeID == "" {
		return ErrInvalidRequest
	}
	c.storeID = storeID
	return c.sdkClient.SetStoreId(storeID)
}

// SetAuthorizationModelID sets the authorization model ID for the client
func (c *client) SetAuthorizationModelID(modelID string) error {
	if modelID == "" {
		return ErrInvalidRequest
	}
	c.authorizationModelID = modelID
	return c.sdkClient.SetAuthorizationModelId(modelID)
}

// GetStoreID returns the current store ID
func (c *client) GetStoreID() string {
	return c.storeID
}

// GetAuthorizationModelID returns the current authorization model ID
func (c *client) GetAuthorizationModelID() string {
	return c.authorizationModelID
}

// Helper methods for common ACL operations

// getUserFromContext extracts user type and UID from request context
func (c *client) getUserFromContext(ctx context.Context) (userType UserType, userUID string, err error) {
	userType = UserType(resource.GetRequestSingleHeader(ctx, constant.HeaderAuthTypeKey))

	if userType == UserTypeUser {
		userUID = resource.GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey)
	} else {
		userUID = resource.GetRequestSingleHeader(ctx, constant.HeaderVisitorUIDKey)
	}

	if userUID == "" {
		return "", "", fmt.Errorf("%w: userUID is empty", errors.ErrUnauthenticated)
	}

	return userType, userUID, nil
}

// formatObject creates object string in format "type:uuid"
func (c *client) formatObject(objectType ObjectType, objectUID uuid.UUID) string {
	return fmt.Sprintf("%s:%s", objectType, objectUID.String())
}

// formatOwner creates owner string in format "type:uuid"
func (c *client) formatOwner(ownerType OwnerType, ownerUID uuid.UUID) string {
	return fmt.Sprintf("%s:%s", ownerType, ownerUID.String())
}

func (c *client) formatUser(userType UserType, userUID string) string {
	return fmt.Sprintf("%s:%s", userType, userUID)
}

// Common ACL operation implementations

// CheckPermission checks if the current user (from context) has a specific permission
func (c *client) CheckPermission(ctx context.Context, objectType ObjectType, objectUID uuid.UUID, role string) (bool, error) {
	if c.sdkClient == nil {
		return false, ErrClientNotSet
	}

	// Check if service is "instill" - bypass ACL for internal services
	serviceType := resource.GetRequestSingleHeader(ctx, constant.HeaderServiceKey)
	if serviceType == "instill" {
		return true, nil
	}

	// Get user type and UID from context
	authType := resource.GetRequestSingleHeader(ctx, constant.HeaderAuthTypeKey)

	userType := UserTypeUser
	var userUID string
	switch authType {
	case "user":
		userType = UserTypeUser
		userUID = resource.GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey)
	case "visitor":
		userType = UserTypeVisitor
		userUID = resource.GetRequestSingleHeader(ctx, constant.HeaderVisitorUIDKey)
	}

	if userUID == "" {
		return false, fmt.Errorf("%w: userUID is empty", errors.ErrUnauthenticated)
	}

	// Check permission with the user
	checkBody := openfgaclient.ClientCheckRequest{
		User:     c.formatUser(userType, userUID),
		Relation: role,
		Object:   c.formatObject(objectType, objectUID),
	}

	data, err := c.sdkClient.Check(ctx).Body(checkBody).Execute()
	if err != nil {
		c.logger.Error("Failed to check permission", zap.Error(err))
		return false, err
	}

	if *data.Allowed {
		return true, nil
	}

	// If not allowed, try with code as fallback
	return c.checkLinkPermission(ctx, objectType, objectUID, role)
}

func (c *client) CheckPermissionByUser(ctx context.Context, objectType ObjectType, objectUID uuid.UUID, userType UserType, userUID string, role string) (bool, error) {
	// Check permission with the user
	checkBody := openfgaclient.ClientCheckRequest{
		User:     c.formatUser(userType, userUID),
		Relation: role,
		Object:   c.formatObject(objectType, objectUID),
	}

	data, err := c.sdkClient.Check(ctx).Body(checkBody).Execute()
	if err != nil {
		c.logger.Error("Failed to check permission", zap.Error(err))
		return false, err
	}

	return *data.Allowed, nil
}

// SetOwner sets the owner of an object with proper validation
func (c *client) SetOwner(ctx context.Context, objectType ObjectType, objectUID uuid.UUID, ownerType OwnerType, ownerUID uuid.UUID) error {
	if c.sdkClient == nil {
		return ErrClientNotSet
	}

	// Check if owner already exists
	readOptions := openfgaclient.ClientReadOptions{
		PageSize: openfga.PtrInt32(1),
	}
	readBody := openfgaclient.ClientReadRequest{
		User:     openfga.PtrString(c.formatOwner(ownerType, ownerUID)),
		Relation: openfga.PtrString("owner"),
		Object:   openfga.PtrString(c.formatObject(objectType, objectUID)),
	}

	data, err := c.sdkClient.Read(ctx).Body(readBody).Options(readOptions).Execute()
	if err != nil {
		c.logger.Error("Failed to check existing owner", zap.Error(err))
		return err
	}
	if len(data.Tuples) > 0 {
		return nil // Owner already exists
	}

	// Write the new owner
	writeBody := openfgaclient.ClientWriteRequest{
		Writes: []openfgaclient.ClientTupleKey{
			{
				User:     c.formatOwner(ownerType, ownerUID),
				Relation: "owner",
				Object:   c.formatObject(objectType, objectUID),
			},
		},
	}
	_, err = c.sdkClient.Write(ctx).Body(writeBody).Execute()
	if err != nil {
		c.logger.Error("Failed to set owner", zap.Error(err))
	}
	return err
}

// Purge removes all permissions for a specific object
func (c *client) Purge(ctx context.Context, objectType ObjectType, objectUID uuid.UUID) error {
	if c.sdkClient == nil {
		return ErrClientNotSet
	}

	options := openfgaclient.ClientReadOptions{
		PageSize: openfga.PtrInt32(100),
	}
	readBody := openfgaclient.ClientReadRequest{
		Object: openfga.PtrString(c.formatObject(objectType, objectUID)),
	}

	data, err := c.sdkClient.Read(ctx).Body(readBody).Options(options).Execute()
	if err != nil {
		c.logger.Error("Failed to read tuples for purge", zap.Error(err))
		return err
	}

	// Delete each tuple
	for _, tuple := range data.Tuples {
		deleteBody := openfgaclient.ClientWriteRequest{
			Deletes: []openfgaclient.ClientTupleKeyWithoutCondition{
				{
					User:     tuple.Key.User,
					Relation: tuple.Key.Relation,
					Object:   tuple.Key.Object,
				},
			},
		}
		_, err = c.sdkClient.Write(ctx).Body(deleteBody).Execute()
		if err != nil {
			c.logger.Error("Failed to delete tuple during purge", zap.Error(err))
			return err
		}
	}

	return nil
}

// ListPermissions lists objects that a user has a specific permission on
func (c *client) ListPermissions(ctx context.Context, objectType ObjectType, role string, isPublic bool) ([]uuid.UUID, error) {
	if c.sdkClient == nil {
		return []uuid.UUID{}, ErrClientNotSet
	}

	userType, userUIDStr, err := c.getUserFromContext(ctx)
	if err != nil && !isPublic {
		return []uuid.UUID{}, err
	}

	if isPublic {
		userUIDStr = "*"
	}

	listBody := openfgaclient.ClientListObjectsRequest{
		User:     fmt.Sprintf("%s:%s", userType, userUIDStr),
		Relation: role,
		Type:     string(objectType),
	}

	listObjectsResult, err := c.sdkClient.ListObjects(ctx).Body(listBody).Execute()
	if err != nil {
		c.logger.Error("Failed to list user permissions", zap.Error(err))
		return []uuid.UUID{}, nil // Return empty list on error
	}

	objectUIDs := []uuid.UUID{}
	for _, object := range listObjectsResult.Objects {
		parts := strings.Split(object, ":")
		if len(parts) == 2 {
			objectUIDs = append(objectUIDs, uuid.FromStringOrNil(parts[1]))
		}
	}

	return objectUIDs, nil
}

// checkLinkPermission checks permission using code from headers (for shareable links)
func (c *client) checkLinkPermission(ctx context.Context, objectType ObjectType, objectUID uuid.UUID, role string) (bool, error) {
	code := resource.GetRequestSingleHeader(ctx, constant.HeaderInstillCodeKey)
	if code == "" {
		return false, nil
	}

	checkBody := openfgaclient.ClientCheckRequest{
		User:     c.formatUser(UserTypeCode, code),
		Relation: role,
		Object:   c.formatObject(objectType, objectUID),
	}

	data, err := c.sdkClient.Check(ctx).Body(checkBody).Execute()
	if err != nil {
		c.logger.Error("Failed to check link permission", zap.Error(err))
		return false, err
	}

	return *data.Allowed, nil
}

// SDKClient returns the underlying SDK client for migration compatibility
func (c *client) SDKClient() *openfgaclient.OpenFgaClient {
	return c.sdkClient
}

// Close closes the client connection
func (c *client) Close() error {
	// The SDK client doesn't have a Close method, so nothing to do here
	return nil
}
