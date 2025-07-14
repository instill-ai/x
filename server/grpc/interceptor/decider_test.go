package interceptor

import (
	"context"
	"errors"
	"testing"

	"github.com/frankban/quicktest"
	"github.com/gojuno/minimock/v3"
	"google.golang.org/grpc"

	mockserver "github.com/instill-ai/x/mock/server"
)

func TestDecideLogGRPCRequest(t *testing.T) {
	qt := quicktest.New(t)

	tests := []struct {
		name                  string
		fullMethod            string
		methodExcludePatterns []string
		err                   error
		expectedShouldLog     bool
		description           string
	}{
		{
			name:                  "no error, no patterns, should log",
			fullMethod:            "test.Service/Method",
			methodExcludePatterns: []string{},
			err:                   nil,
			expectedShouldLog:     true,
			description:           "when no error and no exclude patterns, should log",
		},
		{
			name:                  "no error, pattern matches, should log",
			fullMethod:            "test.PublicService/liveness",
			methodExcludePatterns: []string{"*PublicService/.*ness$"},
			err:                   nil,
			expectedShouldLog:     true, // matches pattern, so should log
			description:           "when pattern matches, should log",
		},
		{
			name:                  "no error, pattern does not match, should log",
			fullMethod:            "test.Service/Method",
			methodExcludePatterns: []string{"*PublicService/.*ness$"},
			err:                   nil,
			expectedShouldLog:     true,
			description:           "when pattern does not match, should log",
		},
		{
			name:                  "with error, pattern matches, should log",
			fullMethod:            "test.PublicService/liveness",
			methodExcludePatterns: []string{"*PublicService/.*ness$"},
			err:                   errors.New("test error"),
			expectedShouldLog:     true,
			description:           "when there is an error, should always log regardless of patterns",
		},
		{
			name:                  "multiple patterns, one matches",
			fullMethod:            "test.PrivateService/method",
			methodExcludePatterns: []string{"*PublicService/.*ness$", "*PrivateService/.*$"},
			err:                   nil,
			expectedShouldLog:     true, // matches second pattern
			description:           "when one of multiple patterns matches, should log",
		},
		{
			name:                  "multiple patterns, none match",
			fullMethod:            "test.Service/method",
			methodExcludePatterns: []string{"*PublicService/.*ness$", "*PrivateService/.*$"},
			err:                   nil,
			expectedShouldLog:     true,
			description:           "when none of multiple patterns match, should log",
		},
	}

	for _, tt := range tests {
		qt.Run(tt.name, func(c *quicktest.C) {
			result := decideLogGRPCRequest(tt.fullMethod, tt.methodExcludePatterns, tt.err)
			c.Check(result, quicktest.Equals, tt.expectedShouldLog)
		})
	}
}

func TestDeciderUnaryServerInterceptor(t *testing.T) {
	qt := quicktest.New(t)

	tests := []struct {
		name                  string
		methodExcludePatterns []string
		fullMethod            string
		expectedShouldLog     bool
		description           string
	}{
		{
			name:                  "nil patterns, uses defaults",
			methodExcludePatterns: nil,
			fullMethod:            "test.PublicService/liveness",
			expectedShouldLog:     true, // matches default pattern
			description:           "when nil patterns provided, should use default patterns",
		},
		{
			name:                  "empty patterns, uses defaults",
			methodExcludePatterns: []string{},
			fullMethod:            "test.PrivateService/method",
			expectedShouldLog:     true, // matches default pattern
			description:           "when empty patterns provided, should use default patterns",
		},
		{
			name:                  "custom patterns appended to defaults",
			methodExcludePatterns: []string{"*CustomService/.*$"},
			fullMethod:            "test.CustomService/method",
			expectedShouldLog:     true, // matches custom pattern
			description:           "when custom patterns provided, should append to defaults",
		},
		{
			name:                  "default patterns still work with custom patterns",
			methodExcludePatterns: []string{"*CustomService/.*$"},
			fullMethod:            "test.PublicService/liveness",
			expectedShouldLog:     true, // matches default pattern
			description:           "default patterns should still work when custom patterns are provided",
		},
	}

	for _, tt := range tests {
		qt.Run(tt.name, func(c *quicktest.C) {
			interceptor := DeciderUnaryServerInterceptor(tt.methodExcludePatterns)

			ctx := context.Background()
			req := "test request"
			info := &grpc.UnaryServerInfo{
				FullMethod: tt.fullMethod,
			}

			var shouldLogValue bool
			handler := func(ctx context.Context, req any) (any, error) {
				shouldLogValue = ShouldLogFromContext(ctx)
				return "response", nil
			}

			_, err := interceptor(ctx, req, info, handler)
			c.Check(err, quicktest.IsNil)
			c.Check(shouldLogValue, quicktest.Equals, tt.expectedShouldLog)
		})
	}
}

