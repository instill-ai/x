package grpc

import (
	"fmt"

	"google.golang.org/grpc"

	"github.com/instill-ai/x/client"

	artifactpb "github.com/instill-ai/protogen-go/artifact/artifact/v1alpha"
	mgmtpb "github.com/instill-ai/protogen-go/core/mgmt/v1beta"
	modelpb "github.com/instill-ai/protogen-go/model/model/v1alpha"
	pipelinepb "github.com/instill-ai/protogen-go/pipeline/pipeline/v1beta"
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
)

// clientCreator is a function type that creates a specific gRPC client
type clientCreator func(*grpc.ClientConn) any

// clientCreators maps ClientType to their respective client creation functions
var clientCreators = map[ClientType]clientCreator{
	PipelinePublic:  func(conn *grpc.ClientConn) any { return pipelinepb.NewPipelinePublicServiceClient(conn) },
	PipelinePrivate: func(conn *grpc.ClientConn) any { return pipelinepb.NewPipelinePrivateServiceClient(conn) },
	ArtifactPublic:  func(conn *grpc.ClientConn) any { return artifactpb.NewArtifactPublicServiceClient(conn) },
	ArtifactPrivate: func(conn *grpc.ClientConn) any { return artifactpb.NewArtifactPrivateServiceClient(conn) },
	ModelPublic:     func(conn *grpc.ClientConn) any { return modelpb.NewModelPublicServiceClient(conn) },
	ModelPrivate:    func(conn *grpc.ClientConn) any { return modelpb.NewModelPrivateServiceClient(conn) },
	MgmtPublic:      func(conn *grpc.ClientConn) any { return mgmtpb.NewMgmtPublicServiceClient(conn) },
	MgmtPrivate:     func(conn *grpc.ClientConn) any { return mgmtpb.NewMgmtPrivateServiceClient(conn) },
}

// NewClient creates a gRPC client of the specified service type with proper type safety
func NewClient[T any](clientType ClientType, svc client.ServiceConfig) (T, func() error, error) {
	var zero T // zero value for type T

	// Determine port based on client type
	var port int
	switch clientType {
	case PipelinePublic, ArtifactPublic, ModelPublic, MgmtPublic:
		port = svc.PublicPort
	case PipelinePrivate, ArtifactPrivate, ModelPrivate, MgmtPrivate:
		port = svc.PrivatePort
	default:
		return zero, nil, fmt.Errorf("unknown client type: %s", clientType)
	}

	// Create connection
	conn, err := newConn(svc.Host, port, svc.HTTPS)
	if err != nil {
		return zero, nil, err
	}

	// Get client creator function
	creator, exists := clientCreators[clientType]
	if !exists {
		if closeErr := conn.Close(); closeErr != nil {
			return zero, nil, fmt.Errorf("failed to close connection: %w, original error: no creator function for client type: %s", closeErr, clientType)
		}
		return zero, nil, fmt.Errorf("no creator function for client type: %s", clientType)
	}

	// Create client
	client := creator(conn)

	// Type assertion with safety check
	typedClient, ok := client.(T)
	if !ok {
		if closeErr := conn.Close(); closeErr != nil {
			return zero, nil, fmt.Errorf("failed to close connection: %w, original error: type assertion failed for client type: %s", closeErr, clientType)
		}
		return zero, nil, fmt.Errorf("type assertion failed for client type: %s", clientType)
	}

	return typedClient, conn.Close, nil
}

func newConn(host string, port int, https client.HTTPSConfig) (conn *grpc.ClientConn, err error) {

	dialOpts, err := NewClientOptionsAndCreds(
		WithHTTPSConfig(https),
	)
	if err != nil {
		return nil, fmt.Errorf("creating dial options: %w", err)
	}

	conn, err = grpc.NewClient(fmt.Sprintf("%s:%d", host, port), dialOpts...)
	if err != nil {
		return nil, fmt.Errorf("creating client connection: %w", err)
	}

	return conn, nil
}
