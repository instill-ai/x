package grpc

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/instill-ai/x/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	artifactpb "github.com/instill-ai/protogen-go/artifact/artifact/v1alpha"
	mgmtpb "github.com/instill-ai/protogen-go/core/mgmt/v1beta"
	modelpb "github.com/instill-ai/protogen-go/model/model/v1alpha"
	pipelinepb "github.com/instill-ai/protogen-go/pipeline/pipeline/v1beta"
)

// ============================================================================
// Mock gRPC Server
// ============================================================================

// MockGRPCServer is a mock gRPC server for testing
type MockGRPCServer struct {
	server *grpc.Server
	lis    net.Listener
	port   int
}

// NewMockGRPCServer creates a new mock gRPC server
func NewMockGRPCServer() (*MockGRPCServer, error) {
	lis, err := net.Listen("tcp", ":0") // Use port 0 to get a random available port
	if err != nil {
		return nil, fmt.Errorf("failed to listen: %w", err)
	}

	port := lis.Addr().(*net.TCPAddr).Port
	server := grpc.NewServer()

	return &MockGRPCServer{
		server: server,
		lis:    lis,
		port:   port,
	}, nil
}

// Start starts the mock server
func (m *MockGRPCServer) Start() error {
	return m.server.Serve(m.lis)
}

// Stop stops the mock server
func (m *MockGRPCServer) Stop() {
	if m.server != nil {
		m.server.Stop()
	}
	if m.lis != nil {
		if closeErr := m.lis.Close(); closeErr != nil {
			fmt.Printf("failed to close listener: %v\n", closeErr)
		}
	}
}

// Port returns the port the server is listening on
func (m *MockGRPCServer) Port() int {
	return m.port
}

// ============================================================================
// Test Utilities
// ============================================================================

// createTestServiceConfig creates a service config for testing
func createTestServiceConfig(host string, publicPort, privatePort int) client.ServiceConfig {
	return client.ServiceConfig{
		Host:        host,
		PublicPort:  publicPort,
		PrivatePort: privatePort,
		HTTPS:       client.HTTPSConfig{},
	}
}

// createTestServiceConfigWithHTTPS creates a service config with HTTPS for testing
func createTestServiceConfigWithHTTPS(host string, publicPort, privatePort int, cert, key string) client.ServiceConfig {
	return client.ServiceConfig{
		Host:        host,
		PublicPort:  publicPort,
		PrivatePort: privatePort,
		HTTPS: client.HTTPSConfig{
			Cert: cert,
			Key:  key,
		},
	}
}

// ============================================================================
// Tests
// ============================================================================

func TestNewClient_PipelinePublic(t *testing.T) {
	// Create mock server
	mockServer, err := NewMockGRPCServer()
	require.NoError(t, err)
	defer mockServer.Stop()

	// Start server in background
	go func() {
		err := mockServer.Start()
		require.NoError(t, err)
	}()

	// Wait a bit for server to start
	time.Sleep(100 * time.Millisecond)

	svc := createTestServiceConfig("localhost", mockServer.Port(), 8081)

	client, closeFn, err := NewClient[pipelinepb.PipelinePublicServiceClient](PipelinePublic, svc)
	require.NoError(t, err)
	assert.NotNil(t, client)
	assert.NotNil(t, closeFn)
	defer func() {
		if closeErr := closeFn(); closeErr != nil {
			fmt.Printf("failed to close client: %v\n", closeErr)
		}
	}()
}

func TestNewClient_PipelinePrivate(t *testing.T) {
	// Create mock server
	mockServer, err := NewMockGRPCServer()
	require.NoError(t, err)
	defer mockServer.Stop()

	// Start server in background
	go func() {
		err := mockServer.Start()
		require.NoError(t, err)
	}()

	// Wait a bit for server to start
	time.Sleep(100 * time.Millisecond)

	svc := createTestServiceConfig("localhost", 8080, mockServer.Port())

	client, closeFn, err := NewClient[pipelinepb.PipelinePrivateServiceClient](PipelinePrivate, svc)
	require.NoError(t, err)
	assert.NotNil(t, client)
	assert.NotNil(t, closeFn)
	defer func() {
		if closeErr := closeFn(); closeErr != nil {
			fmt.Printf("failed to close client: %v\n", closeErr)
		}
	}()
}

