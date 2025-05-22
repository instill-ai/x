package resource

import (
	"context"
	"strings"

	"google.golang.org/grpc/metadata"

	"github.com/gofrs/uuid"
	"github.com/instill-ai/x/constant"
)

// GetRequestSingleHeader gets a request header, assuming the value is a
// single-value string HTTP header.
func GetRequestSingleHeader(ctx context.Context, key string) string {
	metaHeader := metadata.ValueFromIncomingContext(ctx, strings.ToLower(key))
	if len(metaHeader) != 1 {
		return ""
	}
	return metaHeader[0]
}

// GetRequesterUIDAndUserUID extracts the requester and user UIDs from the
// request header.
func GetRequesterUIDAndUserUID(ctx context.Context) (uuid.UUID, uuid.UUID) {
	requesterUID := GetRequestSingleHeader(ctx, constant.HeaderRequesterUIDKey)
	userUID := GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey)
	if strings.TrimSpace(requesterUID) == "" {
		requesterUID = userUID
	}
	return uuid.FromStringOrNil(requesterUID), uuid.FromStringOrNil(userUID)
}
