package acl

import (
	"context"
	"errors"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofrs/uuid"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	openfga "github.com/openfga/api/proto/openfga/v1"

	"github.com/instill-ai/x/constant"
	errorsx "github.com/instill-ai/x/errors"
)

// --- Test fixtures ---

var (
	testUserUID    = "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
	testVisitorUID = "b6e1f5c3-7a2d-4e89-9f01-abc123def456"
	testOrgUID     = "c7d8e9f0-1a2b-3c4d-5e6f-fedcba987654"
	testObjectUID  = uuid.Must(uuid.FromString("d0e1f2a3-b4c5-6789-0abc-def123456789"))
	testModelID    = "model-01HXYZ"
	testStoreID    = "store-01HXYZ"
)

// --- Mock OpenFGA client ---

type mockFGA struct {
	checkFn               func(ctx context.Context, req *openfga.CheckRequest) (*openfga.CheckResponse, error)
	readFn                func(ctx context.Context, req *openfga.ReadRequest) (*openfga.ReadResponse, error)
	writeFn               func(ctx context.Context, req *openfga.WriteRequest) (*openfga.WriteResponse, error)
	streamedListObjectsFn func(ctx context.Context, req *openfga.StreamedListObjectsRequest) (openfga.OpenFGAService_StreamedListObjectsClient, error)
}

func (m *mockFGA) Check(ctx context.Context, in *openfga.CheckRequest, _ ...grpc.CallOption) (*openfga.CheckResponse, error) {
	if m.checkFn != nil {
		return m.checkFn(ctx, in)
	}
	return &openfga.CheckResponse{Allowed: false}, nil
}

func (m *mockFGA) Read(ctx context.Context, in *openfga.ReadRequest, _ ...grpc.CallOption) (*openfga.ReadResponse, error) {
	if m.readFn != nil {
		return m.readFn(ctx, in)
	}
	return &openfga.ReadResponse{}, nil
}

func (m *mockFGA) Write(ctx context.Context, in *openfga.WriteRequest, _ ...grpc.CallOption) (*openfga.WriteResponse, error) {
	if m.writeFn != nil {
		return m.writeFn(ctx, in)
	}
	return &openfga.WriteResponse{}, nil
}

func (m *mockFGA) StreamedListObjects(ctx context.Context, in *openfga.StreamedListObjectsRequest, _ ...grpc.CallOption) (openfga.OpenFGAService_StreamedListObjectsClient, error) {
	if m.streamedListObjectsFn != nil {
		return m.streamedListObjectsFn(ctx, in)
	}
	return &mockStream{items: nil}, nil
}

func (m *mockFGA) Expand(context.Context, *openfga.ExpandRequest, ...grpc.CallOption) (*openfga.ExpandResponse, error) {
	panic("not used")
}
func (m *mockFGA) ReadAuthorizationModels(context.Context, *openfga.ReadAuthorizationModelsRequest, ...grpc.CallOption) (*openfga.ReadAuthorizationModelsResponse, error) {
	panic("not used")
}
func (m *mockFGA) ReadAuthorizationModel(context.Context, *openfga.ReadAuthorizationModelRequest, ...grpc.CallOption) (*openfga.ReadAuthorizationModelResponse, error) {
	panic("not used")
}
func (m *mockFGA) WriteAuthorizationModel(context.Context, *openfga.WriteAuthorizationModelRequest, ...grpc.CallOption) (*openfga.WriteAuthorizationModelResponse, error) {
	panic("not used")
}
func (m *mockFGA) WriteAssertions(context.Context, *openfga.WriteAssertionsRequest, ...grpc.CallOption) (*openfga.WriteAssertionsResponse, error) {
	panic("not used")
}
func (m *mockFGA) ReadAssertions(context.Context, *openfga.ReadAssertionsRequest, ...grpc.CallOption) (*openfga.ReadAssertionsResponse, error) {
	panic("not used")
}
func (m *mockFGA) ReadChanges(context.Context, *openfga.ReadChangesRequest, ...grpc.CallOption) (*openfga.ReadChangesResponse, error) {
	panic("not used")
}
func (m *mockFGA) CreateStore(context.Context, *openfga.CreateStoreRequest, ...grpc.CallOption) (*openfga.CreateStoreResponse, error) {
	panic("not used")
}
func (m *mockFGA) UpdateStore(context.Context, *openfga.UpdateStoreRequest, ...grpc.CallOption) (*openfga.UpdateStoreResponse, error) {
	panic("not used")
}
func (m *mockFGA) DeleteStore(context.Context, *openfga.DeleteStoreRequest, ...grpc.CallOption) (*openfga.DeleteStoreResponse, error) {
	panic("not used")
}
func (m *mockFGA) GetStore(context.Context, *openfga.GetStoreRequest, ...grpc.CallOption) (*openfga.GetStoreResponse, error) {
	panic("not used")
}
func (m *mockFGA) ListStores(context.Context, *openfga.ListStoresRequest, ...grpc.CallOption) (*openfga.ListStoresResponse, error) {
	panic("not used")
}
func (m *mockFGA) ListObjects(context.Context, *openfga.ListObjectsRequest, ...grpc.CallOption) (*openfga.ListObjectsResponse, error) {
	panic("not used")
}
func (m *mockFGA) ListUsers(context.Context, *openfga.ListUsersRequest, ...grpc.CallOption) (*openfga.ListUsersResponse, error) {
	panic("not used")
}

// --- Mock stream for ListPermissions ---

type mockStream struct {
	grpc.ClientStream
	items []*openfga.StreamedListObjectsResponse
	pos   int
}

func (s *mockStream) Recv() (*openfga.StreamedListObjectsResponse, error) {
	if s.pos >= len(s.items) {
		return nil, io.EOF
	}
	resp := s.items[s.pos]
	s.pos++
	return resp, nil
}

type mockErrorStream struct {
	grpc.ClientStream
	items    []*openfga.StreamedListObjectsResponse
	pos      int
	errAfter int
}

func (s *mockErrorStream) Recv() (*openfga.StreamedListObjectsResponse, error) {
	if s.pos >= s.errAfter {
		return nil, fmt.Errorf("stream interrupted")
	}
	if s.pos >= len(s.items) {
		return nil, io.EOF
	}
	resp := s.items[s.pos]
	s.pos++
	return resp, nil
}

// --- Helpers ---

func ctxWithHeaders(headers map[string]string) context.Context {
	md := metadata.New(headers)
	return metadata.NewIncomingContext(context.Background(), md)
}

func newTestClient(fga *mockFGA) *ACLClient {
	return &ACLClient{
		writeClient: fga,
		readClient:  fga,
		storeID:     testStoreID,
		modelID:     testModelID,
	}
}

func newTestClientWithCache(fga *mockFGA) (*ACLClient, *miniredis.Miniredis) {
	mr := miniredis.RunT(&testing.T{})
	rc := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	return &ACLClient{
		writeClient:            fga,
		readClient:             fga,
		redisClient:            rc,
		storeID:                testStoreID,
		modelID:                testModelID,
		cacheEnabled:           true, // CheckPermission cache
		listPermissionsCacheOn: true, // ListPermissions / ListPublicPermissions cache
		cacheTTL:               60 * time.Second,
		listObjectsCfg:         DefaultListObjectsConfig(),
	}, mr
}

func userCtx(userUID string) context.Context {
	return ctxWithHeaders(map[string]string{
		constant.HeaderAuthTypeKey: "user",
		constant.HeaderUserUIDKey:  userUID,
	})
}

func visitorCtx(visitorUID string) context.Context {
	return ctxWithHeaders(map[string]string{
		constant.HeaderAuthTypeKey:   "visitor",
		constant.HeaderVisitorUIDKey: visitorUID,
	})
}

// capabilityVisitorCtx models an anonymous visitor reading a
// `/r/{token}` share link. The gateway stamps
// Instill-Auth-Type=capability together with Instill-Visitor-Uid (no
// user UID) — cap-token adds resource scope on top of the visitor
// identity but does not itself authenticate anyone.
func capabilityVisitorCtx(visitorUID string) context.Context {
	return ctxWithHeaders(map[string]string{
		constant.HeaderAuthTypeKey:   "capability",
		constant.HeaderVisitorUIDKey: visitorUID,
	})
}

// dualAuthCtx models an authenticated user who is also reading a
// `/r/{token}` share link. The gateway stamps
// Instill-Auth-Type=capability AND Instill-User-Uid (from the JWT)
// AND does NOT set a visitor UID. This is the regression vector for
// the dual-auth bug: the old selector branched on auth-type first and
// silently dropped the user UID.
func dualAuthCtx(userUID string) context.Context {
	return ctxWithHeaders(map[string]string{
		constant.HeaderAuthTypeKey: "capability",
		constant.HeaderUserUIDKey:  userUID,
	})
}

func orgDelegationCtx(userUID, orgUID string) context.Context {
	return ctxWithHeaders(map[string]string{
		constant.HeaderAuthTypeKey:     "user",
		constant.HeaderUserUIDKey:      userUID,
		constant.HeaderRequesterUIDKey: orgUID,
	})
}

// ============================================================
// CheckRequesterPermission — namespace delegation
// ============================================================

func TestCheckRequesterPermission_VisitorSkipsCheck(t *testing.T) {
	c := newTestClient(&mockFGA{})
	ctx := visitorCtx(testVisitorUID)

	if err := c.CheckRequesterPermission(ctx); err != nil {
		t.Errorf("visitors should bypass requester check, got: %v", err)
	}
}

func TestCheckRequesterPermission_UnauthenticatedRejected(t *testing.T) {
	c := newTestClient(&mockFGA{})
	ctx := ctxWithHeaders(map[string]string{})

	err := c.CheckRequesterPermission(ctx)
	if err == nil {
		t.Fatal("unauthenticated request should be rejected")
	}
	if !errors.Is(err, errorsx.ErrUnauthenticated) {
		t.Errorf("expected ErrUnauthenticated, got: %v", err)
	}
}

func TestCheckRequesterPermission_UserInOwnNamespace(t *testing.T) {
	c := newTestClient(&mockFGA{})
	ctx := userCtx(testUserUID)

	if err := c.CheckRequesterPermission(ctx); err != nil {
		t.Errorf("user in own namespace (no requester) should pass, got: %v", err)
	}
}

func TestCheckRequesterPermission_UserExplicitSelfRequester(t *testing.T) {
	c := newTestClient(&mockFGA{})
	ctx := orgDelegationCtx(testUserUID, testUserUID)

	if err := c.CheckRequesterPermission(ctx); err != nil {
		t.Errorf("user with requester=self should pass, got: %v", err)
	}
}

func TestCheckRequesterPermission_OrgMemberAllowed(t *testing.T) {
	fga := &mockFGA{
		checkFn: func(_ context.Context, req *openfga.CheckRequest) (*openfga.CheckResponse, error) {
			if req.TupleKey.User != fmt.Sprintf("user:%s", testUserUID) {
				t.Errorf("FGA should check with user identity, got: %s", req.TupleKey.User)
			}
			if req.TupleKey.Relation != "member" {
				t.Errorf("FGA should check member relation, got: %s", req.TupleKey.Relation)
			}
			return &openfga.CheckResponse{Allowed: true}, nil
		},
	}
	c := newTestClient(fga)
	ctx := orgDelegationCtx(testUserUID, testOrgUID)

	if err := c.CheckRequesterPermission(ctx); err != nil {
		t.Errorf("org member should be allowed, got: %v", err)
	}
}

func TestCheckRequesterPermission_NonMemberDenied(t *testing.T) {
	fga := &mockFGA{
		checkFn: func(_ context.Context, _ *openfga.CheckRequest) (*openfga.CheckResponse, error) {
			return &openfga.CheckResponse{Allowed: false}, nil
		},
	}
	c := newTestClient(fga)
	ctx := orgDelegationCtx(testUserUID, testOrgUID)

	err := c.CheckRequesterPermission(ctx)
	if err == nil {
		t.Fatal("non-member should be denied")
	}
	if !errors.Is(err, errorsx.ErrPermissionDenied) {
		t.Errorf("expected ErrPermissionDenied, got: %v", err)
	}
}

func TestCheckRequesterPermission_FGAErrorPropagated(t *testing.T) {
	fga := &mockFGA{
		checkFn: func(_ context.Context, _ *openfga.CheckRequest) (*openfga.CheckResponse, error) {
			return nil, fmt.Errorf("openfga unavailable")
		},
	}
	c := newTestClient(fga)
	ctx := orgDelegationCtx(testUserUID, testOrgUID)

	err := c.CheckRequesterPermission(ctx)
	if err == nil {
		t.Fatal("FGA error should propagate")
	}
}

// ============================================================
// CheckPermission — per-resource access check
// ============================================================

func TestCheckPermission_UserGranted(t *testing.T) {
	fga := &mockFGA{
		checkFn: func(_ context.Context, req *openfga.CheckRequest) (*openfga.CheckResponse, error) {
			if req.TupleKey.User != fmt.Sprintf("user:%s", testUserUID) {
				t.Errorf("expected user identity, got: %s", req.TupleKey.User)
			}
			if req.TupleKey.Relation != "reader" {
				t.Errorf("expected reader relation, got: %s", req.TupleKey.Relation)
			}
			if req.TupleKey.Object != fmt.Sprintf("knowledgebase:%s", testObjectUID) {
				t.Errorf("expected knowledgebase object, got: %s", req.TupleKey.Object)
			}
			return &openfga.CheckResponse{Allowed: true}, nil
		},
	}
	c := newTestClient(fga)
	ctx := userCtx(testUserUID)

	granted, err := c.CheckPermission(ctx, "knowledgebase", testObjectUID, "reader")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !granted {
		t.Error("user with permission should be granted")
	}
}

