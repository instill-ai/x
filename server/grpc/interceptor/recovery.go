package interceptor

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
)

// RecoveryInterceptorOpt - panic handler
func RecoveryInterceptorOpt() grpc_recovery.Option {
	return grpc_recovery.WithRecoveryHandler(func(p any) (err error) {
		return status.Errorf(codes.Unknown, "panic triggered: %v", p)
	})
}
