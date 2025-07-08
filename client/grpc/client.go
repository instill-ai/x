package grpcclient

import (
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/instill-ai/x/client"

	artifactpb "github.com/instill-ai/protogen-go/artifact/artifact/v1alpha"
	mgmtpb "github.com/instill-ai/protogen-go/core/mgmt/v1beta"
	modelpb "github.com/instill-ai/protogen-go/model/model/v1alpha"
	pipelinepb "github.com/instill-ai/protogen-go/pipeline/pipeline/v1beta"
)

// NewPipelinePublicClient returns an initialized gRPC client for the Pipeline
// public API.
func NewPipelinePublicClient(svc client.ServiceConfig) (pbclient pipelinepb.PipelinePublicServiceClient, closeFn func() error, err error) {
	conn, err := newConn(fmt.Sprintf("%v:%v", svc.Host, svc.PublicPort), svc.HTTPS)
	if err != nil {
		return nil, nil, err
	}

	return pipelinepb.NewPipelinePublicServiceClient(conn), conn.Close, nil
}

// NewPipelinePrivateClient returns an initialized gRPC client for the Pipeline
// private API.
func NewPipelinePrivateClient(svc client.ServiceConfig) (pbclient pipelinepb.PipelinePrivateServiceClient, closeFn func() error, err error) {
	conn, err := newConn(fmt.Sprintf("%v:%v", svc.Host, svc.PrivatePort), svc.HTTPS)
	if err != nil {
		return nil, nil, err
	}

	return pipelinepb.NewPipelinePrivateServiceClient(conn), conn.Close, nil
}

// NewArtifactPublicClient returns an initialized gRPC client for the Artifact
// public API.
func NewArtifactPublicClient(svc client.ServiceConfig) (pbclient artifactpb.ArtifactPublicServiceClient, closeFn func() error, err error) {
	conn, err := newConn(fmt.Sprintf("%v:%v", svc.Host, svc.PublicPort), svc.HTTPS)
	if err != nil {
		return nil, nil, err
	}

	return artifactpb.NewArtifactPublicServiceClient(conn), conn.Close, nil
}

// NewArtifactPrivateClient returns an initialized gRPC client for the Artifact
// private API.
func NewArtifactPrivateClient(svc client.ServiceConfig) (pbclient artifactpb.ArtifactPrivateServiceClient, closeFn func() error, err error) {
	conn, err := newConn(fmt.Sprintf("%v:%v", svc.Host, svc.PrivatePort), svc.HTTPS)
	if err != nil {
		return nil, nil, err
	}

	return artifactpb.NewArtifactPrivateServiceClient(conn), conn.Close, nil
}

// NewModelPublicClient returns an initialized gRPC client for the Model
// public API.
func NewModelPublicClient(svc client.ServiceConfig) (pbclient modelpb.ModelPublicServiceClient, closeFn func() error, err error) {
	conn, err := newConn(fmt.Sprintf("%v:%v", svc.Host, svc.PublicPort), svc.HTTPS)
	if err != nil {
		return nil, nil, err
	}

	return modelpb.NewModelPublicServiceClient(conn), conn.Close, nil
}

// NewModelPrivateClient returns an initialized gRPC client for the Model
// private API.
func NewModelPrivateClient(svc client.ServiceConfig) (pbclient modelpb.ModelPrivateServiceClient, closeFn func() error, err error) {
	conn, err := newConn(fmt.Sprintf("%v:%v", svc.Host, svc.PrivatePort), svc.HTTPS)
	if err != nil {
		return nil, nil, err
	}

	return modelpb.NewModelPrivateServiceClient(conn), conn.Close, nil
}

// NewMgmtPublicClient returns an initialized gRPC client for the MGMT public
// API.
func NewMgmtPublicClient(svc client.ServiceConfig) (pbclient mgmtpb.MgmtPublicServiceClient, closeFn func() error, err error) {
	conn, err := newConn(fmt.Sprintf("%v:%v", svc.Host, svc.PublicPort), svc.HTTPS)
	if err != nil {
		return nil, nil, err
	}

	return mgmtpb.NewMgmtPublicServiceClient(conn), conn.Close, nil
}

// NewMgmtPrivateClient returns an initialized gRPC client for the MGMT private
// API.
func NewMgmtPrivateClient(svc client.ServiceConfig) (pbclient mgmtpb.MgmtPrivateServiceClient, closeFn func() error, err error) {
	conn, err := newConn(fmt.Sprintf("%v:%v", svc.Host, svc.PrivatePort), svc.HTTPS)
	if err != nil {
		return nil, nil, err
	}

	return mgmtpb.NewMgmtPrivateServiceClient(conn), conn.Close, nil
}

func newConn(hostport string, https client.HTTPSConfig) (conn *grpc.ClientConn, err error) {
	credDialOpt := grpc.WithTransportCredentials(insecure.NewCredentials())
	if https.Cert != "" && https.Key != "" {
		creds, err := credentials.NewServerTLSFromFile(https.Cert, https.Key)
		if err != nil {
			return nil, fmt.Errorf("creating TLS credentials: %w", err)
		}
		credDialOpt = grpc.WithTransportCredentials(creds)
	}

	conn, err = grpc.NewClient(
		hostport,
		credDialOpt,
		grpc.WithUnaryInterceptor(MetadataPropagatorInterceptor),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(client.MaxPayloadSize),
			grpc.MaxCallSendMsgSize(client.MaxPayloadSize),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("creating client connection: %w", err)
	}

	return conn, nil
}
