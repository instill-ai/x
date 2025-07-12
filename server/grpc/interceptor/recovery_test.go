package interceptor

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
)

func TestRecoveryInterceptorOpt(t *testing.T) {
	// Test that RecoveryInterceptorOpt returns a valid option
	opt := RecoveryInterceptorOpt()
	assert.NotNil(t, opt, "RecoveryInterceptorOpt should return a non-nil option")

	// Test that the option can be used to create interceptors
	unaryInterceptor := grpc_recovery.UnaryServerInterceptor(opt)
	assert.NotNil(t, unaryInterceptor, "should create unary interceptor with the option")
}

func TestRecoveryHandler(t *testing.T) {
	tests := []struct {
		name         string
		panicValue   any
		expectedCode codes.Code
		expectedMsg  string
		description  string
	}{
		{
			name:         "string panic",
			panicValue:   "test panic",
			expectedCode: codes.Unknown,
			expectedMsg:  "panic triggered: test panic",
			description:  "should handle string panic values",
		},
		{
			name:         "error panic",
			panicValue:   errors.New("test error"),
			expectedCode: codes.Unknown,
			expectedMsg:  "panic triggered: test error",
			description:  "should handle error panic values",
		},
		{
			name:         "nil panic",
			panicValue:   nil,
			expectedCode: codes.Unknown,
			expectedMsg:  "panic triggered: panic called with nil argument",
			description:  "should handle nil panic values",
		},
		{
			name:         "int panic",
			panicValue:   42,
			expectedCode: codes.Unknown,
			expectedMsg:  "panic triggered: 42",
			description:  "should handle int panic values",
		},
		{
			name:         "struct panic",
			panicValue:   struct{ Name string }{Name: "test"},
			expectedCode: codes.Unknown,
			expectedMsg:  "panic triggered: {test}",
			description:  "should handle struct panic values",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Get the recovery option
			opt := RecoveryInterceptorOpt()

			// Extract the recovery handler from the option
			// This is a bit tricky since we need to access the internal handler
			// We'll test it by creating a recovery interceptor and triggering a panic

			// Create unary recovery interceptor
			unaryInterceptor := grpc_recovery.UnaryServerInterceptor(opt)

			// Create a handler that panics with the test value
			panicHandler := func(ctx context.Context, req any) (any, error) {
				panic(tt.panicValue)
			}

			// Create mock request info
			info := &grpc.UnaryServerInfo{
				FullMethod: "test.Service/Method",
			}

			// Call the interceptor and expect it to recover
			resp, err := unaryInterceptor(context.Background(), "test request", info, panicHandler)

			// Verify the response is nil (panic was recovered)
			assert.Nil(t, resp, tt.description)

			// Verify the error is a gRPC status error
			assert.Error(t, err, tt.description)
			st, ok := status.FromError(err)
			assert.True(t, ok, tt.description)
			assert.Equal(t, tt.expectedCode, st.Code(), tt.description)
			assert.Equal(t, tt.expectedMsg, st.Message(), tt.description)
		})
	}
}

func TestRecoveryInterceptorIntegration(t *testing.T) {
	// Test that the recovery interceptor works in a real scenario
	opt := RecoveryInterceptorOpt()
	unaryInterceptor := grpc_recovery.UnaryServerInterceptor(opt)

	// Test successful handler (no panic)
	successHandler := func(ctx context.Context, req any) (any, error) {
		return "success", nil
	}

	info := &grpc.UnaryServerInfo{
		FullMethod: "test.Service/SuccessMethod",
	}

	resp, err := unaryInterceptor(context.Background(), "test request", info, successHandler)
	assert.NoError(t, err, "should not error for successful handler")
	assert.Equal(t, "success", resp, "should return success response")

	// Test handler that returns an error (no panic)
	errorHandler := func(ctx context.Context, req any) (any, error) {
		return nil, status.Error(codes.InvalidArgument, "invalid argument")
	}

	resp, err = unaryInterceptor(context.Background(), "test request", info, errorHandler)
	assert.Error(t, err, "should return error from handler")
	assert.Nil(t, resp, "should return nil response when handler errors")
	st, ok := status.FromError(err)
	assert.True(t, ok, "should be a gRPC status error")
	assert.Equal(t, codes.InvalidArgument, st.Code(), "should preserve original error code")
}

