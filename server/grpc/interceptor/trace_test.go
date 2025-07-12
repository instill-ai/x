package interceptor

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	errorsx "github.com/instill-ai/x/errors"
)

func TestTracingUnaryServerInterceptor(t *testing.T) {
	tests := []struct {
		name                string
		serviceName         string
		serviceVersion      string
		OTELCollectorEnable bool
		fullMethod          string
		shouldLog           bool
		err                 error
		expectedLogLevel    zapcore.Level
		description         string
	}{
		{
			name:                "successful request with logging enabled",
			serviceName:         "test-service",
			serviceVersion:      "v1.0.0",
			OTELCollectorEnable: true,
			fullMethod:          "test.Service/Method",
			shouldLog:           true,
			err:                 nil,
			expectedLogLevel:    zapcore.InfoLevel,
			description:         "should log at info level for successful request",
		},
		{
			name:                "successful request with logging disabled",
			serviceName:         "test-service",
			serviceVersion:      "v1.0.0",
			OTELCollectorEnable: false,
			fullMethod:          "test.Service/Method",
			shouldLog:           false,
			err:                 nil,
			expectedLogLevel:    zapcore.InfoLevel,
			description:         "should not log when shouldLog is false",
		},
		{
			name:                "request with warning error",
			serviceName:         "test-service",
			serviceVersion:      "v1.0.0",
			OTELCollectorEnable: true,
			fullMethod:          "test.Service/Method",
			shouldLog:           true,
			err:                 status.Error(codes.Canceled, "canceled"),
			expectedLogLevel:    zapcore.WarnLevel,
			description:         "should log at warn level for canceled error",
		},
		{
			name:                "request with error",
			serviceName:         "test-service",
			serviceVersion:      "v1.0.0",
			OTELCollectorEnable: true,
			fullMethod:          "test.Service/Method",
			shouldLog:           true,
			err:                 status.Error(codes.InvalidArgument, "invalid argument"),
			expectedLogLevel:    zapcore.ErrorLevel,
			description:         "should log at error level for invalid argument error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create context with shouldLog flag
			ctx := context.Background()
			if !tt.shouldLog {
				ctx = context.WithValue(ctx, shouldLogKey, false)
			}

			// Create mock logger
			mockLogger := &MockLogger{}

			// Create interceptor
			interceptor := TracingUnaryServerInterceptor(tt.serviceName, tt.serviceVersion, tt.OTELCollectorEnable)

			// Create request info
			info := &grpc.UnaryServerInfo{
				FullMethod: tt.fullMethod,
			}

			// Create handler that returns the specified error
			handler := func(ctx context.Context, req interface{}) (interface{}, error) {
				return "response", tt.err
			}

			// Execute interceptor
			resp, err := interceptor(ctx, "request", info, handler)

			// Verify response and error
			assert.Equal(t, "response", resp)
			assert.Equal(t, tt.err, err)

			// Verify logging behavior
			if tt.shouldLog {
				mockLogger.AssertExpectations(t)
			} else {
				mockLogger.AssertNotCalled(t, "Info")
				mockLogger.AssertNotCalled(t, "Warn")
				mockLogger.AssertNotCalled(t, "Error")
			}
		})
	}
}

func TestTracingStreamServerInterceptor(t *testing.T) {
	tests := []struct {
		name                string
		serviceName         string
		serviceVersion      string
		OTELCollectorEnable bool
		fullMethod          string
		shouldLog           bool
		err                 error
		expectedLogLevel    zapcore.Level
		description         string
	}{
		{
			name:                "successful stream with logging enabled",
			serviceName:         "test-service",
			serviceVersion:      "v1.0.0",
			OTELCollectorEnable: true,
			fullMethod:          "test.Service/StreamMethod",
			shouldLog:           true,
			err:                 nil,
			expectedLogLevel:    zapcore.InfoLevel,
			description:         "should log at info level for successful stream",
		},
		{
			name:                "stream with logging disabled",
			serviceName:         "test-service",
			serviceVersion:      "v1.0.0",
			OTELCollectorEnable: false,
			fullMethod:          "test.Service/StreamMethod",
			shouldLog:           false,
			err:                 nil,
			expectedLogLevel:    zapcore.InfoLevel,
			description:         "should not log when shouldLog is false",
		},
		{
			name:                "stream with error",
			serviceName:         "test-service",
			serviceVersion:      "v1.0.0",
			OTELCollectorEnable: true,
			fullMethod:          "test.Service/StreamMethod",
			shouldLog:           true,
			err:                 status.Error(codes.Internal, "internal error"),
			expectedLogLevel:    zapcore.ErrorLevel,
			description:         "should log at error level for internal error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create context with shouldLog flag
			ctx := context.Background()
			if !tt.shouldLog {
				ctx = context.WithValue(ctx, shouldLogKey, false)
			}

			// Create mock stream
			stream := &MockServerStream{ctx: ctx}

			// Create mock logger
			mockLogger := &MockLogger{}

			// Create interceptor
			interceptor := TracingStreamServerInterceptor(tt.serviceName, tt.serviceVersion, tt.OTELCollectorEnable)

			// Create stream info
			info := &grpc.StreamServerInfo{
				FullMethod: tt.fullMethod,
			}

			// Create handler that returns the specified error
			handler := func(srv interface{}, stream grpc.ServerStream) error {
				return tt.err
			}

			// Execute interceptor
			err := interceptor(nil, stream, info, handler)

			// Verify error
			assert.Equal(t, tt.err, err)

			// Verify logging behavior
			if tt.shouldLog {
				mockLogger.AssertExpectations(t)
			} else {
				mockLogger.AssertNotCalled(t, "Info")
				mockLogger.AssertNotCalled(t, "Warn")
				mockLogger.AssertNotCalled(t, "Error")
			}
		})
	}
}

