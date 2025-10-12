package interceptor

import (
	"context"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// NewUnaryMetadataServiceIdentificationInterceptor creates a unary client interceptor that adds
// service identification metadata to all outgoing gRPC calls.
func NewUnaryMetadataServiceIdentificationInterceptor(headerKey, serviceName string) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply any,
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		newCtx := metadata.AppendToOutgoingContext(ctx, headerKey, serviceName)
		return invoker(newCtx, method, req, reply, cc, opts...)
	}
}

// NewStreamMetadataServiceIdentificationInterceptor creates a stream client interceptor that adds
// service identification metadata to all outgoing gRPC calls.
func NewStreamMetadataServiceIdentificationInterceptor(headerKey, serviceName string) grpc.StreamClientInterceptor {
	return func(
		ctx context.Context,
		desc *grpc.StreamDesc,
		cc *grpc.ClientConn,
		method string,
		streamer grpc.Streamer,
		opts ...grpc.CallOption,
	) (grpc.ClientStream, error) {
		newCtx := metadata.AppendToOutgoingContext(ctx, headerKey, serviceName)
		return streamer(newCtx, desc, cc, method, opts...)
	}
}

// UnaryMetadataPropagatorInterceptor propagates filtered metadata from incoming to outgoing context.
func UnaryMetadataPropagatorInterceptor(
	ctx context.Context,
	method string,
	req, reply any,
	cc *grpc.ClientConn,
	invoker grpc.UnaryInvoker,
	opts ...grpc.CallOption,
) error {
	if _, outgoingContextAlreadySet := metadata.FromOutgoingContext(ctx); outgoingContextAlreadySet {
		return invoker(ctx, method, req, reply, cc, opts...)
	}

	md, hasIncomingContext := metadata.FromIncomingContext(ctx)
	if !hasIncomingContext {
		return invoker(ctx, method, req, reply, cc, opts...)
	}

	filteredMd := filterMetadata(md)
	newCtx := metadata.NewOutgoingContext(ctx, filteredMd)
	return invoker(newCtx, method, req, reply, cc, opts...)
}

// StreamMetadataPropagatorInterceptor propagates filtered metadata from incoming to outgoing context.
func StreamMetadataPropagatorInterceptor(
	ctx context.Context,
	desc *grpc.StreamDesc,
	cc *grpc.ClientConn,
	method string,
	streamer grpc.Streamer,
	opts ...grpc.CallOption,
) (grpc.ClientStream, error) {
	if _, outgoingContextAlreadySet := metadata.FromOutgoingContext(ctx); outgoingContextAlreadySet {
		return streamer(ctx, desc, cc, method, opts...)
	}

	md, hasIncomingContext := metadata.FromIncomingContext(ctx)
	if !hasIncomingContext {
		return streamer(ctx, desc, cc, method, opts...)
	}

	filteredMd := filterMetadata(md)
	newCtx := metadata.NewOutgoingContext(ctx, filteredMd)
	return streamer(newCtx, desc, cc, method, opts...)
}

// filterMetadata removes HTTP/2 pseudo-headers and browser headers.
// Only propagates authentication and Instill-specific headers.
func filterMetadata(md metadata.MD) metadata.MD {
	filtered := metadata.MD{}
	for key, values := range md {
		lowerKey := strings.ToLower(key)

		// Only allow Instill-specific headers
		if strings.HasPrefix(lowerKey, "instill-") {
			filtered[key] = values
			continue
		}
		if lowerKey == "authorization" {
			filtered[key] = values
			continue
		}
		if strings.HasPrefix(lowerKey, "x-forwarded-") {
			filtered[key] = values
			continue
		}
	}
	return filtered
}
