package interceptor

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	errorsx "github.com/instill-ai/x/errors"
)

func TestUnaryAppendMetadataInterceptor(t *testing.T) {
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
				assert.True(t, ok, "metadata should be present in context")
				assert.Equal(t, "123", md.Get("user-id")[0])
				assert.Equal(t, "abc123", md.Get("token")[0])
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
		t.Run(tt.name, func(t *testing.T) {
			// Create interceptor
			interceptor := UnaryAppendMetadataInterceptor

			// Create mock server info
			info := &grpc.UnaryServerInfo{
				FullMethod: "test.Service/Method",
			}

			// Execute interceptor
			resp, err := interceptor(tt.ctx, tt.req, info, tt.handler)

			// Verify response
			assert.Equal(t, tt.expectedResp, resp, tt.description)

			// Verify error
			if tt.expectedErr != nil {
				assert.Equal(t, tt.expectedErr, err, tt.description)
			} else if tt.expectedErrCode != codes.OK {
				assert.Error(t, err, tt.description)
				st, ok := status.FromError(err)
				assert.True(t, ok, "should be a gRPC status error")
				assert.Equal(t, tt.expectedErrCode, st.Code(), tt.description)
			} else {
				assert.NoError(t, err, tt.description)
			}
		})
	}
}

func TestUnaryAppendMetadataInterceptor_NoMetadata(t *testing.T) {
	// Test case where metadata cannot be extracted
	ctx := context.Background() // No metadata context

	interceptor := UnaryAppendMetadataInterceptor
	info := &grpc.UnaryServerInfo{
		FullMethod: "test.Service/Method",
	}

	handler := func(ctx context.Context, req any) (any, error) {
		// Verify that empty metadata is created when none exists
		md, ok := metadata.FromIncomingContext(ctx)
		assert.True(t, ok, "metadata should be present even when none was provided")
		assert.Empty(t, md, "metadata should be empty when none was provided")
		return "response", nil
	}

	// This should succeed and create empty metadata
	resp, err := interceptor(ctx, "request", info, handler)

	assert.Equal(t, "response", resp, "should return successful response when no metadata is provided")
	assert.NoError(t, err, "should not return error when no metadata is provided")
}

func TestStreamAppendMetadataInterceptor(t *testing.T) {
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
				assert.True(t, ok, "metadata should be present in stream context")
				assert.Equal(t, "123", md.Get("user-id")[0])
				assert.Equal(t, "abc123", md.Get("token")[0])
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
		t.Run(tt.name, func(t *testing.T) {
			// Create interceptor
			interceptor := StreamAppendMetadataInterceptor

			// Create mock stream
			stream := &MockServerStream{ctx: tt.ctx}

			// Create mock server info
			info := &grpc.StreamServerInfo{
				FullMethod: "test.Service/StreamMethod",
			}

			// Execute interceptor
			err := interceptor(nil, stream, info, tt.handler)

			// Verify error
			if tt.expectedErr != nil {
				assert.Equal(t, tt.expectedErr, err, tt.description)
			} else if tt.expectedErrCode != codes.OK {
				assert.Error(t, err, tt.description)
				st, ok := status.FromError(err)
				assert.True(t, ok, "should be a gRPC status error")
				assert.Equal(t, tt.expectedErrCode, st.Code(), tt.description)
			} else {
				assert.NoError(t, err, tt.description)
			}
		})
	}
}

func TestStreamAppendMetadataInterceptor_NoMetadata(t *testing.T) {
	// Test case where metadata cannot be extracted from stream
	ctx := context.Background() // No metadata context

	interceptor := StreamAppendMetadataInterceptor
	stream := &MockServerStream{ctx: ctx}
	info := &grpc.StreamServerInfo{
		FullMethod: "test.Service/StreamMethod",
	}

	handler := func(srv any, stream grpc.ServerStream) error {
		// Verify that empty metadata is created when none exists
		md, ok := metadata.FromIncomingContext(stream.Context())
		assert.True(t, ok, "metadata should be present even when none was provided")
		assert.Empty(t, md, "metadata should be empty when none was provided")
		return nil
	}

	// This should succeed and create empty metadata
	err := interceptor(nil, stream, info, handler)

	assert.NoError(t, err, "should not return error when no metadata is provided")
}

func TestStreamAppendMetadataInterceptor_ContextPreservation(t *testing.T) {
	// Test that the context is properly preserved in the wrapped stream
	originalCtx := metadata.NewIncomingContext(context.Background(), metadata.New(map[string]string{
		"user-id": "123",
		"token":   "abc123",
	}))

	interceptor := StreamAppendMetadataInterceptor
	stream := &MockServerStream{ctx: originalCtx}
	info := &grpc.StreamServerInfo{
		FullMethod: "test.Service/StreamMethod",
	}

	var capturedCtx context.Context
	handler := func(srv any, stream grpc.ServerStream) error {
		capturedCtx = stream.Context()

		// Verify metadata is preserved
		md, ok := metadata.FromIncomingContext(capturedCtx)
		assert.True(t, ok, "metadata should be present in captured context")
		assert.Equal(t, "123", md.Get("user-id")[0])
		assert.Equal(t, "abc123", md.Get("token")[0])

		return nil
	}

	err := interceptor(nil, stream, info, handler)
	assert.NoError(t, err, "should not return error for successful handler")

	// Verify context was captured and is different from original
	assert.NotNil(t, capturedCtx, "context should be captured")
	assert.NotEqual(t, originalCtx, capturedCtx, "context should be wrapped")
}