func TestNewClient_ArtifactPublic(t *testing.T) {
	// Create mock server
	mockServer, err := NewMockGRPCServer()
	require.NoError(t, err)
	defer mockServer.Stop()

	// Start server in background
	go func() {
		err := mockServer.Start()
		require.NoError(t, err)
	}()

	// Wait a bit for server to start
	time.Sleep(100 * time.Millisecond)

	svc := createTestServiceConfig("localhost", mockServer.Port(), 8081)

	client, closeFn, err := NewClient[artifactpb.ArtifactPublicServiceClient](ArtifactPublic, svc)
	require.NoError(t, err)
	assert.NotNil(t, client)
	assert.NotNil(t, closeFn)
	defer func() {
		if closeErr := closeFn(); closeErr != nil {
			fmt.Printf("failed to close client: %v\n", closeErr)
		}
	}()
}

func TestNewClient_ArtifactPrivate(t *testing.T) {
	// Create mock server
	mockServer, err := NewMockGRPCServer()
	require.NoError(t, err)
	defer mockServer.Stop()

	// Start server in background
	go func() {
		err := mockServer.Start()
		require.NoError(t, err)
	}()

	// Wait a bit for server to start
	time.Sleep(100 * time.Millisecond)

	svc := createTestServiceConfig("localhost", 8080, mockServer.Port())

	client, closeFn, err := NewClient[artifactpb.ArtifactPrivateServiceClient](ArtifactPrivate, svc)
	require.NoError(t, err)
	assert.NotNil(t, client)
	assert.NotNil(t, closeFn)
	defer func() {
		if closeErr := closeFn(); closeErr != nil {
			fmt.Printf("failed to close client: %v\n", closeErr)
		}
	}()
}

func TestNewClient_ModelPublic(t *testing.T) {
	// Create mock server
	mockServer, err := NewMockGRPCServer()
	require.NoError(t, err)
	defer mockServer.Stop()

	// Start server in background
	go func() {
		err := mockServer.Start()
		require.NoError(t, err)
	}()

	// Wait a bit for server to start
	time.Sleep(100 * time.Millisecond)

	svc := createTestServiceConfig("localhost", mockServer.Port(), 8081)

	client, closeFn, err := NewClient[modelpb.ModelPublicServiceClient](ModelPublic, svc)
	require.NoError(t, err)
	assert.NotNil(t, client)
	assert.NotNil(t, closeFn)
	defer func() {
		if closeErr := closeFn(); closeErr != nil {
			fmt.Printf("failed to close client: %v\n", closeErr)
		}
	}()
}

func TestNewClient_ModelPrivate(t *testing.T) {
	// Create mock server
	mockServer, err := NewMockGRPCServer()
	require.NoError(t, err)
	defer mockServer.Stop()

	// Start server in background
	go func() {
		err := mockServer.Start()
		require.NoError(t, err)
	}()

	// Wait a bit for server to start
	time.Sleep(100 * time.Millisecond)

	svc := createTestServiceConfig("localhost", 8080, mockServer.Port())

	client, closeFn, err := NewClient[modelpb.ModelPrivateServiceClient](ModelPrivate, svc)
	require.NoError(t, err)
	assert.NotNil(t, client)
	assert.NotNil(t, closeFn)
	defer func() {
		if closeErr := closeFn(); closeErr != nil {
			fmt.Printf("failed to close client: %v\n", closeErr)
		}
	}()
}

