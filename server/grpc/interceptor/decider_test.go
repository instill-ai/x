package interceptor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

func TestDecideLogGRPCRequest(t *testing.T) {
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
			err:                   assert.AnError,
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
		t.Run(tt.name, func(t *testing.T) {
			result := decideLogGRPCRequest(tt.fullMethod, tt.methodExcludePatterns, tt.err)
			assert.Equal(t, tt.expectedShouldLog, result, tt.description)
		})
	}
}

func TestDeciderUnaryServerInterceptor(t *testing.T) {
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
		t.Run(tt.name, func(t *testing.T) {
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
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedShouldLog, shouldLogValue, tt.description)
		})
	}
}

func TestDeciderStreamServerInterceptor(t *testing.T) {
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
		t.Run(tt.name, func(t *testing.T) {
			interceptor := DeciderStreamServerInterceptor(tt.methodExcludePatterns)

			ctx := context.Background()
			stream := &MockServerStream{ctx: ctx}
			info := &grpc.StreamServerInfo{
				FullMethod: tt.fullMethod,
			}

			var shouldLogValue bool
			handler := func(srv any, stream grpc.ServerStream) error {
				shouldLogValue = ShouldLogFromContext(stream.Context())
				return nil
			}

			err := interceptor(nil, stream, info, handler)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedShouldLog, shouldLogValue, tt.description)
		})
	}
}

func TestShouldLogFromContext(t *testing.T) {
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
		t.Run(tt.name, func(t *testing.T) {
			result := ShouldLogFromContext(tt.ctx)
			assert.Equal(t, tt.expectedResult, result, tt.description)
		})
	}
}

func TestDefaultMethodExcludePatterns(t *testing.T) {
	// Test that default patterns are correctly defined
	assert.Len(t, DefaultMethodExcludePatterns, 2)
	assert.Contains(t, DefaultMethodExcludePatterns, "*PublicService/.*ness$")
	assert.Contains(t, DefaultMethodExcludePatterns, "*PrivateService/.*$")
}

// Test that the deciderServerStream wrapper works correctly
func TestDeciderServerStream(t *testing.T) {
	originalCtx := context.Background()
	originalStream := &MockServerStream{ctx: originalCtx}

	wrappedStream := &deciderServerStream{
		ServerStream: originalStream,
		ctx:          context.WithValue(originalCtx, shouldLogKey, false),
	}

	// Test that Context() returns the wrapped context
	wrappedCtx := wrappedStream.Context()
	shouldLog := ShouldLogFromContext(wrappedCtx)
	assert.False(t, shouldLog)

	// Test that the original context is not affected
	originalShouldLog := ShouldLogFromContext(originalCtx)
	assert.True(t, originalShouldLog) // should default to true
}

// Test integration between interceptors and context
func TestInterceptorIntegration(t *testing.T) {
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
	assert.NoError(t, err)

	// Verify the context was modified
	shouldLog := ShouldLogFromContext(capturedCtx)
	assert.True(t, shouldLog) // should match the pattern

	// Test with a method that doesn't match
	info2 := &grpc.UnaryServerInfo{FullMethod: "test.OtherService/method"}
	var capturedCtx2 context.Context
	handler2 := func(ctx context.Context, req any) (any, error) {
		capturedCtx2 = ctx
		return nil, nil
	}

	_, err = interceptor(ctx, req, info2, handler2)
	assert.NoError(t, err)

	shouldLog2 := ShouldLogFromContext(capturedCtx2)
	assert.True(t, shouldLog2) // should not match the pattern, but still log
}