func TestStreamAppendMetadataInterceptor_StreamMethods(t *testing.T) {
	// Test that the wrapped stream properly implements all required methods
	ctx := metadata.NewIncomingContext(context.Background(), metadata.New(map[string]string{
		"test-key": "test-value",
	}))

	interceptor := StreamAppendMetadataInterceptor
	originalStream := &MockServerStream{ctx: ctx}
	info := &grpc.StreamServerInfo{
		FullMethod: "test.Service/StreamMethod",
	}

	var wrappedStream grpc.ServerStream
	handler := func(srv any, stream grpc.ServerStream) error {
		wrappedStream = stream

		// Test that stream methods work
		assert.NotNil(t, stream.Context(), "Context() should return non-nil context")
		assert.NoError(t, stream.RecvMsg("test"), "RecvMsg should not error")
		assert.NoError(t, stream.SendMsg("test"), "SendMsg should not error")
		assert.NoError(t, stream.SendHeader(metadata.New(map[string]string{})), "SendHeader should not error")
		assert.NoError(t, stream.SetHeader(metadata.New(map[string]string{})), "SetHeader should not error")
		stream.SetTrailer(metadata.New(map[string]string{})) // SetTrailer has no return value

		return nil
	}

	err := interceptor(nil, originalStream, info, handler)
	assert.NoError(t, err, "should not return error")
	assert.NotNil(t, wrappedStream, "wrapped stream should be created")
}

func TestUnaryAppendMetadataInterceptor_ErrorConversion(t *testing.T) {
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
		t.Run(tc.name, func(t *testing.T) {
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
				assert.NoError(t, err, tc.description)
				assert.Equal(t, "response", resp, tc.description)
			} else {
				assert.Error(t, err, tc.description)
				st, ok := status.FromError(err)
				assert.True(t, ok, "should be a gRPC status error")
				assert.Equal(t, tc.expectedCode, st.Code(), tc.description)
			}
		})
	}
}

func TestStreamAppendMetadataInterceptor_ErrorConversion(t *testing.T) {
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
		t.Run(tc.name, func(t *testing.T) {
			ctx := metadata.NewIncomingContext(context.Background(), metadata.New(map[string]string{
				"test-key": "test-value",
			}))

			interceptor := StreamAppendMetadataInterceptor
			stream := &MockServerStream{ctx: ctx}
			info := &grpc.StreamServerInfo{
				FullMethod: "test.Service/StreamMethod",
			}

			handler := func(srv any, stream grpc.ServerStream) error {
				return tc.handlerError
			}

			err := interceptor(nil, stream, info, handler)

			if tc.handlerError == nil {
				assert.NoError(t, err, tc.description)
			} else {
				assert.Error(t, err, tc.description)
				st, ok := status.FromError(err)
				assert.True(t, ok, "should be a gRPC status error")
				assert.Equal(t, tc.expectedCode, st.Code(), tc.description)
			}
		})
	}
}

func TestUnaryAppendMetadataInterceptor_MetadataPreservation(t *testing.T) {
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

	assert.NoError(t, err, "should not return error")
	assert.Equal(t, "response", resp, "should return expected response")
	assert.NotNil(t, capturedMD, "metadata should be captured")

	// Verify all metadata keys are preserved
	for key, values := range originalMD {
		assert.Equal(t, values, capturedMD.Get(key), "metadata key %s should be preserved", key)
	}
}

func TestStreamAppendMetadataInterceptor_MetadataPreservation(t *testing.T) {
	// Test that metadata is properly preserved and accessible in stream handler
	originalMD := metadata.New(map[string]string{
		"user-id":    "123",
		"token":      "abc123",
		"request-id": "req-456",
		"trace-id":   "trace-789",
	})

	ctx := metadata.NewIncomingContext(context.Background(), originalMD)

	interceptor := StreamAppendMetadataInterceptor
	stream := &MockServerStream{ctx: ctx}
	info := &grpc.StreamServerInfo{
		FullMethod: "test.Service/StreamMethod",
	}

	var capturedMD metadata.MD
	handler := func(srv any, stream grpc.ServerStream) error {
		capturedMD, _ = metadata.FromIncomingContext(stream.Context())
		return nil
	}

	err := interceptor(nil, stream, info, handler)

	assert.NoError(t, err, "should not return error")
	assert.NotNil(t, capturedMD, "metadata should be captured")

	// Verify all metadata keys are preserved
	for key, values := range originalMD {
		assert.Equal(t, values, capturedMD.Get(key), "metadata key %s should be preserved", key)
	}
}