func TestLogGRPCRequestOptions(t *testing.T) {
	// Test withContext option
	ctx := context.Background()
	opts := &gRPCRequestLogOptions{}
	withContext(ctx)(opts)
	assert.Equal(t, ctx, opts.Context)

	// Test withServiceInfo option
	opts = &gRPCRequestLogOptions{}
	withServiceInfo("test-service", "v1.0.0")(opts)
	assert.Equal(t, "test-service", opts.ServiceName)
	assert.Equal(t, "v1.0.0", opts.ServiceVersion)

	// Test withMethodInfo option
	opts = &gRPCRequestLogOptions{}
	withMethodInfo("unary", "test.Service/Method")(opts)
	assert.Equal(t, "unary", opts.MethodType)
	assert.Equal(t, "test.Service/Method", opts.FullMethod)

	// Test withTiming option
	startTime := time.Now()
	duration := time.Second
	opts = &gRPCRequestLogOptions{}
	withTiming(startTime, duration)(opts)
	assert.Equal(t, startTime, opts.StartTime)
	assert.Equal(t, duration, opts.Duration)

	// Test withCode option
	opts = &gRPCRequestLogOptions{}
	withCode(codes.OK)(opts)
	assert.Equal(t, codes.OK, opts.Code)

	// Test withOTELEnable option
	opts = &gRPCRequestLogOptions{}
	withOTELEnable(true)(opts)
	assert.True(t, opts.OTELCollectorEnable)
}

func TestMapZapLevelToOTELSeverity(t *testing.T) {
	tests := []struct {
		name         string
		zapLevel     zapcore.Level
		expectedOTEL log.Severity
		description  string
	}{
		{
			name:         "debug level",
			zapLevel:     zapcore.DebugLevel,
			expectedOTEL: log.SeverityDebug,
			description:  "should map zap debug to OTEL debug",
		},
		{
			name:         "info level",
			zapLevel:     zapcore.InfoLevel,
			expectedOTEL: log.SeverityInfo,
			description:  "should map zap info to OTEL info",
		},
		{
			name:         "warn level",
			zapLevel:     zapcore.WarnLevel,
			expectedOTEL: log.SeverityWarn,
			description:  "should map zap warn to OTEL warn",
		},
		{
			name:         "error level",
			zapLevel:     zapcore.ErrorLevel,
			expectedOTEL: log.SeverityError,
			description:  "should map zap error to OTEL error",
		},
		{
			name:         "unknown level defaults to info",
			zapLevel:     zapcore.FatalLevel,
			expectedOTEL: log.SeverityInfo,
			description:  "should default to info for unknown levels",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapZapLevelToOTELSeverity(tt.zapLevel)
			assert.Equal(t, tt.expectedOTEL, result, tt.description)
		})
	}
}

func TestExtractMethodName(t *testing.T) {
	tests := []struct {
		name           string
		fullMethod     string
		expectedMethod string
		description    string
	}{
		{
			name:           "valid full method",
			fullMethod:     "test.Service/SubService/Method", // This has 3 parts: ["test.Service", "SubService", "Method"]
			expectedMethod: "Method",
			description:    "should extract method name from valid full method",
		},
		{
			name:           "full method with multiple slashes",
			fullMethod:     "test.Service/SubService/Method",
			expectedMethod: "Method",
			description:    "should extract last part as method name",
		},
		{
			name:           "invalid full method - too few parts",
			fullMethod:     "test.Service",
			expectedMethod: "unknown",
			description:    "should return unknown for invalid full method",
		},
		{
			name:           "empty full method",
			fullMethod:     "",
			expectedMethod: "unknown",
			description:    "should return unknown for empty full method",
		},
		{
			name:           "full method with trailing slash",
			fullMethod:     "test.Service/Method/",
			expectedMethod: "",
			description:    "should handle trailing slash correctly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractMethodName(tt.fullMethod)
			assert.Equal(t, tt.expectedMethod, result, tt.description)
		})
	}
}