func TestRecoveryInterceptorStream(t *testing.T) {
	// Test stream recovery interceptor
	opt := RecoveryInterceptorOpt()
	streamInterceptor := grpc_recovery.StreamServerInterceptor(opt)

	// Test stream handler that panics
	panicStreamHandler := func(srv any, stream grpc.ServerStream) error {
		panic("stream panic")
	}

	info := &grpc.StreamServerInfo{
		FullMethod: "test.Service/StreamMethod",
	}

	// Create a mock stream
	mockStream := &MockServerStream{ctx: context.Background()}

	// Call the interceptor and expect it to recover
	err := streamInterceptor(nil, mockStream, info, panicStreamHandler)

	// Verify the error is a gRPC status error
	assert.Error(t, err, "should return error for panic")
	st, ok := status.FromError(err)
	assert.True(t, ok, "should be a gRPC status error")
	assert.Equal(t, codes.Unknown, st.Code(), "should return Unknown code for panic")
	assert.Equal(t, "panic triggered: stream panic", st.Message(), "should have correct panic message")
}

func TestRecoveryInterceptorNilPanic(t *testing.T) {
	// Test recovery from nil panic
	opt := RecoveryInterceptorOpt()
	unaryInterceptor := grpc_recovery.UnaryServerInterceptor(opt)

	nilPanicHandler := func(ctx context.Context, req any) (any, error) {
		panic(nil)
	}

	info := &grpc.UnaryServerInfo{
		FullMethod: "test.Service/NilPanicMethod",
	}

	resp, err := unaryInterceptor(context.Background(), "test request", info, nilPanicHandler)

	assert.Nil(t, resp, "should return nil response for nil panic")
	assert.Error(t, err, "should return error for nil panic")
	st, ok := status.FromError(err)
	assert.True(t, ok, "should be a gRPC status error")
	assert.Equal(t, codes.Unknown, st.Code(), "should return Unknown code for nil panic")
	assert.Equal(t, "panic triggered: panic called with nil argument", st.Message(), "should handle nil panic correctly")
}

func TestRecoveryInterceptorComplexPanic(t *testing.T) {
	// Test recovery from complex panic values
	opt := RecoveryInterceptorOpt()
	unaryInterceptor := grpc_recovery.UnaryServerInterceptor(opt)

	complexPanicHandler := func(ctx context.Context, req any) (any, error) {
		panic(map[string]any{
			"error":   "complex error",
			"code":    500,
			"details": []string{"detail1", "detail2"},
		})
	}

	info := &grpc.UnaryServerInfo{
		FullMethod: "test.Service/ComplexPanicMethod",
	}

	resp, err := unaryInterceptor(context.Background(), "test request", info, complexPanicHandler)

	assert.Nil(t, resp, "should return nil response for complex panic")
	assert.Error(t, err, "should return error for complex panic")
	st, ok := status.FromError(err)
	assert.True(t, ok, "should be a gRPC status error")
	assert.Equal(t, codes.Unknown, st.Code(), "should return Unknown code for complex panic")
	// The exact message format may vary depending on how the map is stringified
	assert.Contains(t, st.Message(), "panic triggered:", "should contain panic prefix")
}

// Test that the recovery option can be used with both unary and stream interceptors
func TestRecoveryOptionCompatibility(t *testing.T) {
	opt := RecoveryInterceptorOpt()

	// Test that the option can be used with unary interceptor
	unaryInterceptor := grpc_recovery.UnaryServerInterceptor(opt)
	assert.NotNil(t, unaryInterceptor, "should create unary interceptor")

	// Test that the option can be used with stream interceptor
	streamInterceptor := grpc_recovery.StreamServerInterceptor(opt)
	assert.NotNil(t, streamInterceptor, "should create stream interceptor")
}

// Test that the recovery handler preserves the panic value in the error message
func TestRecoveryHandlerPreservesPanicValue(t *testing.T) {
	opt := RecoveryInterceptorOpt()
	unaryInterceptor := grpc_recovery.UnaryServerInterceptor(opt)

	// Test with a custom error type
	customError := errors.New("custom error message")
	panicHandler := func(ctx context.Context, req any) (any, error) {
		panic(customError)
	}

	info := &grpc.UnaryServerInfo{
		FullMethod: "test.Service/CustomErrorMethod",
	}

	resp, err := unaryInterceptor(context.Background(), "test request", info, panicHandler)

	assert.Nil(t, resp, "should return nil response")
	assert.Error(t, err, "should return error")
	st, ok := status.FromError(err)
	assert.True(t, ok, "should be a gRPC status error")
	assert.Equal(t, codes.Unknown, st.Code(), "should return Unknown code")
	assert.Equal(t, "panic triggered: custom error message", st.Message(), "should preserve custom error message")
}