func TestNewClient_MgmtPublic(t *testing.T) {
	// Create mock server
	mockServer, err := NewMockGRPCServer()
	require.NoError(t, err)
	defer mockServer.Stop()

	// Start server in background
	go func() {
		err := mockServer.Start()
		require.NoError(t, err)
	}()

	// Wait a bit for server to start
	time.Sleep(100 * time.Millisecond)

	svc := createTestServiceConfig("localhost", mockServer.Port(), 8081)

	client, closeFn, err := NewClient[mgmtpb.MgmtPublicServiceClient](MgmtPublic, svc)
	require.NoError(t, err)
	assert.NotNil(t, client)
	assert.NotNil(t, closeFn)
	defer func() {
		if closeErr := closeFn(); closeErr != nil {
			fmt.Printf("failed to close client: %v\n", closeErr)
		}
	}()
}

func TestNewClient_MgmtPrivate(t *testing.T) {
	// Create mock server
	mockServer, err := NewMockGRPCServer()
	require.NoError(t, err)
	defer mockServer.Stop()

	// Start server in background
	go func() {
		err := mockServer.Start()
		require.NoError(t, err)
	}()

	// Wait a bit for server to start
	time.Sleep(100 * time.Millisecond)

	svc := createTestServiceConfig("localhost", 8080, mockServer.Port())

	client, closeFn, err := NewClient[mgmtpb.MgmtPrivateServiceClient](MgmtPrivate, svc)
	require.NoError(t, err)
	assert.NotNil(t, client)
	assert.NotNil(t, closeFn)
	defer func() {
		if closeErr := closeFn(); closeErr != nil {
			fmt.Printf("failed to close client: %v\n", closeErr)
		}
	}()
}

func TestNewClient_UnknownClientType(t *testing.T) {
	svc := createTestServiceConfig("localhost", 8080, 8081)

	client, closeFn, err := NewClient[pipelinepb.PipelinePublicServiceClient]("unknown_type", svc)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown client type")
	assert.Nil(t, client)
	assert.Nil(t, closeFn)
}

func TestNewClient_TypeAssertionFailure(t *testing.T) {
	// Create mock server
	mockServer, err := NewMockGRPCServer()
	require.NoError(t, err)
	defer mockServer.Stop()

	// Start server in background
	go func() {
		err := mockServer.Start()
		require.NoError(t, err)
	}()

	// Wait a bit for server to start
	time.Sleep(100 * time.Millisecond)

	svc := createTestServiceConfig("localhost", mockServer.Port(), 8081)

	// Try to create a PipelinePublic client but assert it as a different type
	client, closeFn, err := NewClient[artifactpb.ArtifactPublicServiceClient](PipelinePublic, svc)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "type assertion failed")
	assert.Nil(t, client)
	assert.Nil(t, closeFn)
}

func TestNewClient_WithHTTPS(t *testing.T) {
	svc := createTestServiceConfigWithHTTPS("localhost", 8080, 8081, "/nonexistent/cert.pem", "/nonexistent/key.pem")

	client, closeFn, err := NewClient[pipelinepb.PipelinePublicServiceClient](PipelinePublic, svc)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "TLS credentials")
	assert.Nil(t, client)
	assert.Nil(t, closeFn)
}

func TestNewClient_PortSelection(t *testing.T) {
	// Create mock server for public port
	publicServer, err := NewMockGRPCServer()
	require.NoError(t, err)
	defer publicServer.Stop()

	// Create mock server for private port
	privateServer, err := NewMockGRPCServer()
	require.NoError(t, err)
	defer privateServer.Stop()

	// Start servers in background
	go func() {
		err := publicServer.Start()
		require.NoError(t, err)
	}()
	go func() {
		err := privateServer.Start()
		require.NoError(t, err)
	}()

	// Wait a bit for servers to start
	time.Sleep(100 * time.Millisecond)

	svc := createTestServiceConfig("localhost", publicServer.Port(), privateServer.Port())

	// Test public port selection
	_, _, err = NewClient[pipelinepb.PipelinePublicServiceClient](PipelinePublic, svc)
	require.NoError(t, err)

	// Test private port selection
	_, _, err = NewClient[pipelinepb.PipelinePrivateServiceClient](PipelinePrivate, svc)
	require.NoError(t, err)
}