func TestCheckPermission_UserDenied(t *testing.T) {
	fga := &mockFGA{
		checkFn: func(_ context.Context, _ *openfga.CheckRequest) (*openfga.CheckResponse, error) {
			return &openfga.CheckResponse{Allowed: false}, nil
		},
	}
	c := newTestClient(fga)
	ctx := userCtx(testUserUID)

	granted, err := c.CheckPermission(ctx, "knowledgebase", testObjectUID, "writer")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if granted {
		t.Error("user without permission should be denied")
	}
}

func TestCheckPermission_VisitorUsesVisitorIdentity(t *testing.T) {
	fga := &mockFGA{
		checkFn: func(_ context.Context, req *openfga.CheckRequest) (*openfga.CheckResponse, error) {
			expected := fmt.Sprintf("visitor:%s", testVisitorUID)
			if req.TupleKey.User != expected {
				t.Errorf("visitor should use visitor:{uid} identity, got: %s", req.TupleKey.User)
			}
			return &openfga.CheckResponse{Allowed: true}, nil
		},
	}
	c := newTestClient(fga)
	ctx := visitorCtx(testVisitorUID)

	granted, err := c.CheckPermission(ctx, "knowledgebase", testObjectUID, "reader")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !granted {
		t.Error("visitor with public access should be granted")
	}
}

func TestCheckPermission_EmptyUIDRejected(t *testing.T) {
	c := newTestClient(&mockFGA{})
	ctx := ctxWithHeaders(map[string]string{
		constant.HeaderAuthTypeKey: "user",
	})

	_, err := c.CheckPermission(ctx, "knowledgebase", testObjectUID, "reader")
	if err == nil {
		t.Fatal("empty UID should be rejected")
	}
}

func TestCheckPermission_FGAErrorPropagated(t *testing.T) {
	fga := &mockFGA{
		checkFn: func(_ context.Context, _ *openfga.CheckRequest) (*openfga.CheckResponse, error) {
			return nil, fmt.Errorf("connection refused")
		},
	}
	c := newTestClient(fga)
	ctx := userCtx(testUserUID)

	_, err := c.CheckPermission(ctx, "knowledgebase", testObjectUID, "reader")
	if err == nil {
		t.Fatal("FGA error should propagate")
	}
}

func TestCheckPermission_CachedPermissionSkipsFGA(t *testing.T) {
	var fgaCalled bool
	fga := &mockFGA{
		checkFn: func(_ context.Context, _ *openfga.CheckRequest) (*openfga.CheckResponse, error) {
			fgaCalled = true
			return &openfga.CheckResponse{Allowed: true}, nil
		},
	}
	c, mr := newTestClientWithCache(fga)
	defer mr.Close()
	ctx := userCtx(testUserUID)

	// First call should hit FGA and cache the result
	granted, err := c.CheckPermission(ctx, "knowledgebase", testObjectUID, "reader")
	if err != nil || !granted {
		t.Fatalf("first call should succeed: err=%v, granted=%v", err, granted)
	}
	if !fgaCalled {
		t.Fatal("first call should hit FGA")
	}

	// Second call should use cache without calling FGA
	fgaCalled = false
	granted, err = c.CheckPermission(ctx, "knowledgebase", testObjectUID, "reader")
	if err != nil || !granted {
		t.Fatalf("cached call should succeed: err=%v, granted=%v", err, granted)
	}
	if fgaCalled {
		t.Error("second call should use cache, not hit FGA again")
	}
}

func TestCheckPermission_CachedDenialIsReturned(t *testing.T) {
	callCount := 0
	fga := &mockFGA{
		checkFn: func(_ context.Context, _ *openfga.CheckRequest) (*openfga.CheckResponse, error) {
			callCount++
			return &openfga.CheckResponse{Allowed: false}, nil
		},
	}
	c, mr := newTestClientWithCache(fga)
	defer mr.Close()
	ctx := userCtx(testUserUID)

	// First call: FGA denies, result is cached
	granted, _ := c.CheckPermission(ctx, "knowledgebase", testObjectUID, "writer")
	if granted {
		t.Fatal("should be denied")
	}

	// Second call: should return cached denial
	granted, _ = c.CheckPermission(ctx, "knowledgebase", testObjectUID, "writer")
	if granted {
		t.Fatal("cached denial should still deny")
	}
	if callCount != 1 {
		t.Errorf("expected 1 FGA call (cached second time), got %d", callCount)
	}
}

func TestCheckPermission_ForceConsistencyBypassesCache(t *testing.T) {
	fgaCallCount := 0
	fga := &mockFGA{
		checkFn: func(_ context.Context, req *openfga.CheckRequest) (*openfga.CheckResponse, error) {
			fgaCallCount++
			return &openfga.CheckResponse{Allowed: true}, nil
		},
	}
	c, mr := newTestClientWithCache(fga)
	defer mr.Close()
	ctx := userCtx(testUserUID)

	// First call populates cache
	_, _ = c.CheckPermission(ctx, "knowledgebase", testObjectUID, "reader")

	// Force consistency should bypass cache and hit FGA again
	ctxForce := context.WithValue(ctx, ContextKeyForceHigherConsistency, true)
	_, err := c.CheckPermission(ctxForce, "knowledgebase", testObjectUID, "reader")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fgaCallCount != 2 {
		t.Errorf("force consistency should bypass cache: expected 2 FGA calls, got %d", fgaCallCount)
	}
}

