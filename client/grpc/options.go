package grpc

import (
	"github.com/instill-ai/x/client"
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

// Options contains configuration options for gRPC client setup
type Options struct {
	ServiceConfig        client.ServiceConfig
	SetOTELClientHandler bool
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

// newOptions creates a new Options with default values and applies the given options
func newOptions(options ...Option) *Options {
	opts := &Options{
		SetOTELClientHandler: false,
	}

	for _, option := range options {
		if option != nil {
			option(opts)
		}
	}

	return opts
}
