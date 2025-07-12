package interceptor

import (
	"context"
	"fmt"
	"strings"
	"time"

	"google.golang.org/grpc/codes"

	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"

	errorsx "github.com/instill-ai/x/errors"

	logx "github.com/instill-ai/x/log"
)

// TracingUnaryServerInterceptor creates a unary interceptor that includes trace context in logs
func TracingUnaryServerInterceptor(serviceName string, serviceVersion string, OTELCollectorEnable bool) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		startTime := time.Now()

		resp, err := handler(ctx, req)

		// Only log if the decider allows it
		if ShouldLogFromContext(ctx) {
			duration := time.Since(startTime)
			code := errorsx.ConvertGRPCCode(err)
			logGRPCRequest(
				withContext(ctx),
				withServiceInfo(serviceName, serviceVersion),
				withMethodInfo("unary", info.FullMethod),
				withTiming(startTime, duration),
				withCode(code),
				withOTELEnable(OTELCollectorEnable),
			)
		}

		return resp, err
	}
}

// TracingStreamServerInterceptor creates a stream interceptor that includes trace context in logs
func TracingStreamServerInterceptor(serviceName string, serviceVersion string, OTELCollectorEnable bool) grpc.StreamServerInterceptor {
	return func(srv any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		startTime := time.Now()

		err := handler(srv, stream)

		// Only log if the decider allows it
		if ShouldLogFromContext(stream.Context()) {
			duration := time.Since(startTime)
			code := errorsx.ConvertGRPCCode(err)
			logGRPCRequest(
				withContext(stream.Context()),
				withServiceInfo(serviceName, serviceVersion),
				withMethodInfo("stream", info.FullMethod),
				withTiming(startTime, duration),
				withCode(code),
				withOTELEnable(OTELCollectorEnable),
			)
		}

		return err
	}
}

// gRPCRequestLogOptions contains all the options for logging a gRPC request
type gRPCRequestLogOptions struct {
	Context             context.Context
	ServiceName         string
	ServiceVersion      string
	MethodType          string
	FullMethod          string
	StartTime           time.Time
	Duration            time.Duration
	Code                codes.Code
	OTELCollectorEnable bool
}

// gRPCRequestLogOption is a function that modifies GRPCRequestLogOptions
type gRPCRequestLogOption func(*gRPCRequestLogOptions)

// withContext sets the context for the log request
func withContext(ctx context.Context) gRPCRequestLogOption {
	return func(opts *gRPCRequestLogOptions) {
		opts.Context = ctx
	}
}

// withServiceInfo sets the service name and version
func withServiceInfo(serviceName, serviceVersion string) gRPCRequestLogOption {
	return func(opts *gRPCRequestLogOptions) {
		opts.ServiceName = serviceName
		opts.ServiceVersion = serviceVersion
	}
}

// withMethodInfo sets the method type and full method name
func withMethodInfo(methodType, fullMethod string) gRPCRequestLogOption {
	return func(opts *gRPCRequestLogOptions) {
		opts.MethodType = methodType
		opts.FullMethod = fullMethod
	}
}

// withTiming sets the start time and duration
func withTiming(startTime time.Time, duration time.Duration) gRPCRequestLogOption {
	return func(opts *gRPCRequestLogOptions) {
		opts.StartTime = startTime
		opts.Duration = duration
	}
}

// withCode sets the gRPC status code
func withCode(code codes.Code) gRPCRequestLogOption {
	return func(opts *gRPCRequestLogOptions) {
		opts.Code = code
	}
}

// withOTELEnable sets the OpenTelemetry enable flag
func withOTELEnable(OTELCollectorEnable bool) gRPCRequestLogOption {
	return func(opts *gRPCRequestLogOptions) {
		opts.OTELCollectorEnable = OTELCollectorEnable
	}
}

// logGRPCRequest logs a gRPC request using the options pattern
func logGRPCRequest(opts ...gRPCRequestLogOption) {
	// Set default options
	options := &gRPCRequestLogOptions{
		Context: context.Background(),
	}

	// Apply all options
	for _, opt := range opts {
		opt(options)
	}

	// Always use the context-aware logger
	logger, _ := logx.GetZapLogger(options.Context)

	// Determine log level based on gRPC code
	var logLevel zapcore.Level
	var logFunc func(string, ...zap.Field)

	switch options.Code {
	case codes.OK:
		logLevel = zapcore.InfoLevel
		logFunc = logger.Info
	case codes.Canceled, codes.DeadlineExceeded, codes.PermissionDenied, codes.ResourceExhausted, codes.FailedPrecondition, codes.Aborted, codes.OutOfRange, codes.Unavailable, codes.DataLoss:
		logLevel = zapcore.WarnLevel
		logFunc = logger.Warn
	case codes.InvalidArgument, codes.NotFound, codes.AlreadyExists, codes.Unimplemented, codes.Internal, codes.Unauthenticated:
		logLevel = zapcore.ErrorLevel
		logFunc = logger.Error
	default:
		logLevel = zapcore.InfoLevel
		logFunc = logger.Info
	}

	// The suffix (trace_id: %s) is added to the message for Grafana Loki logs back-reference to Tempo traces.
	msg := fmt.Sprintf("finished %s call %s (trace_id: %s)",
		options.MethodType,
		extractMethodName(options.FullMethod),
		trace.SpanFromContext(options.Context).SpanContext().TraceID().String(),
	)

	logFunc(msg)

	if options.OTELCollectorEnable {
		if otelLogger := global.GetLoggerProvider().Logger(options.ServiceName); otelLogger != nil {
			record := log.Record{}
			record.SetTimestamp(options.StartTime)
			record.SetObservedTimestamp(options.StartTime.Add(options.Duration))
			record.SetSeverity(mapZapLevelToOTELSeverity(logLevel))
			record.SetBody(log.StringValue(msg))
			record.AddAttributes(
				log.String("code", options.Code.String()),
				log.String("method_type", options.MethodType),
				log.String("full_method", options.FullMethod),
			)
			otelLogger.Emit(options.Context, record)
		}
	}
}

// mapZapLevelToOTELSeverity maps zap log levels to OTEL severity levels
func mapZapLevelToOTELSeverity(logLevel zapcore.Level) log.Severity {
	switch logLevel {
	case zapcore.DebugLevel:
		return log.SeverityDebug
	case zapcore.InfoLevel:
		return log.SeverityInfo
	case zapcore.WarnLevel:
		return log.SeverityWarn
	case zapcore.ErrorLevel:
		return log.SeverityError
	default:
		return log.SeverityInfo
	}
}

// extractMethodName extracts just the method name from a gRPC full method name
func extractMethodName(fullMethod string) string {
	if parts := strings.Split(fullMethod, "/"); len(parts) >= 3 {
		return parts[2]
	}
	return "unknown"
}
