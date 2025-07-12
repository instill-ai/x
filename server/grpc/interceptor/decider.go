package interceptor

import (
	"context"
	"regexp"

	"google.golang.org/grpc"
)

type contextKey string

const shouldLogKey contextKey = "shouldLog"

// DefaultMethodExcludePatterns is always included in the methodExcludePatterns used for matching.
var DefaultMethodExcludePatterns = []string{
	// stop logging gRPC calls if it was a call to liveness or readiness and no error was raised
	"*PublicService/.*ness$",
	// stop logging gRPC calls if it was a call to a private function and no error was raised
	"*PrivateService/.*$",
}

// DeciderUnaryServerInterceptor returns a unary interceptor that sets a shouldLog flag in context based on methodExcludePatterns.
// DefaultMethodExcludePatterns is always included, and any provided methodExcludePatterns are appended.
func DeciderUnaryServerInterceptor(methodExcludePatterns []string) grpc.UnaryServerInterceptor {
	methods := append([]string{}, DefaultMethodExcludePatterns...)
	methods = append(methods, methodExcludePatterns...)
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		shouldLog := decideLogGRPCRequest(info.FullMethod, methods, nil) // error is not available at this stage
		ctx = context.WithValue(ctx, shouldLogKey, shouldLog)
		return handler(ctx, req)
	}
}

// DeciderStreamServerInterceptor returns a stream interceptor that sets a shouldLog flag in context based on methodExcludePatterns.
// DefaultMethodExcludePatterns is always included, and any provided methodExcludePatterns are appended.
func DeciderStreamServerInterceptor(methodExcludePatterns []string) grpc.StreamServerInterceptor {
	methods := append([]string{}, DefaultMethodExcludePatterns...)
	methods = append(methods, methodExcludePatterns...)
	return func(srv any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		shouldLog := decideLogGRPCRequest(info.FullMethod, methods, nil) // error is not available at this stage
		wrapped := &deciderServerStream{ServerStream: stream, ctx: context.WithValue(stream.Context(), shouldLogKey, shouldLog)}
		return handler(srv, wrapped)
	}
}

type deciderServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (d *deciderServerStream) Context() context.Context {
	return d.ctx
}

// decideLogGRPCRequest returns true if the fullMethod matches any of methodExcludePatterns, otherwise false.
func decideLogGRPCRequest(fullMethod string, methodExcludePatterns []string, err error) bool {
	if err == nil {
		for _, method := range methodExcludePatterns {
			if match, _ := regexp.MatchString(method, fullMethod); match {
				return true
			}
		}
	}
	return true
}

// ShouldLogFromContext returns the shouldLog flag from context, defaulting to true if not set
func ShouldLogFromContext(ctx context.Context) bool {
	shouldLog, ok := ctx.Value(shouldLogKey).(bool)
	if !ok {
		return true
	}
	return shouldLog
}
