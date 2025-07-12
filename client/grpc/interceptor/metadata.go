package interceptor

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// MetadataPropagatorInterceptor adds the metadata in the incoming context to
// the outgoing context of every outbound gRPC call.
func MetadataPropagatorInterceptor(
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

	newCtx := metadata.NewOutgoingContext(ctx, md)
	return invoker(newCtx, method, req, reply, cc, opts...)
}
