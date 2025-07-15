package grpc

import (
	"crypto/tls"
	"fmt"
	"reflect"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/instill-ai/x/client"

	artifactpb "github.com/instill-ai/protogen-go/artifact/artifact/v1alpha"
	mgmtpb "github.com/instill-ai/protogen-go/core/mgmt/v1beta"
	usagepb "github.com/instill-ai/protogen-go/core/usage/v1beta"
	modelpb "github.com/instill-ai/protogen-go/model/model/v1alpha"
	pipelinepb "github.com/instill-ai/protogen-go/pipeline/pipeline/v1beta"
)

// clientInfo contains information about a client type
type clientInfo struct {
	creator  func(*grpc.ClientConn) any
	isPublic bool
}

// clientRegistry maps client types to their creation functions and metadata
var clientRegistry = map[string]clientInfo{
	"pipelinev1beta.PipelinePublicServiceClient": {
		creator:  func(conn *grpc.ClientConn) any { return pipelinepb.NewPipelinePublicServiceClient(conn) },
		isPublic: true,
	},
	"pipelinev1beta.PipelinePrivateServiceClient": {
		creator:  func(conn *grpc.ClientConn) any { return pipelinepb.NewPipelinePrivateServiceClient(conn) },
		isPublic: false,
	},
	"artifactv1alpha.ArtifactPublicServiceClient": {
		creator:  func(conn *grpc.ClientConn) any { return artifactpb.NewArtifactPublicServiceClient(conn) },
		isPublic: true,
	},
	"artifactv1alpha.ArtifactPrivateServiceClient": {
		creator:  func(conn *grpc.ClientConn) any { return artifactpb.NewArtifactPrivateServiceClient(conn) },
		isPublic: false,
	},
	"modelv1alpha.ModelPublicServiceClient": {
		creator:  func(conn *grpc.ClientConn) any { return modelpb.NewModelPublicServiceClient(conn) },
		isPublic: true,
	},
	"modelv1alpha.ModelPrivateServiceClient": {
		creator:  func(conn *grpc.ClientConn) any { return modelpb.NewModelPrivateServiceClient(conn) },
		isPublic: false,
	},
	"mgmtv1beta.MgmtPublicServiceClient": {
		creator:  func(conn *grpc.ClientConn) any { return mgmtpb.NewMgmtPublicServiceClient(conn) },
		isPublic: true,
	},
	"mgmtv1beta.MgmtPrivateServiceClient": {
		creator:  func(conn *grpc.ClientConn) any { return mgmtpb.NewMgmtPrivateServiceClient(conn) },
		isPublic: false,
	},
	"usagev1beta.UsageServiceClient": {
		creator:  func(conn *grpc.ClientConn) any { return usagepb.NewUsageServiceClient(conn) },
		isPublic: true, // Usage service uses public port
	},
}

// newClientOptions creates a new Options with default values and applies the given options
func newClientOptions(options ...Option) (*Options, error) {
	opts := newOptions(options...)

	// Validate required fields
	if opts.ServiceConfig.Host == "" {
		return nil, fmt.Errorf("service config is required")
	}

	return opts, nil
}

// NewClient creates a gRPC client of the specified service type with proper type safety
func NewClient[T any](options ...Option) (T, func() error, error) {
	var zero T // zero value for type T

	opts, err := newClientOptions(options...)
	if err != nil {
		return zero, nil, err
	}

	// Get client type name using reflection
	clientType := reflect.TypeOf(zero)
	if clientType == nil {
		// Handle interface types by getting the type from a concrete implementation
		clientType = reflect.TypeOf((*T)(nil)).Elem()
	}

	// Get the full type name including package
	typeName := clientType.String()

	// Remove the pointer prefix if present
	if clientType.Kind() == reflect.Ptr {
		typeName = clientType.Elem().String()
	}

	info, exists := clientRegistry[typeName]
	if !exists {
		return zero, nil, fmt.Errorf("unsupported client type: %s", typeName)
	}

	// Determine port based on whether it's public or private
	var port int
	if info.isPublic {
		port = opts.ServiceConfig.PublicPort
	} else {
		port = opts.ServiceConfig.PrivatePort
	}

	// Create connection
	conn, err := newConn(opts.ServiceConfig.Host, port, opts.ServiceConfig.HTTPS, opts.SetOTELClientHandler)
	if err != nil {
		return zero, nil, err
	}

	// Create client
	client := info.creator(conn)

	// Type assertion with safety check
	typedClient, ok := client.(T)
	if !ok {
		if closeErr := conn.Close(); closeErr != nil {
			return zero, nil, fmt.Errorf("failed to close connection: %w, original error: type assertion failed for client type: %s", closeErr, typeName)
		}
		return zero, nil, fmt.Errorf("type assertion failed for client type: %s", typeName)
	}

	return typedClient, conn.Close, nil
}

func newConn(host string, port int, https client.HTTPSConfig, setOTELClientHandler bool) (conn *grpc.ClientConn, err error) {
	dialOpts, err := NewClientOptionsAndCreds(
		WithSetOTELClientHandler(setOTELClientHandler),
	)
	if err != nil {
		return nil, fmt.Errorf("creating dial options: %w", err)
	}

	// Add TLS credentials if HTTPS config is provided
	if https.Cert != "" && https.Key != "" {
		creds, err := credentials.NewServerTLSFromFile(https.Cert, https.Key)
		if err != nil {
			return nil, fmt.Errorf("failed to create TLS credentials: %w", err)
		}
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(creds))
	}

	if host == "usage.instill-ai.com" {
		tlsConfig := &tls.Config{}
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
	}

	conn, err = grpc.NewClient(fmt.Sprintf("%s:%d", host, port), dialOpts...)
	if err != nil {
		return nil, fmt.Errorf("creating client connection: %w", err)
	}

	return conn, nil
}