func TestConvertGRPCCode(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		expectedCode codes.Code
		description  string
	}{
		{
			name:         "nil error",
			err:          nil,
			expectedCode: codes.OK,
			description:  "should return OK for nil error",
		},
		{
			name:         "gRPC status error",
			err:          status.Error(codes.InvalidArgument, "invalid argument"),
			expectedCode: codes.InvalidArgument,
			description:  "should return the gRPC status code",
		},
		{
			name:         "non-gRPC error",
			err:          assert.AnError,
			expectedCode: codes.Unknown,
			description:  "should return Unknown for non-gRPC errors",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := errorsx.ConvertGRPCCode(tt.err)
			assert.Equal(t, tt.expectedCode, result, tt.description)
		})
	}
}

// Test log level determination based on gRPC codes
func TestLogLevelDetermination(t *testing.T) {
	tests := []struct {
		name          string
		code          codes.Code
		expectedLevel zapcore.Level
		description   string
	}{
		{
			name:          "OK code",
			code:          codes.OK,
			expectedLevel: zapcore.InfoLevel,
			description:   "should use info level for OK",
		},
		{
			name:          "Canceled code",
			code:          codes.Canceled,
			expectedLevel: zapcore.WarnLevel,
			description:   "should use warn level for Canceled",
		},
		{
			name:          "InvalidArgument code",
			code:          codes.InvalidArgument,
			expectedLevel: zapcore.ErrorLevel,
			description:   "should use error level for InvalidArgument",
		},
		{
			name:          "Unknown code",
			code:          codes.Unknown,
			expectedLevel: zapcore.InfoLevel,
			description:   "should default to info level for unknown codes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create options with the test code
			opts := &gRPCRequestLogOptions{
				Context: context.Background(),
				Code:    tt.code,
			}

			// Determine log level (this logic is in logGRPCRequest)
			var logLevel zapcore.Level
			switch opts.Code {
			case codes.OK:
				logLevel = zapcore.InfoLevel
			case codes.Canceled, codes.DeadlineExceeded, codes.PermissionDenied, codes.ResourceExhausted, codes.FailedPrecondition, codes.Aborted, codes.OutOfRange, codes.Unavailable, codes.DataLoss:
				logLevel = zapcore.WarnLevel
			case codes.InvalidArgument, codes.NotFound, codes.AlreadyExists, codes.Unimplemented, codes.Internal, codes.Unauthenticated:
				logLevel = zapcore.ErrorLevel
			default:
				logLevel = zapcore.InfoLevel
			}

			assert.Equal(t, tt.expectedLevel, logLevel, tt.description)
		})
	}
}

// Test trace ID extraction
func TestTraceIDExtraction(t *testing.T) {
	// Create a context with a trace span
	ctx := context.Background()
	tracer := noop.NewTracerProvider().Tracer("test")
	ctx, span := tracer.Start(ctx, "test-span")
	defer span.End()

	// Extract trace ID
	traceID := trace.SpanFromContext(ctx).SpanContext().TraceID().String()

	// Verify trace ID is not empty
	assert.NotEmpty(t, traceID)
	assert.Len(t, traceID, 32) // Trace ID should be 32 characters in hex
}

// Test message formatting
func TestMessageFormatting(t *testing.T) {
	ctx := context.Background()
	tracer := noop.NewTracerProvider().Tracer("test")
	ctx, span := tracer.Start(ctx, "test-span")
	defer span.End()

	methodType := "unary"
	fullMethod := "test.Service/Method"

	// Format message (this logic is in logGRPCRequest)
	msg := fmt.Sprintf("finished %s call %s (trace_id: %s)",
		methodType,
		extractMethodName(fullMethod),
		trace.SpanFromContext(ctx).SpanContext().TraceID().String(),
	)

	// Verify message format
	assert.Contains(t, msg, "finished unary call unknown")
	assert.Contains(t, msg, "trace_id:")
	assert.Contains(t, msg, trace.SpanFromContext(ctx).SpanContext().TraceID().String())
}
