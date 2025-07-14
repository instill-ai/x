package interceptor

import (
	"context"
	"errors"
	"testing"

	"github.com/frankban/quicktest"
	"github.com/gojuno/minimock/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	errorsx "github.com/instill-ai/x/errors"

	mockserver "github.com/instill-ai/x/mock/server"
)

func TestUnaryAppendMetadataInterceptor(t *testing.T) {
	qt := quicktest.New(t)

	tests := []struct {
		name            string
		ctx             context.Context
		req             any
		handler         grpc.UnaryHandler
		expectedResp    any
		expectedErr     error
		expectedErrCode codes.Code
		description     string
	}{
		{
			name: "successful request with metadata",
			ctx: metadata.NewIncomingContext(context.Background(), metadata.New(map[string]string{
				"user-id": "123",
				"token":   "abc123",
			})),
			req: "test request",
			handler: func(ctx context.Context, req any) (any, error) {
				// Verify metadata is preserved
				md, ok := metadata.FromIncomingContext(ctx)
				qt.Check(ok, quicktest.IsTrue)
				qt.Check(md.Get("user-id")[0], quicktest.Equals, "123")
				qt.Check(md.Get("token")[0], quicktest.Equals, "abc123")
				return "test response", nil
			},
			expectedResp: "test response",
			expectedErr:  nil,
			description:  "should preserve metadata and return successful response",
		},
		{
			name: "successful request without metadata",
			ctx:  context.Background(),
			req:  "test request",
			handler: func(ctx context.Context, req any) (any, error) {
				// Should still work without metadata
				return "test response", nil
			},
			expectedResp: "test response",
			expectedErr:  nil,
			description:  "should handle request without metadata",
		},
		{
			name: "handler returns error",
			ctx: metadata.NewIncomingContext(context.Background(), metadata.New(map[string]string{
				"user-id": "123",
			})),
			req: "test request",
			handler: func(ctx context.Context, req any) (any, error) {
				return nil, errors.New("handler error")
			},
			expectedResp:    nil,
			expectedErrCode: codes.Unknown,
			description:     "should convert handler error to gRPC error",
		},
		{
			name: "handler returns gRPC status error",
			ctx: metadata.NewIncomingContext(context.Background(), metadata.New(map[string]string{
				"user-id": "123",
			})),
			req: "test request",
			handler: func(ctx context.Context, req any) (any, error) {
				return nil, status.Error(codes.InvalidArgument, "invalid argument")
			},
			expectedResp:    nil,
			expectedErrCode: codes.InvalidArgument,
			description:     "should preserve gRPC status error",
		},
		{
			name: "handler returns domain error",
			ctx: metadata.NewIncomingContext(context.Background(), metadata.New(map[string]string{
				"user-id": "123",
			})),
			req: "test request",
			handler: func(ctx context.Context, req any) (any, error) {
				return nil, errorsx.ErrNotFound
			},
			expectedResp:    nil,
			expectedErrCode: codes.NotFound,
			description:     "should convert domain error to appropriate gRPC code",
		},
		{
			name: "empty metadata context",
			ctx:  metadata.NewIncomingContext(context.Background(), metadata.New(map[string]string{})),
			req:  "test request",
			handler: func(ctx context.Context, req any) (any, error) {
				// Should work with empty metadata
				return "test response", nil
			},
			expectedResp: "test response",
			expectedErr:  nil,
			description:  "should handle empty metadata",
		},
	}

	for _, tt := range tests {
		qt.Run(tt.name, func(c *quicktest.C) {
			// Create interceptor
			interceptor := UnaryAppendMetadataInterceptor

			// Create mock server info
			info := &grpc.UnaryServerInfo{
				FullMethod: "test.Service/Method",
			}

			// Execute interceptor
			resp, err := interceptor(tt.ctx, tt.req, info, tt.handler)

			// Verify response
			c.Check(resp, quicktest.Equals, tt.expectedResp)

			// Verify error
			if tt.expectedErr != nil {
				c.Check(err, quicktest.Equals, tt.expectedErr)
			} else if tt.expectedErrCode != codes.OK {
				c.Check(err, quicktest.Not(quicktest.IsNil))
				st, ok := status.FromError(err)
				c.Check(ok, quicktest.IsTrue)
				c.Check(st.Code(), quicktest.Equals, tt.expectedErrCode)
			} else {
				c.Check(err, quicktest.IsNil)
			}
		})
	}
}

