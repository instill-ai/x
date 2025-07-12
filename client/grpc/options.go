package grpc

import (
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/instill-ai/x/client"
	"github.com/instill-ai/x/client/grpc/interceptor"
)

// Options contains configuration options for gRPC client setup
type Options struct {
	HTTPSConfig         client.HTTPSConfig
	OTELCollectorEnable bool
}

// Option is a function that modifies Options
type Option func(*Options)

// WithHTTPSConfig sets the HTTPS configuration for TLS
func WithHTTPSConfig(config client.HTTPSConfig) Option {
	return func(opts *Options) {
		opts.HTTPSConfig = config
	}
}

// WithOTELCollectorEnable sets the OTEL collector enable flag
func WithOTELCollectorEnable(enable bool) Option {
	return func(opts *Options) {
		opts.OTELCollectorEnable = enable
	}
}

// newOptions creates a new Options with default values and applies the given options
func newOptions(options ...Option) *Options {
	opts := &Options{
		HTTPSConfig:         client.HTTPSConfig{},
		OTELCollectorEnable: false,
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

	// TODO: Revisit this after all backends are instrumented
	// Add OTEL interceptor if enabled
	// if opts.OTELCollectorEnable {
	// 	dialOpts = append(dialOpts, grpc.WithStatsHandler(otelgrpc.NewClientHandler()))
	// }

	dialOpts = append(dialOpts, grpc.WithDefaultCallOptions(
		grpc.MaxCallRecvMsgSize(client.MaxPayloadSize),
		grpc.MaxCallSendMsgSize(client.MaxPayloadSize),
	))

	// Create TLS based credentials
	var creds credentials.TransportCredentials
	var err error
	if opts.HTTPSConfig.Cert != "" && opts.HTTPSConfig.Key != "" {
		creds, err = credentials.NewServerTLSFromFile(opts.HTTPSConfig.Cert, opts.HTTPSConfig.Key)
		if err != nil {
			return nil, fmt.Errorf("failed to create TLS credentials: %w", err)
		}
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(creds))
	} else {
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	return dialOpts, nil
}
