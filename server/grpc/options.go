package grpc

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/stats"

	grpcmiddleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpczap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	grpcrecovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"

	"github.com/instill-ai/x/client"
	"github.com/instill-ai/x/server/grpc/interceptor"

	"time"

	logx "github.com/instill-ai/x/log"
)

// defaultMethodLogExcludePatterns is always included in the methodExcludePatterns used for matching.
var defaultMethodLogExcludePatterns = []string{
	// stop logging gRPC calls if it was a call to liveness or readiness and no error was raised
	".*PublicService/.*ness$",
	// stop logging gRPC calls if it was a call to a private function and no error was raised
	".*PrivateService/.*$",
	// stop logging gRPC calls if it was a call to usage service
	".*UsageService/.*$",
}

// createFilterLogDecider is a closure that creates a filter log decider function that takes patterns as input
func createFilterLogDecider(methodExcludePatterns []string) func(fullMethodName string, err error) bool {
	return func(fullMethodName string, err error) bool {
		if err != nil {
			return true
		}
		allPatterns := append(
			append([]string{}, defaultMethodLogExcludePatterns...),
			methodExcludePatterns...,
		)
		for _, pattern := range allPatterns {
			match, _ := regexp.MatchString(pattern, fullMethodName)
			if match {
				return false
			}
		}
		return true
	}
}

// defaultMethodTraceExcludePatterns contains patterns that are always excluded from tracing
var defaultMethodTraceExcludePatterns = []string{
	// stop tracing gRPC calls if it was a call to liveness or readiness
	".*PublicService/.*ness$",
	// stop tracing gRPC calls if it was a call to a private function
	".*PrivateService/.*$",
	// stop tracing gRPC calls if it was a call to usage service
	".*UsageService/.*$",
}

// createFilterTraceDecider creates a filter function that excludes methods matching the patterns
func createFilterTraceDecider(methodTraceExcludePatterns []string) otelgrpc.Filter {
	allPatterns := append(
		append([]string{}, defaultMethodTraceExcludePatterns...),
		methodTraceExcludePatterns...,
	)
	return func(info *stats.RPCTagInfo) bool {
		for _, pattern := range allPatterns {
			if match, _ := regexp.MatchString(pattern, info.FullMethodName); match {
				return false
			}
		}
		return true
	}
}

// createMessageProducer creates a custom message producer for grpczap
func createMessageProducer(serviceName, serviceVersion string) grpczap.MessageProducer {
	return func(ctx context.Context, msg string, level zapcore.Level, code codes.Code, err error, duration zapcore.Field) {

		// Check if OTEL logger provider is available
		loggerProvider := global.GetLoggerProvider()
		if loggerProvider == nil {
			return
		}

		// Create OTEL logger
		otelLogger := loggerProvider.Logger(fmt.Sprintf("%s:%s", serviceName, serviceVersion))

		// Create log record
		record := log.Record{}
		record.SetTimestamp(time.Now())
		record.SetObservedTimestamp(time.Now())

		// Extract service and method from span name
		var grpcService, grpcMethod string
		if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
			// Get span name from the span
			spanName := span.(interface{ Name() string }).Name()
			if spanName != "" {
				parts := strings.Split(spanName, "/")
				if len(parts) >= 2 {
					grpcService = parts[0]
					grpcMethod = parts[1]
				}
			}
		}

		// Augment message with gRPC service and method
		augmentedMsg := msg
		if grpcService != "" && grpcMethod != "" {
			augmentedMsg = fmt.Sprintf("%s [grpc.service:%s grpc.method:%s]", msg, grpcService, grpcMethod)
		}

		if err != nil {
			augmentedMsg = fmt.Sprintf("%s [error:%s]", msg, err.Error())
		}

		// Set augmented message
		record.SetBody(log.StringValue(augmentedMsg))

		// Set severity based on level
		switch level {
		case zapcore.ErrorLevel, zapcore.FatalLevel:
			record.SetSeverity(log.SeverityError)
		case zapcore.WarnLevel:
			record.SetSeverity(log.SeverityWarn)
		case zapcore.InfoLevel:
			record.SetSeverity(log.SeverityInfo)
		default:
			record.SetSeverity(log.SeverityDebug)
		}

		// Add attributes
		attrs := []log.KeyValue{
			log.String("log.level", level.String()),
			log.String("grpc.code", code.String()),
			log.String("grpc.service", grpcService),
			log.String("grpc.method", grpcMethod),
		}

		if err != nil {
			attrs = append(attrs, log.String("error", err.Error()))
		}

		record.AddAttributes(attrs...)

		// Emit the log record
		otelLogger.Emit(ctx, record)

		// Use the default message producer for regular logging
		grpczap.DefaultMessageProducer(ctx, msg, level, code, err, duration)
	}
}

// Options contains configuration options for gRPC server setup
type Options struct {
	ServiceName                string
	ServiceVersion             string
	HTTPSConfig                client.HTTPSConfig
	MethodLogExcludePatterns   []string
	MethodTraceExcludePatterns []string
	SetOTELServerHandler       bool
	// ServiceMetadata allows injecting metadata into all incoming contexts
	// This is useful for services that need to identify themselves when making
	// requests to other services (e.g., "Instill-Backend": "agent-backend")
	ServiceMetadata map[string]string
}

// Option is a function that modifies Options
type Option func(*Options)

// WithServiceConfig sets the service configuration
func WithServiceConfig(config client.HTTPSConfig) Option {
	return func(opts *Options) {
		opts.HTTPSConfig = config
	}
}

// WithSetOTELServerHandler enables or disables the OTEL collector
func WithSetOTELServerHandler(enable bool) Option {
	return func(opts *Options) {
		opts.SetOTELServerHandler = enable
	}
}