func TestUnaryAppendMetadataInterceptor_NoMetadata(t *testing.T) {
	qt := quicktest.New(t)

	// Test case where metadata cannot be extracted
	ctx := context.Background() // No metadata context

	interceptor := UnaryAppendMetadataInterceptor
	info := &grpc.UnaryServerInfo{
		FullMethod: "test.Service/Method",
	}

	handler := func(ctx context.Context, req any) (any, error) {
		// Verify that empty metadata is created when none exists
		md, ok := metadata.FromIncomingContext(ctx)
		qt.Check(ok, quicktest.IsTrue)
		qt.Check(len(md), quicktest.Equals, 0)
		return "response", nil
	}

	// This should succeed and create empty metadata
	resp, err := interceptor(ctx, "request", info, handler)

	qt.Check(resp, quicktest.Equals, "response")
	qt.Check(err, quicktest.IsNil)
}

func TestStreamAppendMetadataInterceptor(t *testing.T) {
	qt := quicktest.New(t)

	tests := []struct {
		name            string
		ctx             context.Context
		handler         grpc.StreamHandler
		expectedErr     error
		expectedErrCode codes.Code
		description     string
	}{
		{
			name: "successful stream with metadata",
			ctx: metadata.NewIncomingContext(context.Background(), metadata.New(map[string]string{
				"user-id": "123",
				"token":   "abc123",
			})),
			handler: func(srv any, stream grpc.ServerStream) error {
				// Verify metadata is preserved in stream context
				md, ok := metadata.FromIncomingContext(stream.Context())
				qt.Check(ok, quicktest.IsTrue)
				qt.Check(md.Get("user-id")[0], quicktest.Equals, "123")
				qt.Check(md.Get("token")[0], quicktest.Equals, "abc123")
				return nil
			},
			expectedErr: nil,
			description: "should preserve metadata in stream context",
		},
		{
			name: "stream handler returns error",
			ctx: metadata.NewIncomingContext(context.Background(), metadata.New(map[string]string{
				"user-id": "123",
			})),
			handler: func(srv any, stream grpc.ServerStream) error {
				return errors.New("stream handler error")
			},
			expectedErrCode: codes.Unknown,
			description:     "should convert stream handler error to gRPC error",
		},
		{
			name: "stream handler returns gRPC status error",
			ctx: metadata.NewIncomingContext(context.Background(), metadata.New(map[string]string{
				"user-id": "123",
			})),
			handler: func(srv any, stream grpc.ServerStream) error {
				return status.Error(codes.PermissionDenied, "permission denied")
			},
			expectedErrCode: codes.PermissionDenied,
			description:     "should preserve gRPC status error from stream handler",
		},
		{
			name: "stream handler returns domain error",
			ctx: metadata.NewIncomingContext(context.Background(), metadata.New(map[string]string{
				"user-id": "123",
			})),
			handler: func(srv any, stream grpc.ServerStream) error {
				return errorsx.ErrUnauthorized
			},
			expectedErrCode: codes.PermissionDenied,
			description:     "should convert domain error to appropriate gRPC code",
		},
		{
			name: "empty metadata context",
			ctx:  metadata.NewIncomingContext(context.Background(), metadata.New(map[string]string{})),
			handler: func(srv any, stream grpc.ServerStream) error {
				// Should work with empty metadata
				return nil
			},
			expectedErr: nil,
			description: "should handle empty metadata in stream",
		},
	}

	for _, tt := range tests {
		qt.Run(tt.name, func(c *quicktest.C) {
			// Create interceptor
			interceptor := StreamAppendMetadataInterceptor

			// Create mock stream
			mc := minimock.NewController(t)
			stream := mockserver.NewServerStreamMock(mc)

			// Add this line to set up the Context expectation
			stream.ContextMock.Expect().Return(tt.ctx)

			// Create mock server info
			info := &grpc.StreamServerInfo{
				FullMethod: "test.Service/StreamMethod",
			}

			// Execute interceptor
			err := interceptor(nil, stream, info, tt.handler)

			// Verify error
			if tt.expectedErr != nil {
				c.Check(err, quicktest.Equals, tt.expectedErr)
			} else if tt.expectedErrCode != codes.OK {
				c.Check(err, quicktest.Not(quicktest.IsNil))
				st, ok := status.FromError(err)
				c.Check(ok, quicktest.IsTrue)
				c.Check(st.Code(), quicktest.Equals, tt.expectedErrCode)
			} else {
				c.Check(err, quicktest.IsNil)
			}
		})
	}
}