func TestDeciderStreamServerInterceptor(t *testing.T) {
	qt := quicktest.New(t)

	tests := []struct {
		name                  string
		methodExcludePatterns []string
		fullMethod            string
		expectedShouldLog     bool
		description           string
	}{
		{
			name:                  "nil patterns, uses defaults",
			methodExcludePatterns: nil,
			fullMethod:            "test.PublicService/liveness",
			expectedShouldLog:     true, // matches default pattern
			description:           "when nil patterns provided, should use default patterns",
		},
		{
			name:                  "empty patterns, uses defaults",
			methodExcludePatterns: []string{},
			fullMethod:            "test.PrivateService/method",
			expectedShouldLog:     true, // matches default pattern
			description:           "when empty patterns provided, should use default patterns",
		},
		{
			name:                  "custom patterns appended to defaults",
			methodExcludePatterns: []string{"*CustomService/.*$"},
			fullMethod:            "test.CustomService/method",
			expectedShouldLog:     true, // matches custom pattern
			description:           "when custom patterns provided, should append to defaults",
		},
	}

	for _, tt := range tests {
		qt.Run(tt.name, func(c *quicktest.C) {
			interceptor := DeciderStreamServerInterceptor(tt.methodExcludePatterns)

			mc := minimock.NewController(t)
			stream := mockserver.NewServerStreamMock(mc)

			// Add this line to set up the Context expectation
			ctx := context.Background()
			stream.ContextMock.Expect().Return(ctx)

			info := &grpc.StreamServerInfo{
				FullMethod: tt.fullMethod,
			}

			var shouldLogValue bool
			handler := func(srv any, stream grpc.ServerStream) error {
				shouldLogValue = ShouldLogFromContext(stream.Context())
				return nil
			}

			err := interceptor(nil, stream, info, handler)
			c.Check(err, quicktest.IsNil)
			c.Check(shouldLogValue, quicktest.Equals, tt.expectedShouldLog)
		})
	}
}

func TestShouldLogFromContext(t *testing.T) {
	qt := quicktest.New(t)

	tests := []struct {
		name           string
		ctx            context.Context
		expectedResult bool
		description    string
	}{
		{
			name:           "context with shouldLog true",
			ctx:            context.WithValue(context.Background(), shouldLogKey, true),
			expectedResult: true,
			description:    "when context has shouldLog=true, should return true",
		},
		{
			name:           "context with shouldLog false",
			ctx:            context.WithValue(context.Background(), shouldLogKey, false),
			expectedResult: false,
			description:    "when context has shouldLog=false, should return false",
		},
		{
			name:           "context without shouldLog key",
			ctx:            context.Background(),
			expectedResult: true,
			description:    "when context has no shouldLog key, should default to true",
		},
		{
			name:           "context with wrong type value",
			ctx:            context.WithValue(context.Background(), shouldLogKey, "not a bool"),
			expectedResult: true,
			description:    "when context has shouldLog key with wrong type, should default to true",
		},
	}

	for _, tt := range tests {
		qt.Run(tt.name, func(c *quicktest.C) {
			result := ShouldLogFromContext(tt.ctx)
			c.Check(result, quicktest.Equals, tt.expectedResult)
		})
	}
}

func TestDefaultMethodExcludePatterns(t *testing.T) {
	qt := quicktest.New(t)

	// Test that default patterns are correctly defined
	qt.Check(len(DefaultMethodExcludePatterns), quicktest.Equals, 2)

	// Check if patterns contain expected values
	hasPublicPattern := false
	hasPrivatePattern := false
	for _, pattern := range DefaultMethodExcludePatterns {
		if pattern == "*PublicService/.*ness$" {
			hasPublicPattern = true
		}
		if pattern == "*PrivateService/.*$" {
			hasPrivatePattern = true
		}
	}
	qt.Check(hasPublicPattern, quicktest.IsTrue)
	qt.Check(hasPrivatePattern, quicktest.IsTrue)
}

// Test that the deciderServerStream wrapper works correctly
func TestDeciderServerStream(t *testing.T) {
	qt := quicktest.New(t)

	originalCtx := context.Background()
	mc := minimock.NewController(t)
	originalStream := mockserver.NewServerStreamMock(mc)

	wrappedStream := &deciderServerStream{
		ServerStream: originalStream,
		ctx:          context.WithValue(originalCtx, shouldLogKey, false),
	}

	// Test that Context() returns the wrapped context
	wrappedCtx := wrappedStream.Context()
	shouldLog := ShouldLogFromContext(wrappedCtx)
	qt.Check(shouldLog, quicktest.IsFalse)

	// Test that the original context is not affected
	originalShouldLog := ShouldLogFromContext(originalCtx)
	qt.Check(originalShouldLog, quicktest.IsTrue) // should default to true
}

// Test integration between interceptors and context
func TestInterceptorIntegration(t *testing.T) {
	qt := quicktest.New(t)

	// Test that the unary interceptor properly sets context
	interceptor := DeciderUnaryServerInterceptor([]string{"*TestService/.*$"})

	ctx := context.Background()
	req := "test"
	info := &grpc.UnaryServerInfo{FullMethod: "test.TestService/method"}

	var capturedCtx context.Context
	handler := func(ctx context.Context, req any) (any, error) {
		capturedCtx = ctx
		return nil, nil
	}

	_, err := interceptor(ctx, req, info, handler)
	qt.Check(err, quicktest.IsNil)

	// Verify the context was modified
	shouldLog := ShouldLogFromContext(capturedCtx)
	qt.Check(shouldLog, quicktest.IsTrue) // should match the pattern

	// Test with a method that doesn't match
	info2 := &grpc.UnaryServerInfo{FullMethod: "test.OtherService/method"}
	var capturedCtx2 context.Context
	handler2 := func(ctx context.Context, req any) (any, error) {
		capturedCtx2 = ctx
		return nil, nil
	}

	_, err = interceptor(ctx, req, info2, handler2)
	qt.Check(err, quicktest.IsNil)

	shouldLog2 := ShouldLogFromContext(capturedCtx2)
	qt.Check(shouldLog2, quicktest.IsTrue) // should not match the pattern, but still log
}