// WithServiceName sets the service name
func WithServiceName(name string) Option {
	return func(opts *Options) {
		opts.ServiceName = name
	}
}

// WithServiceVersion sets the service version
func WithServiceVersion(version string) Option {
	return func(opts *Options) {
		opts.ServiceVersion = version
	}
}

// WithMethodLogExcludePatterns sets the methods to exclude from logging
func WithMethodLogExcludePatterns(patterns []string) Option {
	return func(opts *Options) {
		opts.MethodLogExcludePatterns = patterns
	}
}

// WithMethodTraceExcludePatterns adds patterns for methods to exclude from tracing
func WithMethodTraceExcludePatterns(patterns []string) Option {
	return func(o *Options) {
		o.MethodTraceExcludePatterns = patterns
	}
}

// WithServiceMetadata sets metadata to be injected into all incoming contexts.
// This is useful for services that need to identify themselves when making
// requests to other services (e.g., map["Instill-Backend"] = "agent-backend")
func WithServiceMetadata(metadata map[string]string) Option {
	return func(o *Options) {
		o.ServiceMetadata = metadata
	}
}

// newOptions creates a new Options with default values and applies the given options
func newOptions(options ...Option) *Options {
	opts := &Options{
		ServiceName:                "unknown",
		ServiceVersion:             "unknown",
		HTTPSConfig:                client.HTTPSConfig{},
		MethodLogExcludePatterns:   []string{},
		MethodTraceExcludePatterns: []string{},
		SetOTELServerHandler:       false,
		ServiceMetadata:            nil,
	}

	for _, option := range options {
		option(opts)
	}

	return opts
}

// NewServerOptionsAndCreds creates a new gRPC server options and credentials
func NewServerOptionsAndCreds(options ...Option) ([]grpc.ServerOption, error) {
	opts := newOptions(options...)

	var grpcServerOpts []grpc.ServerOption

	logger, err := logx.GetZapLogger(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get zap logger: %w", err)
	}

	filterLogDecider := createFilterLogDecider(opts.MethodLogExcludePatterns)

	// Create interceptor options conditionally
	var streamInterceptorOpts []grpczap.Option
	var unaryInterceptorOpts []grpczap.Option

	streamInterceptorOpts = append(streamInterceptorOpts, grpczap.WithDecider(filterLogDecider))
	unaryInterceptorOpts = append(unaryInterceptorOpts, grpczap.WithDecider(filterLogDecider))

	// Only add message producer if OTEL is enabled
	if opts.SetOTELServerHandler {
		messageProducer := createMessageProducer(opts.ServiceName, opts.ServiceVersion)
		streamInterceptorOpts = append(streamInterceptorOpts, grpczap.WithMessageProducer(messageProducer))
		unaryInterceptorOpts = append(unaryInterceptorOpts, grpczap.WithMessageProducer(messageProducer))
	}

	// Build interceptor chains with optional service metadata injection
	var unaryChain []grpc.UnaryServerInterceptor
	var streamChain []grpc.StreamServerInterceptor

	// Add service metadata injectors if configured
	if len(opts.ServiceMetadata) > 0 {
		for headerKey, value := range opts.ServiceMetadata {
			unaryChain = append(unaryChain, interceptor.NewUnaryInjectMetadataInterceptor(headerKey, value))
			streamChain = append(streamChain, interceptor.NewStreamInjectMetadataInterceptor(headerKey, value))
		}
	}

	// Add standard interceptors
	unaryChain = append(unaryChain,
		grpczap.UnaryServerInterceptor(logger, unaryInterceptorOpts...),
		interceptor.UnaryAppendMetadataInterceptor,
		grpcrecovery.UnaryServerInterceptor(interceptor.RecoveryInterceptorOpt()),
	)
	streamChain = append(streamChain,
		grpczap.StreamServerInterceptor(logger, streamInterceptorOpts...),
		interceptor.StreamAppendMetadataInterceptor,
		grpcrecovery.StreamServerInterceptor(interceptor.RecoveryInterceptorOpt()),
	)

	grpcServerOpts = append(grpcServerOpts, []grpc.ServerOption{
		grpc.StreamInterceptor(grpcmiddleware.ChainStreamServer(streamChain...)),
		grpc.UnaryInterceptor(grpcmiddleware.ChainUnaryServer(unaryChain...)),
		grpc.MaxRecvMsgSize(client.MaxPayloadSize),
		grpc.MaxSendMsgSize(client.MaxPayloadSize),
	}...)

	// Add OTEL handler with filter
	if opts.SetOTELServerHandler {
		filterTraceDecider := createFilterTraceDecider(opts.MethodTraceExcludePatterns)
		grpcServerOpts = append(grpcServerOpts, grpc.StatsHandler(
			otelgrpc.NewServerHandler(
				otelgrpc.WithFilter(filterTraceDecider),
				otelgrpc.WithSpanOptions(
					trace.WithAttributes(
						attribute.String("service.name", opts.ServiceName),
						attribute.String("service.version", opts.ServiceVersion),
					),
				),
			),
		))
	}

	// Create TLS based credentials.
	if opts.HTTPSConfig.Cert != "" && opts.HTTPSConfig.Key != "" {
		creds, err := credentials.NewServerTLSFromFile(opts.HTTPSConfig.Cert, opts.HTTPSConfig.Key)
		if err != nil {
			return nil, fmt.Errorf("failed to create credentials: %w", err)
		}
		grpcServerOpts = append(grpcServerOpts, grpc.Creds(creds))
	}

	return grpcServerOpts, nil
}
