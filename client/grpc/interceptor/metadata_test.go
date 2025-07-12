package interceptor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func TestMetadataPropagatorInterceptor_OutgoingAlreadySet(t *testing.T) {
	// Outgoing metadata is already set
	outgoingMD := metadata.Pairs("foo", "bar")
	ctx := metadata.NewOutgoingContext(context.Background(), outgoingMD)

	called := false
	invoker := func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
		called = true
		// Outgoing metadata should be preserved
		md, ok := metadata.FromOutgoingContext(ctx)
		assert.True(t, ok)
		assert.Equal(t, []string{"bar"}, md["foo"])
		return nil
	}

	err := UnaryMetadataPropagatorInterceptor(ctx, "/test.Service/Method", "req", "reply", nil, invoker)
	assert.NoError(t, err)
	assert.True(t, called, "invoker should be called")
}

func TestMetadataPropagatorInterceptor_IncomingToOutgoing(t *testing.T) {
	// Outgoing metadata is not set, but incoming is present
	incomingMD := metadata.Pairs("token", "abc123")
	ctx := metadata.NewIncomingContext(context.Background(), incomingMD)

	called := false
	invoker := func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
		called = true
		// Outgoing metadata should be set from incoming
		md, ok := metadata.FromOutgoingContext(ctx)
		assert.True(t, ok)
		assert.Equal(t, []string{"abc123"}, md["token"])
		return nil
	}

	err := UnaryMetadataPropagatorInterceptor(ctx, "/test.Service/Method", "req", "reply", nil, invoker)
	assert.NoError(t, err)
	assert.True(t, called, "invoker should be called")
}

func TestMetadataPropagatorInterceptor_NoMetadata(t *testing.T) {
	// No outgoing or incoming metadata
	ctx := context.Background()

	called := false
	invoker := func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
		called = true
		// Outgoing metadata should not be set
		_, ok := metadata.FromOutgoingContext(ctx)
		assert.False(t, ok)
		return nil
	}

	err := UnaryMetadataPropagatorInterceptor(ctx, "/test.Service/Method", "req", "reply", nil, invoker)
	assert.NoError(t, err)
	assert.True(t, called, "invoker should be called")
}

func TestMetadataPropagatorInterceptor_BothIncomingAndOutgoing(t *testing.T) {
	// Both incoming and outgoing metadata are set; outgoing should take precedence
	incomingMD := metadata.Pairs("token", "abc123")
	outgoingMD := metadata.Pairs("foo", "bar")
	ctx := metadata.NewIncomingContext(context.Background(), incomingMD)
	ctx = metadata.NewOutgoingContext(ctx, outgoingMD)

	called := false
	invoker := func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
		called = true
		md, ok := metadata.FromOutgoingContext(ctx)
		assert.True(t, ok)
		assert.Equal(t, []string{"bar"}, md["foo"])
		assert.Nil(t, md["token"])
		return nil
	}

	err := UnaryMetadataPropagatorInterceptor(ctx, "/test.Service/Method", "req", "reply", nil, invoker)
	assert.NoError(t, err)
	assert.True(t, called, "invoker should be called")
}

func TestStreamMetadataPropagatorInterceptor_OutgoingAlreadySet(t *testing.T) {
	// Outgoing metadata is already set
	outgoingMD := metadata.Pairs("foo", "bar")
	ctx := metadata.NewOutgoingContext(context.Background(), outgoingMD)

	called := false
	streamer := func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		called = true
		// Outgoing metadata should be preserved
		md, ok := metadata.FromOutgoingContext(ctx)
		assert.True(t, ok)
		assert.Equal(t, []string{"bar"}, md["foo"])
		return nil, nil
	}

	_, err := StreamMetadataPropagatorInterceptor(ctx, &grpc.StreamDesc{}, nil, "/test.Service/Method", streamer)
	assert.NoError(t, err)
	assert.True(t, called, "streamer should be called")
}

func TestStreamMetadataPropagatorInterceptor_IncomingToOutgoing(t *testing.T) {
	// Outgoing metadata is not set, but incoming is present
	incomingMD := metadata.Pairs("token", "abc123")
	ctx := metadata.NewIncomingContext(context.Background(), incomingMD)

	called := false
	streamer := func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		called = true
		// Outgoing metadata should be set from incoming
		md, ok := metadata.FromOutgoingContext(ctx)
		assert.True(t, ok)
		assert.Equal(t, []string{"abc123"}, md["token"])
		return nil, nil
	}

	_, err := StreamMetadataPropagatorInterceptor(ctx, &grpc.StreamDesc{}, nil, "/test.Service/Method", streamer)
	assert.NoError(t, err)
	assert.True(t, called, "streamer should be called")
}

func TestStreamMetadataPropagatorInterceptor_NoMetadata(t *testing.T) {
	// No outgoing or incoming metadata
	ctx := context.Background()

	called := false
	streamer := func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		called = true
		// Outgoing metadata should not be set
		_, ok := metadata.FromOutgoingContext(ctx)
		assert.False(t, ok)
		return nil, nil
	}

	_, err := StreamMetadataPropagatorInterceptor(ctx, &grpc.StreamDesc{}, nil, "/test.Service/Method", streamer)
	assert.NoError(t, err)
	assert.True(t, called, "streamer should be called")
}

func TestStreamMetadataPropagatorInterceptor_BothIncomingAndOutgoing(t *testing.T) {
	// Both incoming and outgoing metadata are set; outgoing should take precedence
	incomingMD := metadata.Pairs("token", "abc123")
	outgoingMD := metadata.Pairs("foo", "bar")
	ctx := metadata.NewIncomingContext(context.Background(), incomingMD)
	ctx = metadata.NewOutgoingContext(ctx, outgoingMD)

	called := false
	streamer := func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		called = true
		md, ok := metadata.FromOutgoingContext(ctx)
		assert.True(t, ok)
		assert.Equal(t, []string{"bar"}, md["foo"])
		assert.Nil(t, md["token"])
		return nil, nil
	}

	_, err := StreamMetadataPropagatorInterceptor(ctx, &grpc.StreamDesc{}, nil, "/test.Service/Method", streamer)
	assert.NoError(t, err)
	assert.True(t, called, "streamer should be called")
}
