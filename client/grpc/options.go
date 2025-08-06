package grpc

import (
	"fmt"
	"regexp"

	"github.com/instill-ai/x/client"
	"github.com/instill-ai/x/client/grpc/interceptor"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/stats"
)

// ClientType represents the type of gRPC client
type ClientType string

const (
	// PipelinePublic is the client type for the public pipeline service.
	PipelinePublic ClientType = "pipeline_public"
	// PipelinePrivate is the client type for the private pipeline service.
	PipelinePrivate ClientType = "pipeline_private"
	// ArtifactPublic is the client type for the public artifact service.
	ArtifactPublic ClientType = "artifact_public"
	// ArtifactPrivate is the client type for the private artifact service.
	ArtifactPrivate ClientType = "artifact_private"
	// ModelPublic is the client type for the public model service.
	ModelPublic ClientType = "model_public"
	// ModelPrivate is the client type for the private model service.
	ModelPrivate ClientType = "model_private"
	// MgmtPublic is the client type for the public mgmt service.
	MgmtPublic ClientType = "mgmt_public"
	// MgmtPrivate is the client type for the private mgmt service.
	MgmtPrivate ClientType = "mgmt_private"
	// Usage is the client type for the usage service.
	Usage ClientType = "usage"
	// External is the client type for the external service.
	External ClientType = "external"
)

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

// Options contains configuration options for gRPC client setup
type Options struct {
	ServiceConfig              client.ServiceConfig
	SetOTELClientHandler       bool
	MethodTraceExcludePatterns []string
	GRPCOptions                []grpc.DialOption
}

// Option is a function that modifies Options
type Option func(*Options)

// WithServiceConfig sets the service configuration
func WithServiceConfig(svc client.ServiceConfig) Option {
	return func(opts *Options) {
		opts.ServiceConfig = svc
	}
}

// WithSetOTELClientHandler sets the OTEL collector enable flag
func WithSetOTELClientHandler(enable bool) Option {
	return func(opts *Options) {
		opts.SetOTELClientHandler = enable
	}
}

// WithMethodTraceExcludePatterns sets the methods to exclude from tracing
func WithMethodTraceExcludePatterns(patterns []string) Option {
	return func(opts *Options) {
		opts.MethodTraceExcludePatterns = patterns
	}
}

// WithGRPCOptions sets the gRPC options
func WithGRPCOptions(options ...grpc.DialOption) Option {
	return func(opts *Options) {
		opts.GRPCOptions = options
	}
}

// newOptions creates a new Options with default values and applies the given options
func newOptions(options ...Option) *Options {
	opts := &Options{
		SetOTELClientHandler:       false,
		MethodTraceExcludePatterns: []string{},
		GRPCOptions:                []grpc.DialOption{},
	}

	for _, option := range options {
		if option != nil {
			option(opts)
		}
	}

	return opts
}

// NewClientOptionsAndCreds creates gRPC client dial options and credentials
func NewClientOptionsAndCreds(options ...Option) ([]grpc.DialOption, error) {
	opts := newOptions(options...)

	var dialOpts []grpc.DialOption

	// Add metadata propagator interceptor (handles both metadata and trace context)
	dialOpts = append(dialOpts, grpc.WithUnaryInterceptor(interceptor.UnaryMetadataPropagatorInterceptor))
	dialOpts = append(dialOpts, grpc.WithStreamInterceptor(interceptor.StreamMetadataPropagatorInterceptor))

	dialOpts = append(dialOpts, grpc.WithDefaultCallOptions(
		grpc.MaxCallRecvMsgSize(client.MaxPayloadSize),
		grpc.MaxCallSendMsgSize(client.MaxPayloadSize),
	))

	// Add OTEL interceptor if enabled
	if opts.SetOTELClientHandler {
		filterTraceDecider := createFilterTraceDecider(opts.MethodTraceExcludePatterns)
		dialOpts = append(dialOpts, grpc.WithStatsHandler(
			otelgrpc.NewClientHandler(
				otelgrpc.WithFilter(filterTraceDecider),
			),
		))
	}

	dialOpts = append(dialOpts, opts.GRPCOptions...)

	// Create TLS based credentials
	var creds credentials.TransportCredentials
	var err error
	if opts.ServiceConfig.HTTPS.Cert != "" && opts.ServiceConfig.HTTPS.Key != "" {
		creds, err = credentials.NewServerTLSFromFile(opts.ServiceConfig.HTTPS.Cert, opts.ServiceConfig.HTTPS.Key)
		if err != nil {
			return nil, fmt.Errorf("failed to create TLS credentials: %w", err)
		}
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(creds))
	} else {
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	return dialOpts, nil
}
