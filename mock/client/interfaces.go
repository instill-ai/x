package client

import (
	"context"
	"crypto/tls"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/instill-ai/x/client"
)

// ClientCreator defines the interface for creating gRPC clients
type ClientCreator interface {
	CreateClient(conn *grpc.ClientConn) any
	IsPublic() bool
}

// ConnectionManager defines the interface for managing gRPC connections
type ConnectionManager interface {
	NewConnection(host string, port int, https client.HTTPSConfig, setOTELClientHandler bool) (*grpc.ClientConn, error)
	NewClientOptionsAndCreds(options ...Option) ([]grpc.DialOption, error)
}

// Option represents a client configuration option
type Option interface {
	Apply(*Options)
}

// Options contains configuration options for gRPC client setup
type Options interface {
	GetServiceConfig() client.ServiceConfig
	GetSetOTELClientHandler() bool
}

// ClientFactory defines the interface for creating clients
type ClientFactory interface {
	NewClient(options ...Option) (any, func() error, error)
}

// TLSProvider defines the interface for TLS credential creation
type TLSProvider interface {
	NewServerTLSFromFile(certFile, keyFile string) (credentials.TransportCredentials, error)
	NewTLS(config *tls.Config) credentials.TransportCredentials
}

// MetadataPropagator defines the interface for metadata propagation
type MetadataPropagator interface {
	UnaryMetadataPropagatorInterceptor(
		ctx context.Context,
		method string,
		req, reply any,
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error
	StreamMetadataPropagatorInterceptor(
		ctx context.Context,
		desc *grpc.StreamDesc,
		cc *grpc.ClientConn,
		method string,
		streamer grpc.Streamer,
		opts ...grpc.CallOption,
	) (grpc.ClientStream, error)
}
