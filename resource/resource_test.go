package resource_test

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	"google.golang.org/grpc/metadata"

	qt "github.com/frankban/quicktest"

	"github.com/instill-ai/x/constant"
	"github.com/instill-ai/x/resource"
)

func TestGetRequesterUIDAndUserUID(t *testing.T) {
	requesterUIDStr := uuid.Must(uuid.NewV4()).String()
	userUIDStr := uuid.Must(uuid.NewV4()).String()
	m := make(map[string]string)
	m[constant.HeaderRequesterUIDKey] = requesterUIDStr
	m[constant.HeaderUserUIDKey] = userUIDStr
	ctx := metadata.NewIncomingContext(context.Background(), metadata.New(m))

	c := qt.New(t)
	checkRequesterUID, checkUserUID := resource.GetRequesterUIDAndUserUID(ctx)
	requesterUID := uuid.FromStringOrNil(requesterUIDStr)
	userUID := uuid.FromStringOrNil(userUIDStr)

	c.Check(checkRequesterUID, qt.Equals, requesterUID)
	c.Check(checkUserUID, qt.Equals, userUID)
}
