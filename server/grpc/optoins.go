package grpc

import (
	"fmt"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	grpcmiddleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpcrecovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"

	"github.com/instill-ai/x/client"
	"github.com/instill-ai/x/server/grpc/interceptor"
)

// Options contains configuration options for gRPC server setup
type Options struct {
	ServiceName         string
	ServiceVersion      string
	HTTPSConfig         client.HTTPSConfig
	OTELCollectorEnable bool
}

// Option is a function that modifies Options
type Option func(*Options)

// WithServiceConfig sets the service configuration
func WithServiceConfig(config client.HTTPSConfig) Option {
	return func(opts *Options) {
		opts.HTTPSConfig = config
	}
}

// WithOTELCollectorEnable enables or disables the OTEL collector
func WithOTELCollectorEnable(enable bool) Option {
	return func(opts *Options) {
		opts.OTELCollectorEnable = enable
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

// newGRPCOptions creates a new GRPCOptions with default values and applies the given options
func newGRPCOptions(options ...Option) *Options {
	opts := &Options{
		ServiceName:         "unknown",
		ServiceVersion:      "unknown",
		HTTPSConfig:         client.HTTPSConfig{},
		OTELCollectorEnable: false,
	}

	for _, option := range options {
		option(opts)
	}

	return opts
}

// NewGRPCOptionAndCreds creates a new gRPC server options and credentials
func NewGRPCOptionAndCreds(options ...Option) ([]grpc.ServerOption, credentials.TransportCredentials, error) {
	opts := newGRPCOptions(options...)

	grpcServerOpts := []grpc.ServerOption{
		grpc.StreamInterceptor(grpcmiddleware.ChainStreamServer(
			interceptor.StreamAppendMetadataInterceptor,
			interceptor.TracingStreamServerInterceptor(opts.ServiceName, opts.ServiceVersion, opts.OTELCollectorEnable),
			grpcrecovery.StreamServerInterceptor(interceptor.RecoveryInterceptorOpt()),
		)),
		grpc.UnaryInterceptor(grpcmiddleware.ChainUnaryServer(
			interceptor.UnaryAppendMetadataAndErrorCodeInterceptor,
			interceptor.TracingUnaryServerInterceptor(opts.ServiceName, opts.ServiceVersion, opts.OTELCollectorEnable),
			grpcrecovery.UnaryServerInterceptor(interceptor.RecoveryInterceptorOpt()),
		)),
		grpc.StatsHandler(otelgrpc.NewServerHandler(
			otelgrpc.WithTracerProvider(otel.GetTracerProvider()),
			otelgrpc.WithPropagators(otel.GetTextMapPropagator()),
		)),
	}

	// Create tls based credential.
	var creds credentials.TransportCredentials
	var err error
	if opts.HTTPSConfig.Cert != "" && opts.HTTPSConfig.Key != "" {
		creds, err = credentials.NewServerTLSFromFile(opts.HTTPSConfig.Cert, opts.HTTPSConfig.Key)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create credentials: %w", err)
		}
		grpcServerOpts = append(grpcServerOpts, grpc.Creds(creds))
	}

	grpcServerOpts = append(grpcServerOpts, grpc.MaxRecvMsgSize(client.MaxPayloadSize))
	grpcServerOpts = append(grpcServerOpts, grpc.MaxSendMsgSize(client.MaxPayloadSize))
	return grpcServerOpts, creds, nil
}
