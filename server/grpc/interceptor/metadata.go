package interceptor

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"

	errorsx "github.com/instill-ai/x/errors"
)

// UnaryAppendMetadataAndErrorCodeInterceptor - append metadata for unary
func UnaryAppendMetadataAndErrorCodeInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		// If no metadata is present, create empty metadata to ensure consistent behavior
		md = metadata.New(map[string]string{})
	}

	newCtx := metadata.NewIncomingContext(ctx, md)
	h, err := handler(newCtx, req)

	return h, errorsx.ConvertToGRPCError(err)
}

// StreamAppendMetadataInterceptor - append metadata for stream
func StreamAppendMetadataInterceptor(srv any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	md, ok := metadata.FromIncomingContext(stream.Context())
	if !ok {
		// If no metadata is present, create empty metadata to ensure consistent behavior
		md = metadata.New(map[string]string{})
	}

	newCtx := metadata.NewIncomingContext(stream.Context(), md)
	wrapped := grpc_middleware.WrapServerStream(stream)
	wrapped.WrappedContext = newCtx

	err := handler(srv, wrapped)

	return errorsx.ConvertToGRPCError(err)
}
