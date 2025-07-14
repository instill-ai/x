package interceptor

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/frankban/quicktest"
	"github.com/gojuno/minimock/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"

	mockserver "github.com/instill-ai/x/mock/server"
)

func TestRecoveryInterceptorOpt(t *testing.T) {
	qt := quicktest.New(t)

	// Test that RecoveryInterceptorOpt returns a valid option
	opt := RecoveryInterceptorOpt()
	qt.Check(opt, quicktest.Not(quicktest.IsNil))

	// Test that the option can be used to create interceptors
	unaryInterceptor := grpc_recovery.UnaryServerInterceptor(opt)
	qt.Check(unaryInterceptor, quicktest.Not(quicktest.IsNil))
}

func TestRecoveryHandler(t *testing.T) {
	qt := quicktest.New(t)

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
		qt.Run(tt.name, func(c *quicktest.C) {
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
			c.Check(resp, quicktest.IsNil)

			// Verify the error is a gRPC status error
			c.Check(err, quicktest.Not(quicktest.IsNil))
			st, ok := status.FromError(err)
			c.Check(ok, quicktest.IsTrue)
			c.Check(st.Code(), quicktest.Equals, tt.expectedCode)
			c.Check(st.Message(), quicktest.Equals, tt.expectedMsg)
		})
	}
}

func TestRecoveryInterceptorIntegration(t *testing.T) {
	qt := quicktest.New(t)

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
	qt.Check(err, quicktest.IsNil)
	qt.Check(resp, quicktest.Equals, "success")

	// Test handler that returns an error (no panic)
	errorHandler := func(ctx context.Context, req any) (any, error) {
		return nil, status.Error(codes.InvalidArgument, "invalid argument")
	}

	resp, err = unaryInterceptor(context.Background(), "test request", info, errorHandler)
	qt.Check(err, quicktest.Not(quicktest.IsNil))
	qt.Check(resp, quicktest.IsNil)
	st, ok := status.FromError(err)
	qt.Check(ok, quicktest.IsTrue)
	qt.Check(st.Code(), quicktest.Equals, codes.InvalidArgument)
}

func TestRecoveryInterceptorStream(t *testing.T) {
	qt := quicktest.New(t)

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
	mc := minimock.NewController(t)
	mockStream := mockserver.NewServerStreamMock(mc)

	// Add this line to set up the Context expectation
	ctx := context.Background()
	mockStream.ContextMock.Expect().Return(ctx)

	// Call the interceptor and expect it to recover
	err := streamInterceptor(nil, mockStream, info, panicStreamHandler)

	// Verify the error is a gRPC status error
	qt.Check(err, quicktest.Not(quicktest.IsNil))
	st, ok := status.FromError(err)
	qt.Check(ok, quicktest.IsTrue)
	qt.Check(st.Code(), quicktest.Equals, codes.Unknown)
	qt.Check(st.Message(), quicktest.Equals, "panic triggered: stream panic")
}

func TestRecoveryInterceptorNilPanic(t *testing.T) {
	qt := quicktest.New(t)

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

	qt.Check(resp, quicktest.IsNil)
	qt.Check(err, quicktest.Not(quicktest.IsNil))
	st, ok := status.FromError(err)
	qt.Check(ok, quicktest.IsTrue)
	qt.Check(st.Code(), quicktest.Equals, codes.Unknown)
	qt.Check(st.Message(), quicktest.Equals, "panic triggered: panic called with nil argument")
}

func TestRecoveryInterceptorComplexPanic(t *testing.T) {
	qt := quicktest.New(t)

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

	qt.Check(resp, quicktest.IsNil)
	qt.Check(err, quicktest.Not(quicktest.IsNil))
	st, ok := status.FromError(err)
	qt.Check(ok, quicktest.IsTrue)
	qt.Check(st.Code(), quicktest.Equals, codes.Unknown)
	// The exact message format may vary depending on how the map is stringified
	qt.Check(strings.Contains(st.Message(), "panic triggered:"), quicktest.IsTrue)
}

// Test that the recovery option can be used with both unary and stream interceptors
func TestRecoveryOptionCompatibility(t *testing.T) {
	qt := quicktest.New(t)

	opt := RecoveryInterceptorOpt()

	// Test that the option can be used with unary interceptor
	unaryInterceptor := grpc_recovery.UnaryServerInterceptor(opt)
	qt.Check(unaryInterceptor, quicktest.Not(quicktest.IsNil))

	// Test that the option can be used with stream interceptor
	streamInterceptor := grpc_recovery.StreamServerInterceptor(opt)
	qt.Check(streamInterceptor, quicktest.Not(quicktest.IsNil))
}

// Test that the recovery handler preserves the panic value in the error message
func TestRecoveryHandlerPreservesPanicValue(t *testing.T) {
	qt := quicktest.New(t)

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

	qt.Check(resp, quicktest.IsNil)
	qt.Check(err, quicktest.Not(quicktest.IsNil))
	st, ok := status.FromError(err)
	qt.Check(ok, quicktest.IsTrue)
	qt.Check(st.Code(), quicktest.Equals, codes.Unknown)
	qt.Check(st.Message(), quicktest.Equals, "panic triggered: custom error message")
}