func TestNewClient_AllClientTypes(t *testing.T) {
	// Create mock servers for public and private ports
	publicServer, err := NewMockGRPCServer()
	require.NoError(t, err)
	defer publicServer.Stop()

	privateServer, err := NewMockGRPCServer()
	require.NoError(t, err)
	defer privateServer.Stop()

	// Start servers in background
	go func() {
		err := publicServer.Start()
		require.NoError(t, err)
	}()
	go func() {
		err := privateServer.Start()
		require.NoError(t, err)
	}()

	// Wait a bit for servers to start
	time.Sleep(100 * time.Millisecond)

	svc := createTestServiceConfig("localhost", publicServer.Port(), privateServer.Port())

	clientTypes := []struct {
		name       string
		clientType ClientType
	}{
		{"PipelinePublic", PipelinePublic},
		{"PipelinePrivate", PipelinePrivate},
		{"ArtifactPublic", ArtifactPublic},
		{"ArtifactPrivate", ArtifactPrivate},
		{"ModelPublic", ModelPublic},
		{"ModelPrivate", ModelPrivate},
		{"MgmtPublic", MgmtPublic},
		{"MgmtPrivate", MgmtPrivate},
	}

	for _, tt := range clientTypes {
		t.Run(tt.name, func(t *testing.T) {
			client, closeFn, err := NewClient[any](tt.clientType, svc)
			require.NoError(t, err)
			assert.NotNil(t, client)
			assert.NotNil(t, closeFn)
			defer func() {
				if closeErr := closeFn(); closeErr != nil {
					fmt.Printf("failed to close client: %v\n", closeErr)
				}
			}()
		})
	}
}

// ============================================================================
// Benchmarks
// ============================================================================

func BenchmarkNewClient_PipelinePublic(b *testing.B) {
	// Create mock server
	mockServer, err := NewMockGRPCServer()
	require.NoError(b, err)
	defer mockServer.Stop()

	// Start server in background
	go func() {
		err := mockServer.Start()
		require.NoError(b, err)
	}()

	// Wait a bit for server to start
	time.Sleep(100 * time.Millisecond)

	svc := createTestServiceConfig("localhost", mockServer.Port(), 8081)

	for i := 0; i < b.N; i++ {
		_, closeFn, err := NewClient[pipelinepb.PipelinePublicServiceClient](PipelinePublic, svc)
		if err != nil {
			b.Fatal(err)
		}
		if closeErr := closeFn(); closeErr != nil {
			b.Fatal(closeErr)
		}
	}
}

func BenchmarkNewClient_AllTypes(b *testing.B) {
	// Create mock servers for public and private ports
	publicServer, err := NewMockGRPCServer()
	require.NoError(b, err)
	defer publicServer.Stop()

	privateServer, err := NewMockGRPCServer()
	require.NoError(b, err)
	defer privateServer.Stop()

	// Start servers in background
	go func() {
		err := publicServer.Start()
		require.NoError(b, err)
	}()
	go func() {
		err := privateServer.Start()
		require.NoError(b, err)
	}()

	// Wait a bit for servers to start
	time.Sleep(100 * time.Millisecond)

	svc := createTestServiceConfig("localhost", publicServer.Port(), privateServer.Port())

	clientTypes := []ClientType{
		PipelinePublic, PipelinePrivate,
		ArtifactPublic, ArtifactPrivate,
		ModelPublic, ModelPrivate,
		MgmtPublic, MgmtPrivate,
	}

	b.ResetTimer()
	for b.Loop() {
		for _, clientType := range clientTypes {
			_, closeFn, err := NewClient[any](clientType, svc)
			if err != nil {
				b.Fatal(err)
			}
			if closeErr := closeFn(); closeErr != nil {
				b.Fatal(closeErr)
			}
		}
	}
}
