package grpc

import (
	"fmt"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/instill-ai/x/client"
	"github.com/instill-ai/x/client/grpc/interceptor"
)

// ClientOptions contains configuration options for gRPC client setup
type ClientOptions struct {
	HTTPSConfig         client.HTTPSConfig
	HostPort            string
	OTELCollectorEnable bool
}

// ClientOption is a function that modifies ClientOptions
type ClientOption func(*ClientOptions)

// WithHTTPSConfig sets the HTTPS configuration for TLS
func WithHTTPSConfig(config client.HTTPSConfig) ClientOption {
	return func(opts *ClientOptions) {
		opts.HTTPSConfig = config
	}
}

// WithHostPort sets the host and port for the connection
func WithHostPort(hostPort string) ClientOption {
	return func(opts *ClientOptions) {
		opts.HostPort = hostPort
	}
}

// WithOTELCollectorEnable sets the OTEL collector enable flag
func WithOTELCollectorEnable(enable bool) ClientOption {
	return func(opts *ClientOptions) {
		opts.OTELCollectorEnable = enable
	}
}

// newClientOptions creates a new ClientOptions with default values and applies the given options
func newClientOptions(options ...ClientOption) *ClientOptions {
	opts := &ClientOptions{
		HTTPSConfig:         client.HTTPSConfig{},
		HostPort:            "",
		OTELCollectorEnable: false,
	}

	for _, option := range options {
		if option != nil {
			option(opts)
		}
	}

	return opts
}

// NewClientDialOptionsAndCreds creates gRPC client dial options and credentials
func NewClientDialOptionsAndCreds(options ...ClientOption) ([]grpc.DialOption, credentials.TransportCredentials, error) {
	opts := newClientOptions(options...)

	dialOpts := []grpc.DialOption{
		grpc.WithUnaryInterceptor(interceptor.MetadataPropagatorInterceptor),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(client.MaxPayloadSize),
			grpc.MaxCallSendMsgSize(client.MaxPayloadSize),
		),
	}

	// Create TLS based credentials
	var creds credentials.TransportCredentials
	var err error
	if opts.HTTPSConfig.Cert != "" && opts.HTTPSConfig.Key != "" {
		creds, err = credentials.NewServerTLSFromFile(opts.HTTPSConfig.Cert, opts.HTTPSConfig.Key)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create TLS credentials: %w", err)
		}
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(creds))
	} else {
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	if opts.OTELCollectorEnable {
		dialOpts = append(dialOpts, grpc.WithStatsHandler(otelgrpc.NewClientHandler(
			otelgrpc.WithTracerProvider(otel.GetTracerProvider()),
			otelgrpc.WithPropagators(otel.GetTextMapPropagator()),
		)))
	}

	return dialOpts, creds, nil
}