func TestStreamAppendMetadataInterceptor_NoMetadata(t *testing.T) {
	qt := quicktest.New(t)

	// Test case where metadata cannot be extracted from stream
	interceptor := StreamAppendMetadataInterceptor
	mc := minimock.NewController(t)
	stream := mockserver.NewServerStreamMock(mc)

	// Set up Context expectation for TWO calls
	ctx := context.Background()
	stream.ContextMock.Expect().Return(ctx)
	stream.ContextMock.Expect().Return(ctx)

	info := &grpc.StreamServerInfo{
		FullMethod: "test.Service/StreamMethod",
	}

	handler := func(srv any, stream grpc.ServerStream) error {
		// Verify that empty metadata is created when none exists
		md, ok := metadata.FromIncomingContext(stream.Context())
		qt.Check(ok, quicktest.IsTrue)
		qt.Check(len(md), quicktest.Equals, 0)
		return nil
	}

	// This should succeed and create empty metadata
	err := interceptor(nil, stream, info, handler)

	qt.Check(err, quicktest.IsNil)
}

func TestStreamAppendMetadataInterceptor_ContextPreservation(t *testing.T) {
	qt := quicktest.New(t)

	// Test that the context is properly preserved in the wrapped stream
	originalCtx := metadata.NewIncomingContext(context.Background(), metadata.New(map[string]string{
		"user-id": "123",
		"token":   "abc123",
	}))

	interceptor := StreamAppendMetadataInterceptor
	mc := minimock.NewController(t)
	stream := mockserver.NewServerStreamMock(mc)

	// Set up Context expectation for TWO calls
	stream.ContextMock.Expect().Return(originalCtx)

	info := &grpc.StreamServerInfo{
		FullMethod: "test.Service/StreamMethod",
	}

	var capturedCtx context.Context
	handler := func(srv any, stream grpc.ServerStream) error {
		capturedCtx = stream.Context()

		// Verify metadata is preserved
		md, ok := metadata.FromIncomingContext(capturedCtx)
		qt.Check(ok, quicktest.IsTrue)
		qt.Check(md.Get("user-id")[0], quicktest.Equals, "123")
		qt.Check(md.Get("token")[0], quicktest.Equals, "abc123")

		return nil
	}

	err := interceptor(nil, stream, info, handler)
	qt.Check(err, quicktest.IsNil)

	// Verify context was captured and is different from original
	qt.Check(capturedCtx, quicktest.Not(quicktest.IsNil))
	qt.Check(capturedCtx, quicktest.Not(quicktest.Equals), originalCtx)
}

func TestStreamAppendMetadataInterceptor_StreamMethods(t *testing.T) {
	qt := quicktest.New(t)

	// Test that the wrapped stream properly implements all required methods
	interceptor := StreamAppendMetadataInterceptor
	mc := minimock.NewController(t)
	originalStream := mockserver.NewServerStreamMock(mc)

	// Set up Context expectation for TWO calls
	ctx := context.Background()
	originalStream.ContextMock.Expect().Return(ctx)
	originalStream.ContextMock.Expect().Return(ctx)

	// Set up expectations for other stream methods that will be called
	originalStream.RecvMsgMock.Expect("test").Return(nil)
	originalStream.SendMsgMock.Expect("test").Return(nil)
	originalStream.SendHeaderMock.Expect(metadata.New(map[string]string{})).Return(nil)
	originalStream.SetHeaderMock.Expect(metadata.New(map[string]string{})).Return(nil)
	originalStream.SetTrailerMock.Expect(metadata.New(map[string]string{}))

	info := &grpc.StreamServerInfo{
		FullMethod: "test.Service/StreamMethod",
	}

	var wrappedStream grpc.ServerStream
	handler := func(srv any, stream grpc.ServerStream) error {
		wrappedStream = stream

		// Test that stream methods work
		qt.Check(stream.Context(), quicktest.Not(quicktest.IsNil))
		qt.Check(stream.RecvMsg("test"), quicktest.IsNil)
		qt.Check(stream.SendMsg("test"), quicktest.IsNil)
		qt.Check(stream.SendHeader(metadata.New(map[string]string{})), quicktest.IsNil)
		qt.Check(stream.SetHeader(metadata.New(map[string]string{})), quicktest.IsNil)
		stream.SetTrailer(metadata.New(map[string]string{})) // SetTrailer has no return value

		return nil
	}

	err := interceptor(nil, originalStream, info, handler)
	qt.Check(err, quicktest.IsNil)
	qt.Check(wrappedStream, quicktest.Not(quicktest.IsNil))
}

