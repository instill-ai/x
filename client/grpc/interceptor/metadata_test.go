package interceptor

import (
	"context"
	"testing"

	"github.com/frankban/quicktest"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func TestMetadataPropagatorInterceptor_OutgoingAlreadySet(t *testing.T) {
	qt := quicktest.New(t)

	// Outgoing metadata is already set
	outgoingMD := metadata.Pairs("foo", "bar")
	ctx := metadata.NewOutgoingContext(context.Background(), outgoingMD)

	called := false
	invoker := func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
		called = true
		// Outgoing metadata should be preserved
		md, ok := metadata.FromOutgoingContext(ctx)
		qt.Assert(ok, quicktest.IsTrue)
		qt.Check(md["foo"], quicktest.DeepEquals, []string{"bar"})
		return nil
	}

	err := UnaryMetadataPropagatorInterceptor(ctx, "/test.Service/Method", "req", "reply", nil, invoker)
	qt.Assert(err, quicktest.IsNil)
	qt.Assert(called, quicktest.IsTrue)
}

func TestMetadataPropagatorInterceptor_IncomingToOutgoing(t *testing.T) {
	qt := quicktest.New(t)

	// Outgoing metadata is not set, but incoming is present
	incomingMD := metadata.Pairs("authorization", "Bearer abc123")
	ctx := metadata.NewIncomingContext(context.Background(), incomingMD)

	called := false
	invoker := func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
		called = true
		// Outgoing metadata should be set from incoming
		md, ok := metadata.FromOutgoingContext(ctx)
		qt.Assert(ok, quicktest.IsTrue)
		qt.Check(md["authorization"], quicktest.DeepEquals, []string{"Bearer abc123"})
		return nil
	}

	err := UnaryMetadataPropagatorInterceptor(ctx, "/test.Service/Method", "req", "reply", nil, invoker)
	qt.Assert(err, quicktest.IsNil)
	qt.Assert(called, quicktest.IsTrue)
}

func TestMetadataPropagatorInterceptor_NoMetadata(t *testing.T) {
	qt := quicktest.New(t)

	// No outgoing or incoming metadata
	ctx := context.Background()

	called := false
	invoker := func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
		called = true
		// Outgoing metadata should not be set
		_, ok := metadata.FromOutgoingContext(ctx)
		qt.Assert(ok, quicktest.IsFalse)
		return nil
	}

	err := UnaryMetadataPropagatorInterceptor(ctx, "/test.Service/Method", "req", "reply", nil, invoker)
	qt.Assert(err, quicktest.IsNil)
	qt.Assert(called, quicktest.IsTrue)
}

func TestMetadataPropagatorInterceptor_BothIncomingAndOutgoing(t *testing.T) {
	qt := quicktest.New(t)

	// Both incoming and outgoing metadata are set; outgoing should take precedence
	incomingMD := metadata.Pairs("authorization", "Bearer abc123")
	outgoingMD := metadata.Pairs("foo", "bar")
	ctx := metadata.NewIncomingContext(context.Background(), incomingMD)
	ctx = metadata.NewOutgoingContext(ctx, outgoingMD)

	called := false
	invoker := func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
		called = true
		md, ok := metadata.FromOutgoingContext(ctx)
		qt.Assert(ok, quicktest.IsTrue)
		qt.Check(md["foo"], quicktest.DeepEquals, []string{"bar"})
		qt.Assert(md["authorization"], quicktest.IsNil)
		return nil
	}

	err := UnaryMetadataPropagatorInterceptor(ctx, "/test.Service/Method", "req", "reply", nil, invoker)
	qt.Assert(err, quicktest.IsNil)
	qt.Assert(called, quicktest.IsTrue)
}

func TestStreamMetadataPropagatorInterceptor_OutgoingAlreadySet(t *testing.T) {
	qt := quicktest.New(t)

	// Outgoing metadata is already set
	outgoingMD := metadata.Pairs("foo", "bar")
	ctx := metadata.NewOutgoingContext(context.Background(), outgoingMD)

	called := false
	streamer := func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		called = true
		// Outgoing metadata should be preserved
		md, ok := metadata.FromOutgoingContext(ctx)
		qt.Assert(ok, quicktest.IsTrue)
		qt.Check(md["foo"], quicktest.DeepEquals, []string{"bar"})
		return nil, nil
	}

	_, err := StreamMetadataPropagatorInterceptor(ctx, &grpc.StreamDesc{}, nil, "/test.Service/Method", streamer)
	qt.Assert(err, quicktest.IsNil)
	qt.Assert(called, quicktest.IsTrue)
}

func TestStreamMetadataPropagatorInterceptor_IncomingToOutgoing(t *testing.T) {
	qt := quicktest.New(t)

	// Outgoing metadata is not set, but incoming is present
	incomingMD := metadata.Pairs("authorization", "Bearer abc123")
	ctx := metadata.NewIncomingContext(context.Background(), incomingMD)

	called := false
	streamer := func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		called = true
		// Outgoing metadata should be set from incoming
		md, ok := metadata.FromOutgoingContext(ctx)
		qt.Assert(ok, quicktest.IsTrue)
		qt.Check(md["authorization"], quicktest.DeepEquals, []string{"Bearer abc123"})
		return nil, nil
	}

	_, err := StreamMetadataPropagatorInterceptor(ctx, &grpc.StreamDesc{}, nil, "/test.Service/Method", streamer)
	qt.Assert(err, quicktest.IsNil)
	qt.Assert(called, quicktest.IsTrue)
}

func TestStreamMetadataPropagatorInterceptor_NoMetadata(t *testing.T) {
	qt := quicktest.New(t)

	// No outgoing or incoming metadata
	ctx := context.Background()

	called := false
	streamer := func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		called = true
		// Outgoing metadata should not be set
		_, ok := metadata.FromOutgoingContext(ctx)
		qt.Assert(ok, quicktest.IsFalse)
		return nil, nil
	}

	_, err := StreamMetadataPropagatorInterceptor(ctx, &grpc.StreamDesc{}, nil, "/test.Service/Method", streamer)
	qt.Assert(err, quicktest.IsNil)
	qt.Assert(called, quicktest.IsTrue)
}

func TestStreamMetadataPropagatorInterceptor_BothIncomingAndOutgoing(t *testing.T) {
	qt := quicktest.New(t)

	// Both incoming and outgoing metadata are set; outgoing should take precedence
	incomingMD := metadata.Pairs("authorization", "Bearer abc123")
	outgoingMD := metadata.Pairs("foo", "bar")
	ctx := metadata.NewIncomingContext(context.Background(), incomingMD)
	ctx = metadata.NewOutgoingContext(ctx, outgoingMD)

	called := false
	streamer := func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		called = true
		md, ok := metadata.FromOutgoingContext(ctx)
		qt.Assert(ok, quicktest.IsTrue)
		qt.Check(md["foo"], quicktest.DeepEquals, []string{"bar"})
		qt.Assert(md["authorization"], quicktest.IsNil)
		return nil, nil
	}

	_, err := StreamMetadataPropagatorInterceptor(ctx, &grpc.StreamDesc{}, nil, "/test.Service/Method", streamer)
	qt.Assert(err, quicktest.IsNil)
	qt.Assert(called, quicktest.IsTrue)
}