func TestCheckPermission_ForceConsistencyAfterVisibilityChange(t *testing.T) {
	fga := &mockFGA{
		checkFn: func(_ context.Context, req *openfga.CheckRequest) (*openfga.CheckResponse, error) {
			if req.Consistency != openfga.ConsistencyPreference_HIGHER_CONSISTENCY {
				t.Error("after a visibility toggle, should use HIGHER_CONSISTENCY to avoid stale cache")
			}
			return &openfga.CheckResponse{Allowed: true}, nil
		},
	}
	c := newTestClient(fga)
	ctx := userCtx(testUserUID)
	ctx = context.WithValue(ctx, ContextKeyForceHigherConsistency, true)

	_, err := c.CheckPermission(ctx, "knowledgebase", testObjectUID, "reader")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ============================================================
// SetOwner — ownership management
// ============================================================

func TestSetOwner_NewOwner(t *testing.T) {
	ownerUID := uuid.Must(uuid.FromString(testUserUID))
	var writeWasCalled bool

	fga := &mockFGA{
		readFn: func(_ context.Context, _ *openfga.ReadRequest) (*openfga.ReadResponse, error) {
			return &openfga.ReadResponse{Tuples: []*openfga.Tuple{}}, nil
		},
		writeFn: func(_ context.Context, req *openfga.WriteRequest) (*openfga.WriteResponse, error) {
			writeWasCalled = true
			tuples := req.Writes.TupleKeys
			if len(tuples) != 1 {
				t.Fatalf("expected 1 tuple, got %d", len(tuples))
			}
			if tuples[0].Relation != "owner" {
				t.Errorf("expected owner relation, got: %s", tuples[0].Relation)
			}
			return &openfga.WriteResponse{}, nil
		},
	}
	c := newTestClient(fga)

	err := c.SetOwner(context.Background(), "pipeline", testObjectUID, "user", ownerUID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !writeWasCalled {
		t.Error("should write owner tuple when no owner exists")
	}
}

func TestSetOwner_AlreadyExists(t *testing.T) {
	ownerUID := uuid.Must(uuid.FromString(testUserUID))

	fga := &mockFGA{
		readFn: func(_ context.Context, _ *openfga.ReadRequest) (*openfga.ReadResponse, error) {
			return &openfga.ReadResponse{
				Tuples: []*openfga.Tuple{{
					Key: &openfga.TupleKey{
						User:     fmt.Sprintf("user:%s", testUserUID),
						Relation: "owner",
						Object:   fmt.Sprintf("pipeline:%s", testObjectUID),
					},
				}},
			}, nil
		},
		writeFn: func(_ context.Context, _ *openfga.WriteRequest) (*openfga.WriteResponse, error) {
			t.Fatal("should NOT write when owner already exists")
			return nil, nil
		},
	}
	c := newTestClient(fga)

	err := c.SetOwner(context.Background(), "pipeline", testObjectUID, "user", ownerUID)
	if err != nil {
		t.Fatalf("idempotent SetOwner should not error, got: %v", err)
	}
}

func TestSetOwner_NormalizesOwnerType(t *testing.T) {
	orgUID := uuid.Must(uuid.FromString(testOrgUID))

	fga := &mockFGA{
		readFn: func(_ context.Context, req *openfga.ReadRequest) (*openfga.ReadResponse, error) {
			if req.TupleKey.User != fmt.Sprintf("organization:%s", testOrgUID) {
				t.Errorf("should strip trailing 's': expected organization:, got: %s", req.TupleKey.User)
			}
			return &openfga.ReadResponse{Tuples: []*openfga.Tuple{}}, nil
		},
		writeFn: func(_ context.Context, req *openfga.WriteRequest) (*openfga.WriteResponse, error) {
			if req.Writes.TupleKeys[0].User != fmt.Sprintf("organization:%s", testOrgUID) {
				t.Errorf("write should use normalized type, got: %s", req.Writes.TupleKeys[0].User)
			}
			return &openfga.WriteResponse{}, nil
		},
	}
	c := newTestClient(fga)

	err := c.SetOwner(context.Background(), "pipeline", testObjectUID, "organizations", orgUID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ============================================================
// Purge — delete all permissions for an object
// ============================================================

func TestPurge_DeletesAllTuples(t *testing.T) {
	var deletedUsers []string

	fga := &mockFGA{
		readFn: func(_ context.Context, _ *openfga.ReadRequest) (*openfga.ReadResponse, error) {
			return &openfga.ReadResponse{
				Tuples: []*openfga.Tuple{
					{Key: &openfga.TupleKey{User: "user:alice", Relation: "owner", Object: "pipeline:" + testObjectUID.String()}},
					{Key: &openfga.TupleKey{User: "user:bob", Relation: "reader", Object: "pipeline:" + testObjectUID.String()}},
				},
			}, nil
		},
		writeFn: func(_ context.Context, req *openfga.WriteRequest) (*openfga.WriteResponse, error) {
			for _, tk := range req.Deletes.TupleKeys {
				deletedUsers = append(deletedUsers, tk.User)
			}
			return &openfga.WriteResponse{}, nil
		},
	}
	c := newTestClient(fga)

	err := c.Purge(context.Background(), "pipeline", testObjectUID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(deletedUsers) != 2 {
		t.Errorf("expected 2 deletions, got %d", len(deletedUsers))
	}
}

func TestPurge_NoTuplesIsNoop(t *testing.T) {
	fga := &mockFGA{
		readFn: func(_ context.Context, _ *openfga.ReadRequest) (*openfga.ReadResponse, error) {
			return &openfga.ReadResponse{Tuples: []*openfga.Tuple{}}, nil
		},
		writeFn: func(_ context.Context, _ *openfga.WriteRequest) (*openfga.WriteResponse, error) {
			t.Fatal("should not write when no tuples exist")
			return nil, nil
		},
	}
	c := newTestClient(fga)

	if err := c.Purge(context.Background(), "pipeline", testObjectUID); err != nil {
		t.Fatalf("purge with no tuples should succeed, got: %v", err)
	}
}

// ============================================================
// GetOwner — retrieve ownership
// ============================================================

func TestGetOwner_ReturnsOwner(t *testing.T) {
	fga := &mockFGA{
		readFn: func(_ context.Context, _ *openfga.ReadRequest) (*openfga.ReadResponse, error) {
			return &openfga.ReadResponse{
				Tuples: []*openfga.Tuple{{
					Key: &openfga.TupleKey{User: "organization:" + testOrgUID, Relation: "owner"},
				}},
			}, nil
		},
	}
	c := newTestClient(fga)

	ownerType, ownerUID, err := c.GetOwner(context.Background(), "pipeline", testObjectUID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ownerType != "organization" || ownerUID != testOrgUID {
		t.Errorf("expected organization/%s, got %s/%s", testOrgUID, ownerType, ownerUID)
	}
}

func TestGetOwner_NoOwnerReturnsError(t *testing.T) {
	fga := &mockFGA{
		readFn: func(_ context.Context, _ *openfga.ReadRequest) (*openfga.ReadResponse, error) {
			return &openfga.ReadResponse{Tuples: []*openfga.Tuple{}}, nil
		},
	}
	c := newTestClient(fga)

	_, _, err := c.GetOwner(context.Background(), "pipeline", testObjectUID)
	if err == nil {
		t.Fatal("expected error when no owner exists")
	}
}

// ============================================================
// SetResourcePermission / DeleteResourcePermission
// ============================================================

func TestSetResourcePermission_EnableWritesTuple(t *testing.T) {
	var writtenUser, writtenRelation string

	fga := &mockFGA{
		writeFn: func(_ context.Context, req *openfga.WriteRequest) (*openfga.WriteResponse, error) {
			if req.Writes != nil && len(req.Writes.TupleKeys) > 0 {
				writtenUser = req.Writes.TupleKeys[0].User
				writtenRelation = req.Writes.TupleKeys[0].Relation
			}
			return &openfga.WriteResponse{}, nil
		},
	}
	c := newTestClient(fga)

	err := c.SetResourcePermission(context.Background(), "file", testObjectUID, "user:"+testUserUID, "editor", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if writtenUser != "user:"+testUserUID {
		t.Errorf("expected user:%s, got: %s", testUserUID, writtenUser)
	}
	if writtenRelation != "editor" {
		t.Errorf("expected editor relation, got: %s", writtenRelation)
	}
}

func TestSetResourcePermission_DisableOnlyDeletes(t *testing.T) {
	var writesCalled int

	fga := &mockFGA{
		writeFn: func(_ context.Context, req *openfga.WriteRequest) (*openfga.WriteResponse, error) {
			writesCalled++
			if req.Writes != nil {
				t.Error("disable=false should not write new tuples")
			}
			return &openfga.WriteResponse{}, nil
		},
	}
	c := newTestClient(fga)

	err := c.SetResourcePermission(context.Background(), "file", testObjectUID, "user:"+testUserUID, "editor", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if writesCalled == 0 {
		t.Error("should call write (for delete) even when disabling")
	}
}

func TestDeleteResourcePermission_DeletesAllStandardRoles(t *testing.T) {
	var deletedRoles []string

	fga := &mockFGA{
		writeFn: func(_ context.Context, req *openfga.WriteRequest) (*openfga.WriteResponse, error) {
			if req.Deletes != nil {
				for _, tk := range req.Deletes.TupleKeys {
					deletedRoles = append(deletedRoles, tk.Relation)
				}
			}
			return &openfga.WriteResponse{}, nil
		},
	}
	c := newTestClient(fga)

	err := c.DeleteResourcePermission(context.Background(), "file", testObjectUID, "user:"+testUserUID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := map[string]bool{"admin": true, "writer": true, "executor": true, "reader": true}
	for _, role := range deletedRoles {
		delete(expected, role)
	}
	if len(expected) > 0 {
		t.Errorf("did not delete all standard roles, missing: %v", expected)
	}
}

// ============================================================
// SetPublicPermission / DeletePublicPermission
// ============================================================

func TestSetPublicPermission_GrantsCorrectTuples(t *testing.T) {
	var writtenTuples []string

	fga := &mockFGA{
		writeFn: func(_ context.Context, req *openfga.WriteRequest) (*openfga.WriteResponse, error) {
			if req.Writes != nil {
				for _, tk := range req.Writes.TupleKeys {
					writtenTuples = append(writtenTuples, tk.User+"/"+tk.Relation)
				}
			}
			return &openfga.WriteResponse{}, nil
		},
	}
	c := newTestClient(fga)

	err := c.SetPublicPermission(context.Background(), "knowledgebase", testObjectUID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := map[string]bool{
		"user:*/reader":    true,
		"visitor:*/reader": true,
		"user:*/executor":  true,
	}
	for _, tuple := range writtenTuples {
		delete(expected, tuple)
	}
	if len(expected) > 0 {
		t.Errorf("missing public permission tuples: %v", expected)
	}
}

// ============================================================
// CheckLinkPermission — share code access
// ============================================================

func TestCheckLinkPermission_ValidCodeChecked(t *testing.T) {
	shareCode := "abc-share-123"

	fga := &mockFGA{
		checkFn: func(_ context.Context, req *openfga.CheckRequest) (*openfga.CheckResponse, error) {
			expected := fmt.Sprintf("code:%s", shareCode)
			if req.TupleKey.User != expected {
				t.Errorf("expected code identity %s, got: %s", expected, req.TupleKey.User)
			}
			return &openfga.CheckResponse{Allowed: true}, nil
		},
	}
	c := newTestClient(fga)
	ctx := ctxWithHeaders(map[string]string{
		"instill-share-code": shareCode,
	})

	granted, err := c.CheckLinkPermission(ctx, "pipeline", testObjectUID, "executor", "instill-share-code")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !granted {
		t.Error("valid share code should be granted")
	}
}

func TestCheckLinkPermission_CodeExistsButDenied(t *testing.T) {
	fga := &mockFGA{
		checkFn: func(_ context.Context, _ *openfga.CheckRequest) (*openfga.CheckResponse, error) {
			return &openfga.CheckResponse{Allowed: false}, nil
		},
	}
	c := newTestClient(fga)
	ctx := ctxWithHeaders(map[string]string{
		"instill-share-code": "expired-code-456",
	})

	granted, err := c.CheckLinkPermission(ctx, "pipeline", testObjectUID, "executor", "instill-share-code")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if granted {
		t.Error("expired or revoked share code should be denied")
	}
}

func TestCheckLinkPermission_FGAErrorPropagated(t *testing.T) {
	fga := &mockFGA{
		checkFn: func(_ context.Context, _ *openfga.CheckRequest) (*openfga.CheckResponse, error) {
			return nil, fmt.Errorf("openfga timeout")
		},
	}
	c := newTestClient(fga)
	ctx := ctxWithHeaders(map[string]string{
		"instill-share-code": "some-code",
	})

	_, err := c.CheckLinkPermission(ctx, "pipeline", testObjectUID, "executor", "instill-share-code")
	if err == nil {
		t.Fatal("FGA error should propagate, not silently fail")
	}
}

func TestCheckLinkPermission_MissingCodeReturnsFalse(t *testing.T) {
	c := newTestClient(&mockFGA{})
	ctx := ctxWithHeaders(map[string]string{})

	granted, err := c.CheckLinkPermission(ctx, "pipeline", testObjectUID, "executor", "instill-share-code")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if granted {
		t.Error("missing share code should return false, not error")
	}
}

// ============================================================
// CheckShareLinkPermission — share link token
// ============================================================

func TestCheckShareLinkPermission_ValidToken(t *testing.T) {
	token := "share-token-xyz"

	fga := &mockFGA{
		checkFn: func(_ context.Context, req *openfga.CheckRequest) (*openfga.CheckResponse, error) {
			if req.TupleKey.User != fmt.Sprintf("share_link:%s", token) {
				t.Errorf("expected share_link identity, got: %s", req.TupleKey.User)
			}
			return &openfga.CheckResponse{Allowed: true}, nil
		},
	}
	c := newTestClient(fga)

	granted, err := c.CheckShareLinkPermission(context.Background(), token, "chat", testObjectUID, "reader")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !granted {
		t.Error("valid share link should be granted")
	}
}

func TestCheckShareLinkPermission_TokenDenied(t *testing.T) {
	fga := &mockFGA{
		checkFn: func(_ context.Context, _ *openfga.CheckRequest) (*openfga.CheckResponse, error) {
			return &openfga.CheckResponse{Allowed: false}, nil
		},
	}
	c := newTestClient(fga)

	granted, err := c.CheckShareLinkPermission(context.Background(), "revoked-token", "chat", testObjectUID, "reader")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if granted {
		t.Error("revoked share link token should be denied")
	}
}

func TestCheckShareLinkPermission_FGAErrorPropagated(t *testing.T) {
	fga := &mockFGA{
		checkFn: func(_ context.Context, _ *openfga.CheckRequest) (*openfga.CheckResponse, error) {
			return nil, status.Error(codes.Internal, "internal error")
		},
	}
	c := newTestClient(fga)

	_, err := c.CheckShareLinkPermission(context.Background(), "token", "chat", testObjectUID, "reader")
	if err == nil {
		t.Fatal("generic FGA error should propagate, not be swallowed")
	}
}

func TestCheckShareLinkPermission_TypeNotFoundReturnsFalse(t *testing.T) {
	fga := &mockFGA{
		checkFn: func(_ context.Context, _ *openfga.CheckRequest) (*openfga.CheckResponse, error) {
			return nil, status.Error(codes.Code(openfga.ErrorCode_type_not_found), "type not found")
		},
	}
	c := newTestClient(fga)

	granted, err := c.CheckShareLinkPermission(context.Background(), "token", "nonexistent", testObjectUID, "reader")
	if err != nil {
		t.Fatalf("type_not_found should return false, not error: %v", err)
	}
	if granted {
		t.Error("type_not_found should return false")
	}
}

// ============================================================
// ListPermissions — object enumeration
// ============================================================

func TestListPermissions_ReturnsObjectUIDs(t *testing.T) {
	uid1 := uuid.Must(uuid.NewV4())
	uid2 := uuid.Must(uuid.NewV4())

	fga := &mockFGA{
		streamedListObjectsFn: func(_ context.Context, req *openfga.StreamedListObjectsRequest) (openfga.OpenFGAService_StreamedListObjectsClient, error) {
			if req.User != fmt.Sprintf("user:%s", testUserUID) {
				t.Errorf("expected user identity, got: %s", req.User)
			}
			return &mockStream{
				items: []*openfga.StreamedListObjectsResponse{
					{Object: fmt.Sprintf("pipeline:%s", uid1)},
					{Object: fmt.Sprintf("pipeline:%s", uid2)},
				},
			}, nil
		},
	}
	c := newTestClient(fga)
	ctx := userCtx(testUserUID)

	uids, err := c.ListPermissions(ctx, "pipeline", "reader")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(uids) != 2 {
		t.Errorf("expected 2 objects, got %d", len(uids))
	}
}

// TestListPublicPermissions_UsesWildcardSubject pins the contract of
// the dedicated public-listing method: it issues the FGA query with
// the `user:*` wildcard subject, regardless of who (if anyone) the
// caller is. The previous `ListPermissions(..., isPublic bool)` API
// coupled this to a parameter flag and conflated unauthenticated
// listings with public ones; the split makes the intent explicit at
// the call site.
func TestListPublicPermissions_UsesWildcardSubject(t *testing.T) {
	fga := &mockFGA{
		streamedListObjectsFn: func(_ context.Context, req *openfga.StreamedListObjectsRequest) (openfga.OpenFGAService_StreamedListObjectsClient, error) {
			if req.User != "user:*" {
				t.Errorf("public listing should use wildcard, got: %s", req.User)
			}
			return &mockStream{items: nil}, nil
		},
	}
	c := newTestClient(fga)
	ctx := userCtx(testUserUID)

	if _, err := c.ListPublicPermissions(ctx, "pipeline", "reader"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestListPermissions_VisitorUsesVisitorIdentity(t *testing.T) {
	fga := &mockFGA{
		streamedListObjectsFn: func(_ context.Context, req *openfga.StreamedListObjectsRequest) (openfga.OpenFGAService_StreamedListObjectsClient, error) {
			expected := fmt.Sprintf("visitor:%s", testVisitorUID)
			if req.User != expected {
				t.Errorf("visitor should use visitor identity, got: %s", req.User)
			}
			return &mockStream{items: nil}, nil
		},
	}
	c := newTestClient(fga)
	ctx := visitorCtx(testVisitorUID)

	_, err := c.ListPermissions(ctx, "knowledgebase", "reader")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestListPermissions_RecvErrorMidStream(t *testing.T) {
	fga := &mockFGA{
		streamedListObjectsFn: func(_ context.Context, _ *openfga.StreamedListObjectsRequest) (openfga.OpenFGAService_StreamedListObjectsClient, error) {
			return &mockErrorStream{
				items: []*openfga.StreamedListObjectsResponse{
					{Object: fmt.Sprintf("pipeline:%s", testObjectUID)},
				},
				errAfter: 1,
			}, nil
		},
	}
	c := newTestClient(fga)
	ctx := userCtx(testUserUID)

	_, err := c.ListPermissions(ctx, "pipeline", "reader")
	if err == nil {
		t.Fatal("error during stream receive should propagate")
	}
}

func TestListPermissions_StreamErrorPropagated(t *testing.T) {
	fga := &mockFGA{
		streamedListObjectsFn: func(_ context.Context, _ *openfga.StreamedListObjectsRequest) (openfga.OpenFGAService_StreamedListObjectsClient, error) {
			return nil, fmt.Errorf("connection lost mid-stream")
		},
	}
	c := newTestClient(fga)
	ctx := userCtx(testUserUID)

	_, err := c.ListPermissions(ctx, "pipeline", "reader")
	if err == nil {
		t.Fatal("stream error should propagate so caller can retry")
	}
}

func TestListPermissions_TypeNotFoundReturnsEmpty(t *testing.T) {
	fga := &mockFGA{
		streamedListObjectsFn: func(_ context.Context, _ *openfga.StreamedListObjectsRequest) (openfga.OpenFGAService_StreamedListObjectsClient, error) {
			return nil, status.Error(codes.Code(openfga.ErrorCode_type_not_found), "type not found")
		},
	}
	c := newTestClient(fga)
	ctx := userCtx(testUserUID)

	uids, err := c.ListPermissions(ctx, "nonexistent", "reader")
	if err != nil {
		t.Fatalf("type_not_found should return empty list, not error: %v", err)
	}
	if len(uids) != 0 {
		t.Errorf("expected empty list, got %d items", len(uids))
	}
}

// ============================================================
// resolveACLSubject — dual-auth regression coverage
// ============================================================
//
// These tests pin the behavioural contract of resolveACLSubject
// through its two public callers (CheckPermission and
// ListPermissions). The shared invariant is:
//
//	Instill-User-Uid, when present, wins over Instill-Auth-Type.
//
// That invariant used to be inverted — the selector keyed on auth
// type first, so a request that legitimately carried BOTH a JWT and
// an X-Capability-Token (the "logged-in user follows a /r/{token}
// share link" flow) resolved to a capability subject with an empty
// UID and was rejected as unauthenticated at the FGA layer.

// TestCheckPermission_DualAuth_UsesUserSubject ensures that an
// authenticated caller with an additional capability token is still
// checked against FGA as `user:<uid>`. The cap-token only widens
// which resources they may reach; it must not strip their identity.
func TestCheckPermission_DualAuth_UsesUserSubject(t *testing.T) {
	fga := &mockFGA{
		checkFn: func(_ context.Context, req *openfga.CheckRequest) (*openfga.CheckResponse, error) {
			want := fmt.Sprintf("user:%s", testUserUID)
			if req.TupleKey.User != want {
				t.Errorf("dual-auth must check as %s, got %s", want, req.TupleKey.User)
			}
			return &openfga.CheckResponse{Allowed: true}, nil
		},
	}
	c := newTestClient(fga)
	ctx := dualAuthCtx(testUserUID)

	granted, err := c.CheckPermission(ctx, "pipeline", testObjectUID, "reader")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !granted {
		t.Error("dual-auth user should be granted when FGA allows")
	}
}

// TestCheckPermission_CapabilityVisitor_CollapsesToVisitorSubject
// covers the "unauthenticated visitor on a share link" flow. Both
// `Instill-Auth-Type: visitor` and `Instill-Auth-Type: capability`
// must resolve to FGA subject type `visitor`: the identity (browser
// cookie UID) is identical, and the FGA schema has no `capability`
// type. Per-resource share-link grants are evaluated separately via
// CheckShareLinkPermission against `share_link:<token>` tuples, not
// via the caller's FGA subject. Emitting `capability:<uid>` here
// would produce a subject with no possible tuple match and surface
// as a `type 'capability' not found` FGA error — exactly the
// failure mode this test guards against.
func TestCheckPermission_CapabilityVisitor_CollapsesToVisitorSubject(t *testing.T) {
	fga := &mockFGA{
		checkFn: func(_ context.Context, req *openfga.CheckRequest) (*openfga.CheckResponse, error) {
			want := fmt.Sprintf("visitor:%s", testVisitorUID)
			if req.TupleKey.User != want {
				t.Errorf("capability visitor must check as %s, got %s", want, req.TupleKey.User)
			}
			return &openfga.CheckResponse{Allowed: true}, nil
		},
	}
	c := newTestClient(fga)
	ctx := capabilityVisitorCtx(testVisitorUID)

	if _, err := c.CheckPermission(ctx, "pipeline", testObjectUID, "reader"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestCheckPermission_NoIdentity_Unauthenticated guards the negative
// case: a context with neither a user UID nor a recognised
// visitor/capability auth type must produce an explicit
// unauthenticated error instead of silently querying FGA with an
// empty subject (which historically returned "permission denied" and
// hid the real problem).
func TestCheckPermission_NoIdentity_Unauthenticated(t *testing.T) {
	c := newTestClient(&mockFGA{})
	ctx := ctxWithHeaders(map[string]string{})

	_, err := c.CheckPermission(ctx, "pipeline", testObjectUID, "reader")
	if err == nil {
		t.Fatal("missing identity should surface as an error, not a silent deny")
	}
	if !errors.Is(err, errorsx.ErrUnauthenticated) {
		t.Errorf("expected ErrUnauthenticated, got %v", err)
	}
}

// TestListPermissions_DualAuth_UsesUserSubject mirrors the
// CheckPermission dual-auth case for the streaming list API, since
// both functions share resolveACLSubject and we want each public
// entry point to have explicit regression coverage.
func TestListPermissions_DualAuth_UsesUserSubject(t *testing.T) {
	fga := &mockFGA{
		streamedListObjectsFn: func(_ context.Context, req *openfga.StreamedListObjectsRequest) (openfga.OpenFGAService_StreamedListObjectsClient, error) {
			want := fmt.Sprintf("user:%s", testUserUID)
			if req.User != want {
				t.Errorf("dual-auth must list as %s, got %s", want, req.User)
			}
			return &mockStream{items: nil}, nil
		},
	}
	c := newTestClient(fga)
	ctx := dualAuthCtx(testUserUID)

	if _, err := c.ListPermissions(ctx, "pipeline", "reader"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ============================================================
// CheckPublicExecutable
// ============================================================

func TestCheckPublicExecutable_Allowed(t *testing.T) {
	fga := &mockFGA{
		checkFn: func(_ context.Context, req *openfga.CheckRequest) (*openfga.CheckResponse, error) {
			if req.TupleKey.User != "user:*" {
				t.Errorf("public check should use user:*, got: %s", req.TupleKey.User)
			}
			if req.TupleKey.Relation != "executor" {
				t.Errorf("should check executor relation, got: %s", req.TupleKey.Relation)
			}
			return &openfga.CheckResponse{Allowed: true}, nil
		},
	}
	c := newTestClient(fga)

	allowed, err := c.CheckPublicExecutable(context.Background(), "pipeline", testObjectUID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !allowed {
		t.Error("public executable should be allowed")
	}
}

func TestCheckPublicExecutable_CachedResultSkipsFGA(t *testing.T) {
	fgaCalls := 0
	fga := &mockFGA{
		checkFn: func(_ context.Context, _ *openfga.CheckRequest) (*openfga.CheckResponse, error) {
			fgaCalls++
			return &openfga.CheckResponse{Allowed: true}, nil
		},
	}
	c, mr := newTestClientWithCache(fga)
	defer mr.Close()

	// First call hits FGA
	_, _ = c.CheckPublicExecutable(context.Background(), "pipeline", testObjectUID)
	// Second call should use cache
	allowed, err := c.CheckPublicExecutable(context.Background(), "pipeline", testObjectUID)
	if err != nil || !allowed {
		t.Fatalf("unexpected: err=%v, allowed=%v", err, allowed)
	}
	if fgaCalls != 1 {
		t.Errorf("expected 1 FGA call (second should be cached), got %d", fgaCalls)
	}
}

func TestCheckPublicExecutable_NotPublic(t *testing.T) {
	fga := &mockFGA{
		checkFn: func(_ context.Context, _ *openfga.CheckRequest) (*openfga.CheckResponse, error) {
			return &openfga.CheckResponse{Allowed: false}, nil
		},
	}
	c := newTestClient(fga)

	allowed, err := c.CheckPublicExecutable(context.Background(), "pipeline", testObjectUID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if allowed {
		t.Error("non-public pipeline should not be executable")
	}
}

// ============================================================
// DeletePublicPermission — revoke public access
// ============================================================

func TestDeletePublicPermission_RemovesBothUserAndVisitor(t *testing.T) {
	var deletedUsers []string

	fga := &mockFGA{
		writeFn: func(_ context.Context, req *openfga.WriteRequest) (*openfga.WriteResponse, error) {
			if req.Deletes != nil {
				for _, tk := range req.Deletes.TupleKeys {
					deletedUsers = append(deletedUsers, tk.User)
				}
			}
			return &openfga.WriteResponse{}, nil
		},
	}
	c := newTestClient(fga)

	err := c.DeletePublicPermission(context.Background(), "knowledgebase", testObjectUID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	hasUser, hasVisitor := false, false
	for _, u := range deletedUsers {
		if u == "user:*" {
			hasUser = true
		}
		if u == "visitor:*" {
			hasVisitor = true
		}
	}
	if !hasUser || !hasVisitor {
		t.Errorf("should delete both user:* and visitor:*, deleted: %v", deletedUsers)
	}
}

// ============================================================
// IsUserPinned / PinUserForConsistency — read-after-write
// ============================================================

func TestIsUserPinned_NoRedisReturnsFalse(t *testing.T) {
	c := newTestClient(&mockFGA{})
	ctx := userCtx(testUserUID)

	if c.IsUserPinned(ctx) {
		t.Error("without Redis, user should never be pinned")
	}
}

func TestIsUserPinned_UnpinnedUserReturnsFalse(t *testing.T) {
	c, mr := newTestClientWithCache(&mockFGA{})
	defer mr.Close()
	ctx := userCtx(testUserUID)

	if c.IsUserPinned(ctx) {
		t.Error("user should not be pinned by default")
	}
}

func TestPinUserForConsistency_PinnedUserIsDetected(t *testing.T) {
	c, mr := newTestClientWithCache(&mockFGA{})
	defer mr.Close()
	c.config.Replica.ReplicationTimeFrame = 30
	ctx := userCtx(testUserUID)

	c.PinUserForConsistency(ctx)

	if !c.IsUserPinned(ctx) {
		t.Error("after pinning, user should be detected as pinned")
	}
}

func TestPinUserForConsistency_NoRedisIsNoop(t *testing.T) {
	c := newTestClient(&mockFGA{})
	ctx := userCtx(testUserUID)

	// Should not panic
	c.PinUserForConsistency(ctx)
}

func TestPinUserForConsistency_ZeroTimeFrameIsNoop(t *testing.T) {
	c, mr := newTestClientWithCache(&mockFGA{})
	defer mr.Close()
	c.config.Replica.ReplicationTimeFrame = 0
	ctx := userCtx(testUserUID)

	c.PinUserForConsistency(ctx)

	if c.IsUserPinned(ctx) {
		t.Error("zero replication time frame should not pin user")
	}
}

// ============================================================
// getClient — read/write routing with pinning
// ============================================================

func TestGetClient_PinnedUserReadsFromPrimary(t *testing.T) {
	primary := &mockFGA{}
	replica := &mockFGA{}

	mr := miniredis.RunT(t)
	rc := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	c := &ACLClient{
		writeClient:  primary,
		readClient:   replica,
		redisClient:  rc,
		storeID:      testStoreID,
		modelID:      testModelID,
		cacheEnabled: true,
		cacheTTL:     60 * time.Second,
		config:       Config{Replica: ReplicaConfig{ReplicationTimeFrame: 30}},
	}
	ctx := userCtx(testUserUID)

	// Pin the user
	c.PinUserForConsistency(ctx)

	// Read should go to primary (writeClient), not replica
	client := c.getClient(ctx, ReadMode)
	if client != primary {
		t.Error("pinned user should read from primary, not replica")
	}
}

func TestGetClient_UnpinnedUserReadsFromReplica(t *testing.T) {
	primary := &mockFGA{}
	replica := &mockFGA{}

	mr := miniredis.RunT(t)
	rc := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	c := &ACLClient{
		writeClient: primary,
		readClient:  replica,
		redisClient: rc,
		storeID:     testStoreID,
		modelID:     testModelID,
	}
	ctx := userCtx(testUserUID)

	client := c.getClient(ctx, ReadMode)
	if client != replica {
		t.Error("unpinned user should read from replica")
	}
}

func TestGetClient_WriteAlwaysGoesToPrimary(t *testing.T) {
	primary := &mockFGA{}
	replica := &mockFGA{}

	c := &ACLClient{
		writeClient: primary,
		readClient:  replica,
		storeID:     testStoreID,
		modelID:     testModelID,
	}
	ctx := userCtx(testUserUID)

	client := c.getClient(ctx, WriteMode)
	if client != primary {
		t.Error("writes should always go to primary")
	}
}

// ============================================================
// Cache invalidation after permission changes
// ============================================================

func TestSetResourcePermission_InvalidatesCacheAfterGrant(t *testing.T) {
	fga := &mockFGA{
		checkFn: func(_ context.Context, _ *openfga.CheckRequest) (*openfga.CheckResponse, error) {
			return &openfga.CheckResponse{Allowed: false}, nil
		},
		writeFn: func(_ context.Context, _ *openfga.WriteRequest) (*openfga.WriteResponse, error) {
			return &openfga.WriteResponse{}, nil
		},
	}
	c, mr := newTestClientWithCache(fga)
	defer mr.Close()
	ctx := userCtx(testUserUID)

	// Cache a "denied" result
	granted, _ := c.CheckPermission(ctx, "file", testObjectUID, "reader")
	if granted {
		t.Fatal("should be denied initially")
	}

	// Grant permission — should invalidate cache
	_ = c.SetResourcePermission(context.Background(), "file", testObjectUID, "user:"+testUserUID, "reader", true)

	// Now update mock to return allowed
	fga.checkFn = func(_ context.Context, _ *openfga.CheckRequest) (*openfga.CheckResponse, error) {
		return &openfga.CheckResponse{Allowed: true}, nil
	}

	// Next check should hit FGA (cache invalidated), not return stale "denied"
	granted, _ = c.CheckPermission(ctx, "file", testObjectUID, "reader")
	if !granted {
		t.Error("after granting permission and invalidating cache, should be allowed")
	}
}

// ============================================================
// SetOwner — error propagation
// ============================================================

func TestSetOwner_WriteErrorPropagated(t *testing.T) {
	fga := &mockFGA{
		readFn: func(_ context.Context, _ *openfga.ReadRequest) (*openfga.ReadResponse, error) {
			return &openfga.ReadResponse{Tuples: []*openfga.Tuple{}}, nil
		},
		writeFn: func(_ context.Context, _ *openfga.WriteRequest) (*openfga.WriteResponse, error) {
			return nil, fmt.Errorf("FGA write failed")
		},
	}
	c := newTestClient(fga)

	err := c.SetOwner(context.Background(), "pipeline", testObjectUID, "user", uuid.Must(uuid.FromString(testUserUID)))
	if err == nil {
		t.Fatal("write error should propagate")
	}
}

func TestSetOwner_ReadErrorPropagated(t *testing.T) {
	fga := &mockFGA{
		readFn: func(_ context.Context, _ *openfga.ReadRequest) (*openfga.ReadResponse, error) {
			return nil, fmt.Errorf("FGA read failed")
		},
	}
	c := newTestClient(fga)

	err := c.SetOwner(context.Background(), "pipeline", testObjectUID, "user", uuid.Must(uuid.FromString(testUserUID)))
	if err == nil {
		t.Fatal("read error should propagate")
	}
}

// ============================================================
// Config
// ============================================================

// TestDefaultCacheConfig pins the post-cache-split contract:
//   - CheckPermission Redis cache OFF by default. OpenFGA's own
//     in-memory check cache (OPENFGA_CHECK_QUERY_CACHE_ENABLED) covers
//     the hot path, so the extra Redis hop is wasted latency.
//   - ListPermissions Redis cache ON by default. StreamedListObjects
//     is graph-walking and seconds-slow under load, so caching the
//     resolved object set in Redis is a measurable win even when the
//     OpenFGA check cache is active.
//   - TTL stays at 60s for both layers.
func TestDefaultCacheConfig(t *testing.T) {
	cfg := DefaultCacheConfig()
	if cfg.Enabled {
		t.Error("default CheckPermission cache should be disabled (OpenFGA server cache suffices)")
	}
	if !cfg.ListPermissionsEnabled {
		t.Error("default ListPermissions cache should be enabled (StreamedListObjects is expensive)")
	}
	if cfg.TTL != 60 {
		t.Errorf("default TTL should be 60, got %d", cfg.TTL)
	}
}

// TestDefaultListObjectsConfig pins the truncation-guard defaults
// against the OpenFGA server defaults shipped in instill-core. If
// either side moves, both must move together; otherwise the
// truncation heuristic fires on the wrong threshold.
func TestDefaultListObjectsConfig(t *testing.T) {
	cfg := DefaultListObjectsConfig()
	if cfg.Deadline != 3*time.Second {
		t.Errorf("default deadline should be 3s (mirrors OPENFGA_LIST_OBJECTS_DEADLINE), got %v", cfg.Deadline)
	}
	if cfg.MaxResults != 1000 {
		t.Errorf("default max results should be 1000 (mirrors OPENFGA_LIST_OBJECTS_MAX_RESULTS), got %d", cfg.MaxResults)
	}
	if cfg.Slack != 200*time.Millisecond {
		t.Errorf("default slack should be 200ms (jitter absorber), got %v", cfg.Slack)
	}
}

func TestCacheTTLDuration_DefaultsTo60Seconds(t *testing.T) {
	cfg := CacheConfig{TTL: 0}
	if cfg.CacheTTLDuration() != 60*time.Second {
		t.Errorf("zero TTL should default to 60s, got %v", cfg.CacheTTLDuration())
	}
}

func TestCacheTTLDuration_CustomValue(t *testing.T) {
	cfg := CacheConfig{TTL: 120}
	if cfg.CacheTTLDuration() != 120*time.Second {
		t.Errorf("expected 120s, got %v", cfg.CacheTTLDuration())
	}
}

// ============================================================
// Utility functions
// ============================================================

func TestPermissionCacheKey_Format(t *testing.T) {
	key := permissionCacheKey("user", testUserUID, "pipeline", testObjectUID.String(), "reader")
	expected := fmt.Sprintf("acl:perm:user:%s:pipeline:%s:reader", testUserUID, testObjectUID)
	if key != expected {
		t.Errorf("cache key mismatch:\n  got:  %s\n  want: %s", key, expected)
	}
}

func TestGetModelID_ReturnsCachedValue(t *testing.T) {
	c := newTestClient(&mockFGA{})
	if c.GetModelID() != testModelID {
		t.Errorf("expected %s, got %s", testModelID, c.GetModelID())
	}
}

func TestGetStoreID_ReturnsCachedValue(t *testing.T) {
	c := newTestClient(&mockFGA{})
	if c.GetStoreID() != testStoreID {
		t.Errorf("expected %s, got %s", testStoreID, c.GetStoreID())
	}
}

func TestGetAuthorizationModelID_EmptyReturnsError(t *testing.T) {
	c := &ACLClient{modelID: ""}
	_, err := c.getAuthorizationModelID(context.Background())
	if err == nil {
		t.Fatal("empty model ID should return error")
	}
}

// ============================================================
// ListPermissions — Redis cache
// ============================================================

func TestListPermissions_CacheHitSkipsFGA(t *testing.T) {
	uid1 := uuid.Must(uuid.NewV4())
	uid2 := uuid.Must(uuid.NewV4())
	fgaCalls := 0

	fga := &mockFGA{
		streamedListObjectsFn: func(_ context.Context, _ *openfga.StreamedListObjectsRequest) (openfga.OpenFGAService_StreamedListObjectsClient, error) {
			fgaCalls++
			return &mockStream{
				items: []*openfga.StreamedListObjectsResponse{
					{Object: fmt.Sprintf("file:%s", uid1)},
					{Object: fmt.Sprintf("file:%s", uid2)},
				},
			}, nil
		},
	}
	c, mr := newTestClientWithCache(fga)
	defer mr.Close()
	ctx := userCtx(testUserUID)

	// First call: cache miss, hits FGA
	uids, err := c.ListPermissions(ctx, "file", "can_read_file")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(uids) != 2 {
		t.Fatalf("expected 2 results, got %d", len(uids))
	}
	if fgaCalls != 1 {
		t.Fatalf("expected 1 FGA call, got %d", fgaCalls)
	}

	// Second call: cache hit, no FGA call
	uids, err = c.ListPermissions(ctx, "file", "can_read_file")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(uids) != 2 {
		t.Fatalf("expected 2 cached results, got %d", len(uids))
	}
	if fgaCalls != 1 {
		t.Errorf("expected no additional FGA call (should be cached), got %d total", fgaCalls)
	}
}

func TestListPermissions_CacheEmptyResult(t *testing.T) {
	fgaCalls := 0
	fga := &mockFGA{
		streamedListObjectsFn: func(_ context.Context, _ *openfga.StreamedListObjectsRequest) (openfga.OpenFGAService_StreamedListObjectsClient, error) {
			fgaCalls++
			return &mockStream{items: nil}, nil
		},
	}
	c, mr := newTestClientWithCache(fga)
	defer mr.Close()
	ctx := userCtx(testUserUID)

	// Empty results should also be cached
	uids, _ := c.ListPermissions(ctx, "file", "can_read_file")
	if len(uids) != 0 {
		t.Fatalf("expected 0 results, got %d", len(uids))
	}

	uids, _ = c.ListPermissions(ctx, "file", "can_read_file")
	if len(uids) != 0 {
		t.Fatalf("expected 0 cached results, got %d", len(uids))
	}
	if fgaCalls != 1 {
		t.Errorf("empty result should be cached: expected 1 FGA call, got %d", fgaCalls)
	}
}

func TestListPermissions_ForceConsistencyBypassesCache(t *testing.T) {
	fgaCalls := 0
	fga := &mockFGA{
		streamedListObjectsFn: func(_ context.Context, _ *openfga.StreamedListObjectsRequest) (openfga.OpenFGAService_StreamedListObjectsClient, error) {
			fgaCalls++
			return &mockStream{items: nil}, nil
		},
	}
	c, mr := newTestClientWithCache(fga)
	defer mr.Close()
	ctx := userCtx(testUserUID)

	// Populate cache
	_, _ = c.ListPermissions(ctx, "file", "can_read_file")

	// Force consistency should bypass cache
	ctxForce := context.WithValue(ctx, ContextKeyForceHigherConsistency, true)
	_, err := c.ListPermissions(ctxForce, "file", "can_read_file")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fgaCalls != 2 {
		t.Errorf("force consistency should bypass cache: expected 2 FGA calls, got %d", fgaCalls)
	}
}

func TestListPermissions_PinnedUserBypassesCache(t *testing.T) {
	fgaCalls := 0
	fga := &mockFGA{
		streamedListObjectsFn: func(_ context.Context, _ *openfga.StreamedListObjectsRequest) (openfga.OpenFGAService_StreamedListObjectsClient, error) {
			fgaCalls++
			return &mockStream{items: nil}, nil
		},
	}
	c, mr := newTestClientWithCache(fga)
	defer mr.Close()
	c.config.Replica.ReplicationTimeFrame = 30
	ctx := userCtx(testUserUID)

	// Populate cache
	_, _ = c.ListPermissions(ctx, "file", "can_read_file")

	// Pin the user
	c.PinUserForConsistency(ctx)

	// Pinned user should bypass cache
	_, err := c.ListPermissions(ctx, "file", "can_read_file")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fgaCalls != 2 {
		t.Errorf("pinned user should bypass cache: expected 2 FGA calls, got %d", fgaCalls)
	}
}

func TestSetResourcePermission_InvalidatesListPermissionsCache(t *testing.T) {
	fgaCalls := 0
	fga := &mockFGA{
		streamedListObjectsFn: func(_ context.Context, _ *openfga.StreamedListObjectsRequest) (openfga.OpenFGAService_StreamedListObjectsClient, error) {
			fgaCalls++
			return &mockStream{items: nil}, nil
		},
		writeFn: func(_ context.Context, _ *openfga.WriteRequest) (*openfga.WriteResponse, error) {
			return &openfga.WriteResponse{}, nil
		},
	}
	c, mr := newTestClientWithCache(fga)
	defer mr.Close()
	ctx := userCtx(testUserUID)

	// Populate cache
	_, _ = c.ListPermissions(ctx, "file", "can_read_file")
	if fgaCalls != 1 {
		t.Fatalf("expected 1 FGA call after first list, got %d", fgaCalls)
	}

	// Grant permission to the same user — should invalidate their list cache
	_ = c.SetResourcePermission(context.Background(), "file", testObjectUID, "user:"+testUserUID, "reader", true)

	// Next list should hit FGA again (cache invalidated)
	_, _ = c.ListPermissions(ctx, "file", "can_read_file")
	if fgaCalls != 2 {
		t.Errorf("after SetResourcePermission, cache should be invalidated: expected 2 FGA calls, got %d", fgaCalls)
	}
}

func TestSetPublicPermission_InvalidatesAllListPermissionsCache(t *testing.T) {
	fgaCalls := 0
	fga := &mockFGA{
		streamedListObjectsFn: func(_ context.Context, _ *openfga.StreamedListObjectsRequest) (openfga.OpenFGAService_StreamedListObjectsClient, error) {
			fgaCalls++
			return &mockStream{items: nil}, nil
		},
		writeFn: func(_ context.Context, _ *openfga.WriteRequest) (*openfga.WriteResponse, error) {
			return &openfga.WriteResponse{}, nil
		},
	}
	c, mr := newTestClientWithCache(fga)
	defer mr.Close()
	ctx := userCtx(testUserUID)

	// Populate cache for user
	_, _ = c.ListPermissions(ctx, "knowledgebase", "reader")
	if fgaCalls != 1 {
		t.Fatalf("expected 1 FGA call, got %d", fgaCalls)
	}

	// Set public permission — should invalidate all list caches for this object type
	_ = c.SetPublicPermission(context.Background(), "knowledgebase", testObjectUID)

	// Next list should hit FGA again
	_, _ = c.ListPermissions(ctx, "knowledgebase", "reader")
	if fgaCalls != 2 {
		t.Errorf("after SetPublicPermission, all list caches should be invalidated: expected 2 FGA calls, got %d", fgaCalls)
	}
}

func TestListPermissions_NoCacheWithoutRedis(t *testing.T) {
	fgaCalls := 0
	fga := &mockFGA{
		streamedListObjectsFn: func(_ context.Context, _ *openfga.StreamedListObjectsRequest) (openfga.OpenFGAService_StreamedListObjectsClient, error) {
			fgaCalls++
			return &mockStream{items: nil}, nil
		},
	}
	c := newTestClient(fga)
	ctx := userCtx(testUserUID)

	_, _ = c.ListPermissions(ctx, "file", "can_read_file")
	_, _ = c.ListPermissions(ctx, "file", "can_read_file")

	if fgaCalls != 2 {
		t.Errorf("without Redis, every call should hit FGA: expected 2, got %d", fgaCalls)
	}
}

func TestListPermissionsCacheKey_Format(t *testing.T) {
	key := listPermissionsCacheKey("user", testUserUID, "file", "can_read_file")
	expected := fmt.Sprintf("acl:list:user:%s:file:can_read_file", testUserUID)
	if key != expected {
		t.Errorf("cache key mismatch:\n  got:  %s\n  want: %s", key, expected)
	}
}

func TestListPermissions_DifferentUsersCacheIndependently(t *testing.T) {
	userA := "aaaa-aaaa-aaaa-aaaa"
	userB := "bbbb-bbbb-bbbb-bbbb"
	uidForA := uuid.Must(uuid.NewV4())
	uidForB := uuid.Must(uuid.NewV4())

	fga := &mockFGA{
		streamedListObjectsFn: func(_ context.Context, req *openfga.StreamedListObjectsRequest) (openfga.OpenFGAService_StreamedListObjectsClient, error) {
			if req.User == fmt.Sprintf("user:%s", userA) {
				return &mockStream{items: []*openfga.StreamedListObjectsResponse{
					{Object: fmt.Sprintf("file:%s", uidForA)},
				}}, nil
			}
			return &mockStream{items: []*openfga.StreamedListObjectsResponse{
				{Object: fmt.Sprintf("file:%s", uidForB)},
			}}, nil
		},
	}
	c, mr := newTestClientWithCache(fga)
	defer mr.Close()

	uidsA, _ := c.ListPermissions(userCtx(userA), "file", "reader")
	uidsB, _ := c.ListPermissions(userCtx(userB), "file", "reader")

	if len(uidsA) != 1 || uidsA[0] != uidForA {
		t.Errorf("user A should see uidForA, got %v", uidsA)
	}
	if len(uidsB) != 1 || uidsB[0] != uidForB {
		t.Errorf("user B should see uidForB, got %v", uidsB)
	}
}

func TestListPermissions_DifferentObjectTypesAndRolesCacheIndependently(t *testing.T) {
	fgaCalls := 0
	fga := &mockFGA{
		streamedListObjectsFn: func(_ context.Context, _ *openfga.StreamedListObjectsRequest) (openfga.OpenFGAService_StreamedListObjectsClient, error) {
			fgaCalls++
			return &mockStream{items: nil}, nil
		},
	}
	c, mr := newTestClientWithCache(fga)
	defer mr.Close()
	ctx := userCtx(testUserUID)

	_, _ = c.ListPermissions(ctx, "file", "can_read_file")
	_, _ = c.ListPermissions(ctx, "collection", "reader")
	_, _ = c.ListPermissions(ctx, "file", "writer")

	if fgaCalls != 3 {
		t.Errorf("different objectType/role combos should each hit FGA: expected 3, got %d", fgaCalls)
	}

	// Repeat all three — all should be cached
	_, _ = c.ListPermissions(ctx, "file", "can_read_file")
	_, _ = c.ListPermissions(ctx, "collection", "reader")
	_, _ = c.ListPermissions(ctx, "file", "writer")

	if fgaCalls != 3 {
		t.Errorf("repeated calls should all hit cache: expected 3 total, got %d", fgaCalls)
	}
}

func TestListPublicPermissions_UsesCacheWithWildcardSubject(t *testing.T) {
	fgaCalls := 0
	uid1 := uuid.Must(uuid.NewV4())
	fga := &mockFGA{
		streamedListObjectsFn: func(_ context.Context, req *openfga.StreamedListObjectsRequest) (openfga.OpenFGAService_StreamedListObjectsClient, error) {
			fgaCalls++
			if req.User != "user:*" {
				t.Errorf("ListPublicPermissions should query as user:*, got %s", req.User)
			}
			return &mockStream{items: []*openfga.StreamedListObjectsResponse{
				{Object: fmt.Sprintf("collection:%s", uid1)},
			}}, nil
		},
	}
	c, mr := newTestClientWithCache(fga)
	defer mr.Close()
	ctx := userCtx(testUserUID)

	uids, _ := c.ListPublicPermissions(ctx, "collection", "reader")
	if len(uids) != 1 {
		t.Fatalf("expected 1 result, got %d", len(uids))
	}

	// Second call should use cache
	uids, _ = c.ListPublicPermissions(ctx, "collection", "reader")
	if fgaCalls != 1 {
		t.Errorf("ListPublicPermissions should use cache on second call: expected 1 FGA call, got %d", fgaCalls)
	}
	if len(uids) != 1 || uids[0] != uid1 {
		t.Errorf("cached result should match original, got %v", uids)
	}
}

func TestDeleteResourcePermission_InvalidatesListPermissionsCache(t *testing.T) {
	fgaCalls := 0
	fga := &mockFGA{
		streamedListObjectsFn: func(_ context.Context, _ *openfga.StreamedListObjectsRequest) (openfga.OpenFGAService_StreamedListObjectsClient, error) {
			fgaCalls++
			return &mockStream{items: nil}, nil
		},
		writeFn: func(_ context.Context, _ *openfga.WriteRequest) (*openfga.WriteResponse, error) {
			return &openfga.WriteResponse{}, nil
		},
	}
	c, mr := newTestClientWithCache(fga)
	defer mr.Close()
	ctx := userCtx(testUserUID)

	// Populate cache
	_, _ = c.ListPermissions(ctx, "file", "can_read_file")

	// Delete permission — should invalidate cache
	_ = c.DeleteResourcePermission(context.Background(), "file", testObjectUID, "user:"+testUserUID)

	_, _ = c.ListPermissions(ctx, "file", "can_read_file")
	if fgaCalls != 2 {
		t.Errorf("after DeleteResourcePermission, cache should be invalidated: expected 2 FGA calls, got %d", fgaCalls)
	}
}

func TestPurge_InvalidatesListPermissionsCache(t *testing.T) {
	fgaCalls := 0
	fga := &mockFGA{
		streamedListObjectsFn: func(_ context.Context, _ *openfga.StreamedListObjectsRequest) (openfga.OpenFGAService_StreamedListObjectsClient, error) {
			fgaCalls++
			return &mockStream{items: nil}, nil
		},
		readFn: func(_ context.Context, _ *openfga.ReadRequest) (*openfga.ReadResponse, error) {
			return &openfga.ReadResponse{Tuples: []*openfga.Tuple{}}, nil
		},
	}
	c, mr := newTestClientWithCache(fga)
	defer mr.Close()
	ctx := userCtx(testUserUID)

	// Populate cache
	_, _ = c.ListPermissions(ctx, "file", "can_read_file")

	// Purge — should invalidate all list caches for this object type
	_ = c.Purge(context.Background(), "file", testObjectUID)

	_, _ = c.ListPermissions(ctx, "file", "can_read_file")
	if fgaCalls != 2 {
		t.Errorf("after Purge, cache should be invalidated: expected 2 FGA calls, got %d", fgaCalls)
	}
}

func TestListPermissions_FGAErrorNotCached(t *testing.T) {
	fgaCalls := 0
	fga := &mockFGA{
		streamedListObjectsFn: func(_ context.Context, _ *openfga.StreamedListObjectsRequest) (openfga.OpenFGAService_StreamedListObjectsClient, error) {
			fgaCalls++
			if fgaCalls == 1 {
				return nil, fmt.Errorf("FGA unavailable")
			}
			return &mockStream{items: nil}, nil
		},
	}
	c, mr := newTestClientWithCache(fga)
	defer mr.Close()
	ctx := userCtx(testUserUID)

	// First call fails — should NOT be cached
	_, err := c.ListPermissions(ctx, "file", "can_read_file")
	if err == nil {
		t.Fatal("expected error from FGA")
	}

	// Second call should hit FGA again (error was not cached)
	_, err = c.ListPermissions(ctx, "file", "can_read_file")
	if err != nil {
		t.Fatalf("second call should succeed: %v", err)
	}
	if fgaCalls != 2 {
		t.Errorf("FGA error should not be cached: expected 2 FGA calls, got %d", fgaCalls)
	}
}

func TestListPermissions_CachedUIDsMatchOriginal(t *testing.T) {
	uid1 := uuid.Must(uuid.FromString("11111111-1111-1111-1111-111111111111"))
	uid2 := uuid.Must(uuid.FromString("22222222-2222-2222-2222-222222222222"))
	uid3 := uuid.Must(uuid.FromString("33333333-3333-3333-3333-333333333333"))

	fga := &mockFGA{
		streamedListObjectsFn: func(_ context.Context, _ *openfga.StreamedListObjectsRequest) (openfga.OpenFGAService_StreamedListObjectsClient, error) {
			return &mockStream{
				items: []*openfga.StreamedListObjectsResponse{
					{Object: fmt.Sprintf("file:%s", uid1)},
					{Object: fmt.Sprintf("file:%s", uid2)},
					{Object: fmt.Sprintf("file:%s", uid3)},
				},
			}, nil
		},
	}
	c, mr := newTestClientWithCache(fga)
	defer mr.Close()
	ctx := userCtx(testUserUID)

	// Populate cache
	_, _ = c.ListPermissions(ctx, "file", "reader")

	// Read from cache and verify exact content + order
	cached, err := c.ListPermissions(ctx, "file", "reader")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := []uuid.UUID{uid1, uid2, uid3}
	if len(cached) != len(expected) {
		t.Fatalf("expected %d UIDs, got %d", len(expected), len(cached))
	}
	for i, u := range cached {
		if u != expected[i] {
			t.Errorf("UID[%d]: expected %s, got %s", i, expected[i], u)
		}
	}
}

func TestSetResourcePermission_OnlyInvalidatesTargetUser(t *testing.T) {
	userA := testUserUID
	userB := "bbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	fgaCalls := 0

	fga := &mockFGA{
		streamedListObjectsFn: func(_ context.Context, _ *openfga.StreamedListObjectsRequest) (openfga.OpenFGAService_StreamedListObjectsClient, error) {
			fgaCalls++
			return &mockStream{items: nil}, nil
		},
		writeFn: func(_ context.Context, _ *openfga.WriteRequest) (*openfga.WriteResponse, error) {
			return &openfga.WriteResponse{}, nil
		},
	}
	c, mr := newTestClientWithCache(fga)
	defer mr.Close()

	// Populate cache for both users
	_, _ = c.ListPermissions(userCtx(userA), "file", "can_read_file")
	_, _ = c.ListPermissions(userCtx(userB), "file", "can_read_file")
	if fgaCalls != 2 {
		t.Fatalf("expected 2 FGA calls, got %d", fgaCalls)
	}

	// Grant permission to User B only
	_ = c.SetResourcePermission(context.Background(), "file", testObjectUID, "user:"+userB, "reader", true)

	// User A's cache should still be valid (no FGA call)
	_, _ = c.ListPermissions(userCtx(userA), "file", "can_read_file")
	if fgaCalls != 2 {
		t.Errorf("user A's cache should not be invalidated by granting to user B: expected 2 FGA calls, got %d", fgaCalls)
	}

	// User B's cache should be invalidated (triggers new FGA call)
	_, _ = c.ListPermissions(userCtx(userB), "file", "can_read_file")
	if fgaCalls != 3 {
		t.Errorf("user B's cache should be invalidated after grant: expected 3 FGA calls, got %d", fgaCalls)
	}
}

func TestSetPublicPermission_InvalidatesMultipleUsersCaches(t *testing.T) {
	userA := testUserUID
	userB := "bbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	fgaCalls := 0

	fga := &mockFGA{
		streamedListObjectsFn: func(_ context.Context, _ *openfga.StreamedListObjectsRequest) (openfga.OpenFGAService_StreamedListObjectsClient, error) {
			fgaCalls++
			return &mockStream{items: nil}, nil
		},
		writeFn: func(_ context.Context, _ *openfga.WriteRequest) (*openfga.WriteResponse, error) {
			return &openfga.WriteResponse{}, nil
		},
	}
	c, mr := newTestClientWithCache(fga)
	defer mr.Close()

	// Populate cache for user A, user B, and a visitor
	_, _ = c.ListPermissions(userCtx(userA), "file", "reader")
	_, _ = c.ListPermissions(userCtx(userB), "file", "reader")
	_, _ = c.ListPermissions(visitorCtx(testVisitorUID), "file", "reader")
	if fgaCalls != 3 {
		t.Fatalf("expected 3 FGA calls, got %d", fgaCalls)
	}

	// Set public permission on file — wildcard grant affects everyone
	_ = c.SetPublicPermission(context.Background(), "file", testObjectUID)

	// All three caches should be invalidated
	_, _ = c.ListPermissions(userCtx(userA), "file", "reader")
	_, _ = c.ListPermissions(userCtx(userB), "file", "reader")
	_, _ = c.ListPermissions(visitorCtx(testVisitorUID), "file", "reader")
	if fgaCalls != 6 {
		t.Errorf("SetPublicPermission should invalidate all users' caches: expected 6 FGA calls, got %d", fgaCalls)
	}
}

func TestDeletePublicPermission_InvalidatesAllUsersCaches(t *testing.T) {
	fgaCalls := 0

	fga := &mockFGA{
		streamedListObjectsFn: func(_ context.Context, _ *openfga.StreamedListObjectsRequest) (openfga.OpenFGAService_StreamedListObjectsClient, error) {
			fgaCalls++
			return &mockStream{items: nil}, nil
		},
		writeFn: func(_ context.Context, _ *openfga.WriteRequest) (*openfga.WriteResponse, error) {
			return &openfga.WriteResponse{}, nil
		},
	}
	c, mr := newTestClientWithCache(fga)
	defer mr.Close()

	// Populate cache
	_, _ = c.ListPermissions(userCtx(testUserUID), "collection", "reader")
	if fgaCalls != 1 {
		t.Fatalf("expected 1 FGA call, got %d", fgaCalls)
	}

	// Revoke public access
	_ = c.DeletePublicPermission(context.Background(), "collection", testObjectUID)

	// Cache should be invalidated
	_, _ = c.ListPermissions(userCtx(testUserUID), "collection", "reader")
	if fgaCalls != 2 {
		t.Errorf("DeletePublicPermission should invalidate caches: expected 2, got %d", fgaCalls)
	}
}

// TestSetResourcePermission_InvalidatesAllObjectTypesForUser verifies that
// per-user invalidation clears all objectType caches for that user, not just
// the objectType being modified. This is intentional: FGA graph resolution
// can cross object types (e.g., collection viewer → file viewer via
// parent_collection), so a grant on "file" can affect the user's resolved
// "collection" permissions. Broad per-user invalidation is the safe default.
func TestSetResourcePermission_InvalidatesAllObjectTypesForUser(t *testing.T) {
	fgaCalls := 0

	fga := &mockFGA{
		streamedListObjectsFn: func(_ context.Context, _ *openfga.StreamedListObjectsRequest) (openfga.OpenFGAService_StreamedListObjectsClient, error) {
			fgaCalls++
			return &mockStream{items: nil}, nil
		},
		writeFn: func(_ context.Context, _ *openfga.WriteRequest) (*openfga.WriteResponse, error) {
			return &openfga.WriteResponse{}, nil
		},
	}
	c, mr := newTestClientWithCache(fga)
	defer mr.Close()
	ctx := userCtx(testUserUID)

	// Populate cache for both file and collection
	_, _ = c.ListPermissions(ctx, "file", "can_read_file")
	_, _ = c.ListPermissions(ctx, "collection", "reader")
	if fgaCalls != 2 {
		t.Fatalf("expected 2 FGA calls, got %d", fgaCalls)
	}

	// Grant file permission — per-user invalidation clears ALL of this user's caches
	_ = c.SetResourcePermission(context.Background(), "file", testObjectUID, "user:"+testUserUID, "reader", true)

	// Both caches should be invalidated (broad user-level invalidation)
	_, _ = c.ListPermissions(ctx, "file", "can_read_file")
	_, _ = c.ListPermissions(ctx, "collection", "reader")
	if fgaCalls != 4 {
		t.Errorf("per-user invalidation should clear all object types: expected 4 FGA calls, got %d", fgaCalls)
	}
}

func TestPurge_DoesNotInvalidateOtherObjectTypes(t *testing.T) {
	fgaCalls := 0

	fga := &mockFGA{
		streamedListObjectsFn: func(_ context.Context, _ *openfga.StreamedListObjectsRequest) (openfga.OpenFGAService_StreamedListObjectsClient, error) {
			fgaCalls++
			return &mockStream{items: nil}, nil
		},
		readFn: func(_ context.Context, _ *openfga.ReadRequest) (*openfga.ReadResponse, error) {
			return &openfga.ReadResponse{Tuples: []*openfga.Tuple{}}, nil
		},
	}
	c, mr := newTestClientWithCache(fga)
	defer mr.Close()
	ctx := userCtx(testUserUID)

	// Populate cache for file and collection
	_, _ = c.ListPermissions(ctx, "file", "reader")
	_, _ = c.ListPermissions(ctx, "collection", "reader")
	if fgaCalls != 2 {
		t.Fatalf("expected 2 FGA calls, got %d", fgaCalls)
	}

	// Purge a file object — should only invalidate file caches
	_ = c.Purge(context.Background(), "file", testObjectUID)

	// Collection cache should still be valid
	_, _ = c.ListPermissions(ctx, "collection", "reader")
	if fgaCalls != 2 {
		t.Errorf("purge file should not invalidate collection cache: expected 2, got %d", fgaCalls)
	}

	// File cache should be invalidated
	_, _ = c.ListPermissions(ctx, "file", "reader")
	if fgaCalls != 3 {
		t.Errorf("file cache should be invalidated: expected 3, got %d", fgaCalls)
	}
}

func TestDeleteResourcePermission_OnlyInvalidatesTargetUser(t *testing.T) {
	userA := testUserUID
	userB := "bbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	fgaCalls := 0

	fga := &mockFGA{
		streamedListObjectsFn: func(_ context.Context, _ *openfga.StreamedListObjectsRequest) (openfga.OpenFGAService_StreamedListObjectsClient, error) {
			fgaCalls++
			return &mockStream{items: nil}, nil
		},
		writeFn: func(_ context.Context, _ *openfga.WriteRequest) (*openfga.WriteResponse, error) {
			return &openfga.WriteResponse{}, nil
		},
	}
	c, mr := newTestClientWithCache(fga)
	defer mr.Close()

	// Populate cache for both users
	_, _ = c.ListPermissions(userCtx(userA), "file", "reader")
	_, _ = c.ListPermissions(userCtx(userB), "file", "reader")
	if fgaCalls != 2 {
		t.Fatalf("expected 2 FGA calls, got %d", fgaCalls)
	}

	// Delete permission for User B
	_ = c.DeleteResourcePermission(context.Background(), "file", testObjectUID, "user:"+userB)

	// User A's cache should still be valid
	_, _ = c.ListPermissions(userCtx(userA), "file", "reader")
	if fgaCalls != 2 {
		t.Errorf("user A cache should be preserved after deleting user B's permission: expected 2, got %d", fgaCalls)
	}

	// User B's cache should be invalidated
	_, _ = c.ListPermissions(userCtx(userB), "file", "reader")
	if fgaCalls != 3 {
		t.Errorf("user B cache should be invalidated: expected 3, got %d", fgaCalls)
	}
}

func TestSetPublicPermission_DoesNotInvalidateOtherObjectTypes(t *testing.T) {
	fgaCalls := 0

	fga := &mockFGA{
		streamedListObjectsFn: func(_ context.Context, _ *openfga.StreamedListObjectsRequest) (openfga.OpenFGAService_StreamedListObjectsClient, error) {
			fgaCalls++
			return &mockStream{items: nil}, nil
		},
		writeFn: func(_ context.Context, _ *openfga.WriteRequest) (*openfga.WriteResponse, error) {
			return &openfga.WriteResponse{}, nil
		},
	}
	c, mr := newTestClientWithCache(fga)
	defer mr.Close()
	ctx := userCtx(testUserUID)

	// Populate cache for file and collection
	_, _ = c.ListPermissions(ctx, "file", "reader")
	_, _ = c.ListPermissions(ctx, "collection", "reader")
	if fgaCalls != 2 {
		t.Fatalf("expected 2 FGA calls, got %d", fgaCalls)
	}

	// Set public permission on collection — should invalidate collection caches only
	_ = c.SetPublicPermission(context.Background(), "collection", testObjectUID)

	// File cache should still be valid
	_, _ = c.ListPermissions(ctx, "file", "reader")
	if fgaCalls != 2 {
		t.Errorf("file cache should be preserved after SetPublicPermission on collection: expected 2, got %d", fgaCalls)
	}

	// Collection cache should be invalidated
	_, _ = c.ListPermissions(ctx, "collection", "reader")
	if fgaCalls != 3 {
		t.Errorf("collection cache should be invalidated: expected 3, got %d", fgaCalls)
	}
}

func TestDeletePublicPermission_DoesNotInvalidateOtherObjectTypes(t *testing.T) {
	fgaCalls := 0

	fga := &mockFGA{
		streamedListObjectsFn: func(_ context.Context, _ *openfga.StreamedListObjectsRequest) (openfga.OpenFGAService_StreamedListObjectsClient, error) {
			fgaCalls++
			return &mockStream{items: nil}, nil
		},
		writeFn: func(_ context.Context, _ *openfga.WriteRequest) (*openfga.WriteResponse, error) {
			return &openfga.WriteResponse{}, nil
		},
	}
	c, mr := newTestClientWithCache(fga)
	defer mr.Close()
	ctx := userCtx(testUserUID)

	// Populate cache for file and collection
	_, _ = c.ListPermissions(ctx, "file", "reader")
	_, _ = c.ListPermissions(ctx, "collection", "reader")
	if fgaCalls != 2 {
		t.Fatalf("expected 2 FGA calls, got %d", fgaCalls)
	}

	// Delete public permission on collection
	_ = c.DeletePublicPermission(context.Background(), "collection", testObjectUID)

	// File cache should still be valid
	_, _ = c.ListPermissions(ctx, "file", "reader")
	if fgaCalls != 2 {
		t.Errorf("file cache should be preserved after DeletePublicPermission on collection: expected 2, got %d", fgaCalls)
	}

	// Collection cache should be invalidated
	_, _ = c.ListPermissions(ctx, "collection", "reader")
	if fgaCalls != 3 {
		t.Errorf("collection cache should be invalidated: expected 3, got %d", fgaCalls)
	}
}

func TestSetOwner_InvalidatesNewOwnerCache(t *testing.T) {
	ownerUID := uuid.Must(uuid.FromString(testUserUID))
	otherUser := "cccc-cccc-cccc-cccc-cccccccccccc"
	fgaCalls := 0

	fga := &mockFGA{
		streamedListObjectsFn: func(_ context.Context, _ *openfga.StreamedListObjectsRequest) (openfga.OpenFGAService_StreamedListObjectsClient, error) {
			fgaCalls++
			return &mockStream{items: nil}, nil
		},
		readFn: func(_ context.Context, _ *openfga.ReadRequest) (*openfga.ReadResponse, error) {
			return &openfga.ReadResponse{Tuples: []*openfga.Tuple{}}, nil
		},
		writeFn: func(_ context.Context, _ *openfga.WriteRequest) (*openfga.WriteResponse, error) {
			return &openfga.WriteResponse{}, nil
		},
	}
	c, mr := newTestClientWithCache(fga)
	defer mr.Close()

	// Populate cache for the owner and another user
	_, _ = c.ListPermissions(userCtx(testUserUID), "pipeline", "reader")
	_, _ = c.ListPermissions(userCtx(otherUser), "pipeline", "reader")
	if fgaCalls != 2 {
		t.Fatalf("expected 2 FGA calls, got %d", fgaCalls)
	}

	// Set owner — should invalidate the new owner's cache
	_ = c.SetOwner(context.Background(), "pipeline", testObjectUID, "user", ownerUID)

	// New owner's cache should be invalidated
	_, _ = c.ListPermissions(userCtx(testUserUID), "pipeline", "reader")
	if fgaCalls != 3 {
		t.Errorf("new owner's cache should be invalidated: expected 3 FGA calls, got %d", fgaCalls)
	}

	// Other user's cache should still be valid
	_, _ = c.ListPermissions(userCtx(otherUser), "pipeline", "reader")
	if fgaCalls != 3 {
		t.Errorf("other user's cache should be preserved: expected 3 FGA calls, got %d", fgaCalls)
	}
}

func TestListPermissions_VisitorAndUserCacheIsolated(t *testing.T) {
	uidForUser := uuid.Must(uuid.NewV4())
	uidForVisitor := uuid.Must(uuid.NewV4())

	fga := &mockFGA{
		streamedListObjectsFn: func(_ context.Context, req *openfga.StreamedListObjectsRequest) (openfga.OpenFGAService_StreamedListObjectsClient, error) {
			if req.User == fmt.Sprintf("user:%s", testUserUID) {
				return &mockStream{items: []*openfga.StreamedListObjectsResponse{
					{Object: fmt.Sprintf("file:%s", uidForUser)},
				}}, nil
			}
			return &mockStream{items: []*openfga.StreamedListObjectsResponse{
				{Object: fmt.Sprintf("file:%s", uidForVisitor)},
			}}, nil
		},
	}
	c, mr := newTestClientWithCache(fga)
	defer mr.Close()

	userUids, _ := c.ListPermissions(userCtx(testUserUID), "file", "reader")
	visitorUids, _ := c.ListPermissions(visitorCtx(testVisitorUID), "file", "reader")

	if len(userUids) != 1 || userUids[0] != uidForUser {
		t.Errorf("user should see uidForUser, got %v", userUids)
	}
	if len(visitorUids) != 1 || visitorUids[0] != uidForVisitor {
		t.Errorf("visitor should see uidForVisitor, got %v", visitorUids)
	}

	// Verify cached results are still isolated
	userUids2, _ := c.ListPermissions(userCtx(testUserUID), "file", "reader")
	visitorUids2, _ := c.ListPermissions(visitorCtx(testVisitorUID), "file", "reader")

	if len(userUids2) != 1 || userUids2[0] != uidForUser {
		t.Errorf("cached user result should still be uidForUser, got %v", userUids2)
	}
	if len(visitorUids2) != 1 || visitorUids2[0] != uidForVisitor {
		t.Errorf("cached visitor result should still be uidForVisitor, got %v", visitorUids2)
	}
}

// ============================================================
// listObjectsForSubject — truncation guard
// ============================================================

// slowStream simulates a StreamedListObjects response that emits all
// items synchronously but sleeps once before EOF. Used to push the
// elapsed time across the deadline-Slack boundary so the truncation
// heuristic fires deterministically without flaky wall-clock waits.
type slowStream struct {
	grpc.ClientStream
	items    []*openfga.StreamedListObjectsResponse
	pos      int
	sleepEOF time.Duration
	hasSlept bool
}

func (s *slowStream) Recv() (*openfga.StreamedListObjectsResponse, error) {
	if s.pos >= len(s.items) {
		if !s.hasSlept && s.sleepEOF > 0 {
			s.hasSlept = true
			time.Sleep(s.sleepEOF)
		}
		return nil, io.EOF
	}
	resp := s.items[s.pos]
	s.pos++
	return resp, nil
}

// TestIsLikelyTruncated_Matrix exercises the threshold heuristic
// directly so the cache-write decision can be verified without
// spinning a mock stream. The matrix covers each branch:
//   - well below both ceilings → not truncated
//   - elapsed within Slack of Deadline → truncated (deadline branch)
//   - elapsed exactly at Deadline → truncated
//   - returned >= MaxResults → truncated (max branch)
//   - returned == MaxResults-1 with quick elapsed → not truncated
//   - zero/empty config falls back to defaults
func TestIsLikelyTruncated_Matrix(t *testing.T) {
	cfg := ListObjectsConfig{
		Deadline:   3 * time.Second,
		MaxResults: 1000,
		Slack:      200 * time.Millisecond,
	}

	cases := []struct {
		name     string
		elapsed  time.Duration
		returned int
		want     bool
	}{
		{"fast and small", 10 * time.Millisecond, 5, false},
		{"slow but well under deadline", 1500 * time.Millisecond, 10, false},
		{"within slack of deadline", 2900 * time.Millisecond, 10, true},
		{"exactly at deadline", 3 * time.Second, 10, true},
		{"max-results minus one", 50 * time.Millisecond, 999, false},
		{"max-results exactly", 50 * time.Millisecond, 1000, true},
		{"max-results plus one", 50 * time.Millisecond, 1001, true},
		{"both branches fire", 2900 * time.Millisecond, 1000, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := isLikelyTruncated(tc.elapsed, tc.returned, cfg)
			if got != tc.want {
				t.Errorf("isLikelyTruncated(elapsed=%v, returned=%d) = %v, want %v",
					tc.elapsed, tc.returned, got, tc.want)
			}
		})
	}
}

// TestIsLikelyTruncated_DefaultsFillFromZeroConfig pins the zero-
// config fallback so callers that pass an empty ListObjectsConfig
// (e.g. unit tests, embedded-mode services) still get the correct
// defaults applied to the heuristic.
func TestIsLikelyTruncated_DefaultsFillFromZeroConfig(t *testing.T) {
	if !isLikelyTruncated(3*time.Second, 0, ListObjectsConfig{}) {
		t.Error("zero config should fall back to defaults; 3s elapsed must trip deadline branch")
	}
	if !isLikelyTruncated(0, 1000, ListObjectsConfig{}) {
		t.Error("zero config should fall back to defaults; 1000 returned must trip max-results branch")
	}
	if isLikelyTruncated(10*time.Millisecond, 5, ListObjectsConfig{}) {
		t.Error("zero config should fall back to defaults; fast & small must not flag")
	}
}

// TestListPermissions_TruncationGuard_DeadlineBranch_RefusesCacheWrite
// drives listObjectsForSubject through a mock stream that sleeps once
// past the deadline-slack boundary so the elapsed-time branch fires.
// Asserts the user-visible contract: (a) the partial result is still
// returned to the caller, (b) no Redis cache entry is written, (c) a
// follow-up ListPermissions hits FGA again instead of being served
// from a poisoned cache. A `log.Warn` carrying the stable token
// `acl.list_objects_truncated` fires as a side effect (greppable in
// Cloud Logging on d0/prod where no metrics collector exists); we do
// not assert on the log line here because the cache-miss-on-next-call
// behavior is the actual stale-data-prevention contract — the log is
// the alerting surface, not the contract surface.
func TestListPermissions_TruncationGuard_DeadlineBranch_RefusesCacheWrite(t *testing.T) {
	uid1 := uuid.Must(uuid.NewV4())

	fgaCalls := 0
	fga := &mockFGA{
		streamedListObjectsFn: func(_ context.Context, _ *openfga.StreamedListObjectsRequest) (openfga.OpenFGAService_StreamedListObjectsClient, error) {
			fgaCalls++
			return &slowStream{
				items: []*openfga.StreamedListObjectsResponse{
					{Object: fmt.Sprintf("file:%s", uid1)},
				},
				sleepEOF: 250 * time.Millisecond,
			}, nil
		},
	}
	c, mr := newTestClientWithCache(fga)
	defer mr.Close()
	c.listObjectsCfg = ListObjectsConfig{
		Deadline:   200 * time.Millisecond,
		MaxResults: 1000,
		Slack:      0,
	}

	uids, err := c.ListPermissions(userCtx(testUserUID), "file", "reader")
	if err != nil {
		t.Fatalf("partial result must still be returned, got error: %v", err)
	}
	if len(uids) != 1 || uids[0] != uid1 {
		t.Errorf("partial result must include the items received before truncation, got %v", uids)
	}

	cacheKey := listPermissionsCacheKey("user", testUserUID, "file", "reader")
	if mr.Exists(cacheKey) {
		t.Error("truncated result must NOT be cached (cache key was written)")
	}

	// Second call must re-issue FGA, not serve a poisoned cache.
	if _, err := c.ListPermissions(userCtx(testUserUID), "file", "reader"); err != nil {
		t.Fatalf("follow-up call should succeed: %v", err)
	}
	if fgaCalls != 2 {
		t.Errorf("follow-up call must hit FGA again (cache must not be served), got %d FGA calls", fgaCalls)
	}
}

// TestListPermissions_TruncationGuard_MaxResultsBranch covers the
// other half of the heuristic: a fast stream that returns at least
// MaxResults items must also be flagged as truncated and skip the
// cache write. This is the branch that fires on workspaces with
// thousands of files where the deadline is never approached but the
// max-results ceiling is hit silently. Same contract assertion shape
// as the deadline branch — partial result returned, no cache, second
// call re-hits FGA.
func TestListPermissions_TruncationGuard_MaxResultsBranch(t *testing.T) {
	maxResults := 5
	items := make([]*openfga.StreamedListObjectsResponse, maxResults)
	for i := 0; i < maxResults; i++ {
		items[i] = &openfga.StreamedListObjectsResponse{
			Object: fmt.Sprintf("file:%s", uuid.Must(uuid.NewV4())),
		}
	}

	fgaCalls := 0
	fga := &mockFGA{
		streamedListObjectsFn: func(_ context.Context, _ *openfga.StreamedListObjectsRequest) (openfga.OpenFGAService_StreamedListObjectsClient, error) {
			fgaCalls++
			return &mockStream{items: items}, nil
		},
	}
	c, mr := newTestClientWithCache(fga)
	defer mr.Close()
	c.listObjectsCfg = ListObjectsConfig{
		Deadline:   3 * time.Second,
		MaxResults: maxResults,
		Slack:      200 * time.Millisecond,
	}

	uids, err := c.ListPermissions(userCtx(testUserUID), "file", "reader")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(uids) != maxResults {
		t.Errorf("expected %d items, got %d", maxResults, len(uids))
	}

	cacheKey := listPermissionsCacheKey("user", testUserUID, "file", "reader")
	if mr.Exists(cacheKey) {
		t.Error("max-results-capped result must NOT be cached")
	}

	if _, err := c.ListPermissions(userCtx(testUserUID), "file", "reader"); err != nil {
		t.Fatalf("follow-up call should succeed: %v", err)
	}
	if fgaCalls != 2 {
		t.Errorf("follow-up call must hit FGA again (cache must not be served), got %d FGA calls", fgaCalls)
	}
}

// TestListPermissions_TruncationGuard_HappyPath_CachesAndServesCachedResult
// covers the negative case: a stream that completes well below both
// ceilings must populate the cache, and a follow-up call must be
// served from cache without re-issuing FGA. Without this test a
// misconfigured Deadline / MaxResults (e.g. zero values) could
// silently turn every list call into a truncation event and drop
// cache hit rate to 0.
func TestListPermissions_TruncationGuard_HappyPath_CachesAndServesCachedResult(t *testing.T) {
	uid1 := uuid.Must(uuid.NewV4())
	uid2 := uuid.Must(uuid.NewV4())

	fgaCalls := 0
	fga := &mockFGA{
		streamedListObjectsFn: func(_ context.Context, _ *openfga.StreamedListObjectsRequest) (openfga.OpenFGAService_StreamedListObjectsClient, error) {
			fgaCalls++
			return &mockStream{items: []*openfga.StreamedListObjectsResponse{
				{Object: fmt.Sprintf("file:%s", uid1)},
				{Object: fmt.Sprintf("file:%s", uid2)},
			}}, nil
		},
	}
	c, mr := newTestClientWithCache(fga)
	defer mr.Close()
	c.listObjectsCfg = ListObjectsConfig{
		Deadline:   3 * time.Second,
		MaxResults: 1000,
		Slack:      200 * time.Millisecond,
	}

	uids, err := c.ListPermissions(userCtx(testUserUID), "file", "reader")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(uids) != 2 {
		t.Errorf("expected 2 items, got %d", len(uids))
	}

	cacheKey := listPermissionsCacheKey("user", testUserUID, "file", "reader")
	if !mr.Exists(cacheKey) {
		t.Error("complete result must be cached")
	}

	if _, err := c.ListPermissions(userCtx(testUserUID), "file", "reader"); err != nil {
		t.Fatalf("follow-up call should succeed: %v", err)
	}
	if fgaCalls != 1 {
		t.Errorf("follow-up call must be served from cache, got %d FGA calls", fgaCalls)
	}
}

// ============================================================
// ReadTuples — direct-grant lookup via OpenFGA Read API
// ============================================================

// TestReadTuples_PropagatesFilterAndReturnsTuples pins the basic
// happy path: ReadTuples threads (Object, Relation, User) through to
// the OpenFGA Read RPC verbatim and materialises the response into
// the local ReadTuple shape. Direct-grant lookups for "who has
// access to this file" rely on this single-page contract.
func TestReadTuples_PropagatesFilterAndReturnsTuples(t *testing.T) {
	objStr := fmt.Sprintf("file:%s", testObjectUID)
	userStr := fmt.Sprintf("user:%s", testUserUID)

	var seen *openfga.ReadRequest
	fga := &mockFGA{
		readFn: func(_ context.Context, req *openfga.ReadRequest) (*openfga.ReadResponse, error) {
			seen = req
			return &openfga.ReadResponse{
				Tuples: []*openfga.Tuple{
					{Key: &openfga.TupleKey{Object: objStr, Relation: "viewer", User: userStr}},
				},
			}, nil
		},
	}
	c := newTestClient(fga)

	got, err := c.ReadTuples(context.Background(), ReadTupleFilter{
		Object:   objStr,
		Relation: "viewer",
		User:     "",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if seen == nil || seen.TupleKey == nil {
		t.Fatal("Read RPC was not invoked")
	}
	if seen.TupleKey.Object != objStr || seen.TupleKey.Relation != "viewer" || seen.TupleKey.User != "" {
		t.Errorf("filter must be forwarded verbatim, got object=%q relation=%q user=%q",
			seen.TupleKey.Object, seen.TupleKey.Relation, seen.TupleKey.User)
	}
	if len(got) != 1 || got[0].Object != objStr || got[0].Relation != "viewer" || got[0].User != userStr {
		t.Errorf("ReadTuple shape mismatch, got %+v", got)
	}
}

// TestReadTuples_WalksContinuationToken pins the pagination
// contract: when the OpenFGA Read response carries a non-empty
// continuation token, ReadTuples must follow it transparently and
// return the concatenated result. Callers building "list everyone
// with direct access" UIs depend on this so they do not have to
// reimplement paging at every call site (a real anti-pattern that
// reproduced at the SDK layer of a downstream consumer, where the
// caller invoked the wrong API and missed paginated tuples).
func TestReadTuples_WalksContinuationToken(t *testing.T) {
	page := 0
	fga := &mockFGA{
		readFn: func(_ context.Context, req *openfga.ReadRequest) (*openfga.ReadResponse, error) {
			page++
			switch page {
			case 1:
				if req.ContinuationToken != "" {
					t.Errorf("first page must send empty continuation token, got %q", req.ContinuationToken)
				}
				return &openfga.ReadResponse{
					Tuples: []*openfga.Tuple{
						{Key: &openfga.TupleKey{Object: "file:a", Relation: "viewer", User: "user:1"}},
						{Key: &openfga.TupleKey{Object: "file:a", Relation: "viewer", User: "user:2"}},
					},
					ContinuationToken: "page-2-token",
				}, nil
			case 2:
				if req.ContinuationToken != "page-2-token" {
					t.Errorf("second page must echo continuation token, got %q", req.ContinuationToken)
				}
				return &openfga.ReadResponse{
					Tuples: []*openfga.Tuple{
						{Key: &openfga.TupleKey{Object: "file:a", Relation: "viewer", User: "user:3"}},
					},
					ContinuationToken: "",
				}, nil
			default:
				t.Errorf("unexpected page %d (continuation token loop)", page)
				return &openfga.ReadResponse{}, nil
			}
		},
	}
	c := newTestClient(fga)

	got, err := c.ReadTuples(context.Background(), ReadTupleFilter{Object: "file:a"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if page != 2 {
		t.Errorf("expected 2 Read calls (one per page), got %d", page)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 concatenated tuples, got %d", len(got))
	}
	wantUsers := []string{"user:1", "user:2", "user:3"}
	for i, want := range wantUsers {
		if got[i].User != want {
			t.Errorf("tuple[%d].User = %q, want %q", i, got[i].User, want)
		}
	}
}
