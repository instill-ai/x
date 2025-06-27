package grpcclient

import (
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/instill-ai/x/client"

	pb "github.com/instill-ai/protogen-go/pipeline/pipeline/v1beta"
)

// NewPipelinePublicClient returns an initialized gRPC client for the Pipeline
// public API.
func NewPipelinePublicClient(svc client.ServiceConfig) (
	pbclient pb.PipelinePublicServiceClient,
	closeFn func() error,
	err error,
) {
	credDialOpt := grpc.WithTransportCredentials(insecure.NewCredentials())
	if svc.HTTPS.Cert != "" && svc.HTTPS.Key != "" {
		creds, err := credentials.NewServerTLSFromFile(svc.HTTPS.Cert, svc.HTTPS.Key)
		if err != nil {
			return nil, nil, fmt.Errorf("creating TLS credentials: %w", err)
		}
		credDialOpt = grpc.WithTransportCredentials(creds)
	}

	conn, err := grpc.NewClient(
		fmt.Sprintf("%v:%v", svc.Host, svc.PublicPort),
		credDialOpt,
		grpc.WithUnaryInterceptor(MetadataPropagatorInterceptor),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(client.MaxPayloadSize),
			grpc.MaxCallSendMsgSize(client.MaxPayloadSize),
		),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("creating client connection: %w", err)
	}

	return pb.NewPipelinePublicServiceClient(conn), conn.Close, nil
}
