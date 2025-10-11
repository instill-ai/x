package interceptor

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"

	errorsx "github.com/instill-ai/x/errors"
)

// UnaryAppendMetadataInterceptor - append metadata for unary
func UnaryAppendMetadataInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
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

// NewUnaryInjectMetadataInterceptor creates a unary server interceptor that injects
// the specified metadata key-value pairs into the incoming context. This metadata
// will then be automatically propagated to all outgoing gRPC calls by the client-side
// UnaryMetadataPropagatorInterceptor.
//
// This is useful for services that need to identify themselves when making requests
// to other services (e.g., agent-backend adding "Instill-Backend: agent-backend").
func NewUnaryInjectMetadataInterceptor(headerKey, value string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		// Get existing metadata or create new
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			md = metadata.New(map[string]string{})
		}
		// Clone and append new metadata
		md = md.Copy()
		md.Append(headerKey, value)
		// Create new context with updated metadata
		ctx = metadata.NewIncomingContext(ctx, md)
		return handler(ctx, req)
	}
}

// NewStreamInjectMetadataInterceptor creates a stream server interceptor that injects
// the specified metadata key-value pairs into the incoming context.
func NewStreamInjectMetadataInterceptor(headerKey, value string) grpc.StreamServerInterceptor {
	return func(srv any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// Get existing metadata or create new
		md, ok := metadata.FromIncomingContext(stream.Context())
		if !ok {
			md = metadata.New(map[string]string{})
		}
		// Clone and append new metadata
		md = md.Copy()
		md.Append(headerKey, value)
		// Create new context with updated metadata
		ctx := metadata.NewIncomingContext(stream.Context(), md)
		wrapped := grpc_middleware.WrapServerStream(stream)
		wrapped.WrappedContext = ctx
		return handler(srv, wrapped)
	}
}
