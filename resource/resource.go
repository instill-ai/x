package resource

import (
	"context"
	"strings"

	"google.golang.org/grpc/metadata"

	"github.com/gofrs/uuid"
	"github.com/instill-ai/x/constant"
)

// GetRequestSingleHeader get a request header, the header has to be single-value HTTP header
func GetRequestSingleHeader(ctx context.Context, header string) string {
	metaHeader := metadata.ValueFromIncomingContext(ctx, strings.ToLower(header))
	if len(metaHeader) != 1 {
		return ""
	}
	return metaHeader[0]
}

func GetRequesterUIDAndUserUID(ctx context.Context) (uuid.UUID, uuid.UUID) {
	requesterUID := GetRequestSingleHeader(ctx, constant.HeaderRequesterUIDKey)
	userUID := GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey)
	if strings.TrimSpace(requesterUID) == "" {
		requesterUID = userUID
	}
	return uuid.FromStringOrNil(requesterUID), uuid.FromStringOrNil(userUID)
}
