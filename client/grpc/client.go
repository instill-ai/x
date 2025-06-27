package grpcclient

import (
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/instill-ai/x/client"

	mgmtpb "github.com/instill-ai/protogen-go/core/mgmt/v1beta"
	pipelinepb "github.com/instill-ai/protogen-go/pipeline/pipeline/v1beta"
)

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

// NewPipelinePublicClient returns an initialized gRPC client for the Pipeline
// public API.
func NewPipelinePublicClient(svc client.ServiceConfig) (pbclient pipelinepb.PipelinePublicServiceClient, closeFn func() error, err error) {
	conn, err := newConn(fmt.Sprintf("%v:%v", svc.Host, svc.PublicPort), svc.HTTPS)
	if err != nil {
		return nil, nil, err
	}

	return pipelinepb.NewPipelinePublicServiceClient(conn), conn.Close, nil
}

// NewMGMTPrivateClient returns an initialized gRPC client for the MGMT private
// API.
func NewMGMTPrivateClient(svc client.ServiceConfig) (pbclient mgmtpb.MgmtPrivateServiceClient, closeFn func() error, err error) {
	conn, err := newConn(fmt.Sprintf("%v:%v", svc.Host, svc.PrivatePort), svc.HTTPS)
	if err != nil {
		return nil, nil, err
	}

	return mgmtpb.NewMgmtPrivateServiceClient(conn), conn.Close, nil
}