func TestUnaryAppendMetadataInterceptor_ErrorConversion(t *testing.T) {
	qt := quicktest.New(t)

	// Test various error types and their conversion
	testCases := []struct {
		name         string
		handlerError error
		expectedCode codes.Code
		description  string
	}{
		{
			name:         "nil error",
			handlerError: nil,
			expectedCode: codes.OK,
			description:  "should handle nil error",
		},
		{
			name:         "standard error",
			handlerError: errors.New("standard error"),
			expectedCode: codes.Unknown,
			description:  "should convert standard error to Unknown",
		},
		{
			name:         "gRPC status error",
			handlerError: status.Error(codes.NotFound, "not found"),
			expectedCode: codes.NotFound,
			description:  "should preserve gRPC status error",
		},
		{
			name:         "domain error - NotFound",
			handlerError: errorsx.ErrNotFound,
			expectedCode: codes.NotFound,
			description:  "should convert domain NotFound error",
		},
		{
			name:         "domain error - InvalidArgument",
			handlerError: errorsx.ErrInvalidArgument,
			expectedCode: codes.InvalidArgument,
			description:  "should convert domain InvalidArgument error",
		},
		{
			name:         "domain error - AlreadyExists",
			handlerError: errorsx.ErrAlreadyExists,
			expectedCode: codes.AlreadyExists,
			description:  "should convert domain AlreadyExists error",
		},
		{
			name:         "domain error - Unauthorized",
			handlerError: errorsx.ErrUnauthorized,
			expectedCode: codes.PermissionDenied,
			description:  "should convert domain Unauthorized error",
		},
	}

	for _, tc := range testCases {
		qt.Run(tc.name, func(c *quicktest.C) {
			ctx := metadata.NewIncomingContext(context.Background(), metadata.New(map[string]string{
				"test-key": "test-value",
			}))

			interceptor := UnaryAppendMetadataInterceptor
			info := &grpc.UnaryServerInfo{
				FullMethod: "test.Service/Method",
			}

			handler := func(ctx context.Context, req any) (any, error) {
				return "response", tc.handlerError
			}

			resp, err := interceptor(ctx, "request", info, handler)

			if tc.handlerError == nil {
				c.Check(err, quicktest.IsNil)
				c.Check(resp, quicktest.Equals, "response")
			} else {
				c.Check(err, quicktest.Not(quicktest.IsNil))
				st, ok := status.FromError(err)
				c.Check(ok, quicktest.IsTrue)
				c.Check(st.Code(), quicktest.Equals, tc.expectedCode)
			}
		})
	}
}

func TestStreamAppendMetadataInterceptor_ErrorConversion(t *testing.T) {
	qt := quicktest.New(t)

	// Test various error types and their conversion for streams
	testCases := []struct {
		name         string
		handlerError error
		expectedCode codes.Code
		description  string
	}{
		{
			name:         "nil error",
			handlerError: nil,
			expectedCode: codes.OK,
			description:  "should handle nil error in stream",
		},
		{
			name:         "standard error",
			handlerError: errors.New("standard error"),
			expectedCode: codes.Unknown,
			description:  "should convert standard error to Unknown in stream",
		},
		{
			name:         "gRPC status error",
			handlerError: status.Error(codes.PermissionDenied, "permission denied"),
			expectedCode: codes.PermissionDenied,
			description:  "should preserve gRPC status error in stream",
		},
		{
			name:         "domain error - NotFound",
			handlerError: errorsx.ErrNotFound,
			expectedCode: codes.NotFound,
			description:  "should convert domain NotFound error in stream",
		},
		{
			name:         "domain error - InvalidArgument",
			handlerError: errorsx.ErrInvalidArgument,
			expectedCode: codes.InvalidArgument,
			description:  "should convert domain InvalidArgument error in stream",
		},
	}

	for _, tc := range testCases {
		qt.Run(tc.name, func(c *quicktest.C) {
			interceptor := StreamAppendMetadataInterceptor
			mc := minimock.NewController(t)
			stream := mockserver.NewServerStreamMock(mc)

			// Set up Context expectation for TWO calls
			ctx := metadata.NewIncomingContext(context.Background(), metadata.New(map[string]string{
				"test-key": "test-value",
			}))
			stream.ContextMock.Expect().Return(ctx)

			info := &grpc.StreamServerInfo{
				FullMethod: "test.Service/StreamMethod",
			}

			handler := func(srv any, stream grpc.ServerStream) error {
				return tc.handlerError
			}

			err := interceptor(nil, stream, info, handler)

			if tc.handlerError == nil {
				c.Check(err, quicktest.IsNil)
			} else {
				c.Check(err, quicktest.Not(quicktest.IsNil))
				st, ok := status.FromError(err)
				c.Check(ok, quicktest.IsTrue)
				c.Check(st.Code(), quicktest.Equals, tc.expectedCode)
			}
		})
	}
}

func TestUnaryAppendMetadataInterceptor_MetadataPreservation(t *testing.T) {
	qt := quicktest.New(t)

	// Test that metadata is properly preserved and accessible in handler
	originalMD := metadata.New(map[string]string{
		"user-id":    "123",
		"token":      "abc123",
		"request-id": "req-456",
		"trace-id":   "trace-789",
	})

	ctx := metadata.NewIncomingContext(context.Background(), originalMD)

	interceptor := UnaryAppendMetadataInterceptor
	info := &grpc.UnaryServerInfo{
		FullMethod: "test.Service/Method",
	}

	var capturedMD metadata.MD
	handler := func(ctx context.Context, req any) (any, error) {
		capturedMD, _ = metadata.FromIncomingContext(ctx)
		return "response", nil
	}

	resp, err := interceptor(ctx, "request", info, handler)

	qt.Check(err, quicktest.IsNil)
	qt.Check(resp, quicktest.Equals, "response")
	qt.Check(capturedMD, quicktest.Not(quicktest.IsNil))

	// Verify all metadata keys are preserved
	for key, values := range originalMD {
		qt.Check(capturedMD.Get(key), quicktest.DeepEquals, values)
	}
}

func TestStreamAppendMetadataInterceptor_MetadataPreservation(t *testing.T) {
	qt := quicktest.New(t)

	// Test that metadata is properly preserved and accessible in stream handler
	originalMD := metadata.New(map[string]string{
		"user-id":    "123",
		"token":      "abc123",
		"request-id": "req-456",
		"trace-id":   "trace-789",
	})

	interceptor := StreamAppendMetadataInterceptor
	mc := minimock.NewController(t)
	stream := mockserver.NewServerStreamMock(mc)

	// Set up Context expectation for TWO calls
	ctx := metadata.NewIncomingContext(context.Background(), originalMD)
	stream.ContextMock.Expect().Return(ctx)
	stream.ContextMock.Expect().Return(ctx)

	info := &grpc.StreamServerInfo{
		FullMethod: "test.Service/StreamMethod",
	}

	handler := func(srv any, stream grpc.ServerStream) error {
		// Verify metadata is preserved
		md, ok := metadata.FromIncomingContext(stream.Context())
		qt.Check(ok, quicktest.IsTrue)
		qt.Check(md.Get("user-id")[0], quicktest.Equals, "123")
		qt.Check(md.Get("token")[0], quicktest.Equals, "abc123")
		return nil
	}

	err := interceptor(nil, stream, info, handler)
	qt.Check(err, quicktest.IsNil)
}
