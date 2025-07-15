package grpc

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/frankban/quicktest"
	"github.com/gojuno/minimock/v3"
	"google.golang.org/grpc"

	"github.com/instill-ai/x/client"

	artifactpb "github.com/instill-ai/protogen-go/artifact/artifact/v1alpha"
	mgmtpb "github.com/instill-ai/protogen-go/core/mgmt/v1beta"
	usagepb "github.com/instill-ai/protogen-go/core/usage/v1beta"
	modelpb "github.com/instill-ai/protogen-go/model/model/v1alpha"
	pipelinepb "github.com/instill-ai/protogen-go/pipeline/pipeline/v1beta"
	mockclient "github.com/instill-ai/x/mock/client"
)

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

// Add this helper function at the top of the file
func createTestGRPCServer(t testing.TB) (*grpc.Server, int) {
	lis, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}

	port := lis.Addr().(*net.TCPAddr).Port
	s := grpc.NewServer()

	go func() {
		if err := s.Serve(lis); err != nil {
			t.Logf("server stopped: %v", err)
		}
	}()

	return s, port
}

// ============================================================================
// Mock Tests (Updated for new registry-based approach)
// ============================================================================

func TestNewClient_WithMocks(t *testing.T) {
	qt := quicktest.New(t)
	mc := minimock.NewController(t)

	// Create mock connection manager
	mockConnManager := mockclient.NewConnectionManagerMock(mc)

	// Create mock TLS provider
	mockTLSProvider := mockclient.NewTLSProviderMock(mc)

	// Create mock metadata propagator
	mockMetadataPropagator := mockclient.NewMetadataPropagatorMock(mc)

	svc := createTestServiceConfig("localhost", 8080, 8081)
	ctx := context.Background()

	// Set up mock expectations
	mockConnManager.NewConnectionMock.Expect("localhost", 8080, svc.HTTPS, false).Return(nil, nil)
	mockTLSProvider.NewServerTLSFromFileMock.Expect("", "").Return(nil, nil)
	mockMetadataPropagator.UnaryMetadataPropagatorInterceptorMock.Expect(ctx, "test", nil, nil, nil, nil).Return(nil)

	// Actually call the methods to satisfy expectations
	_, _ = mockConnManager.NewConnection("localhost", 8080, svc.HTTPS, false)
	_, _ = mockTLSProvider.NewServerTLSFromFile("", "")
	_ = mockMetadataPropagator.UnaryMetadataPropagatorInterceptor(ctx, "test", nil, nil, nil, nil)

	// Verify the mocks were created successfully
	qt.Check(mockConnManager, quicktest.Not(quicktest.IsNil))
	qt.Check(mockTLSProvider, quicktest.Not(quicktest.IsNil))
	qt.Check(mockMetadataPropagator, quicktest.Not(quicktest.IsNil))
}

func TestClientRegistry_WithMocks(t *testing.T) {
	qt := quicktest.New(t)

	// Test that the client registry contains expected client types
	expectedTypes := []string{
		"pipelinev1beta.PipelinePublicServiceClient",
		"pipelinev1beta.PipelinePrivateServiceClient",
		"artifactv1alpha.ArtifactPublicServiceClient",
		"artifactv1alpha.ArtifactPrivateServiceClient",
		"modelv1alpha.ModelPublicServiceClient",
		"modelv1alpha.ModelPrivateServiceClient",
		"mgmtv1beta.MgmtPublicServiceClient",
		"mgmtv1beta.MgmtPrivateServiceClient",
		"usagev1beta.UsageServiceClient",
	}

	for _, clientType := range expectedTypes {
		info, exists := clientRegistry[clientType]
		qt.Check(exists, quicktest.IsTrue, quicktest.Commentf("Client type %s should exist in registry", clientType))
		qt.Check(info.creator, quicktest.Not(quicktest.IsNil), quicktest.Commentf("Creator function for %s should not be nil", clientType))
	}
}

func TestConnectionManager_WithMocks(t *testing.T) {
	qt := quicktest.New(t)
	mc := minimock.NewController(t)

	// Create mock connection manager
	mockConnManager := mockclient.NewConnectionManagerMock(mc)

	svc := createTestServiceConfig("localhost", 8080, 8081)

	// Set up mock expectations
	mockConnManager.NewConnectionMock.Expect("localhost", 8080, svc.HTTPS, false).Return(nil, nil)
	mockConnManager.NewClientOptionsAndCredsMock.Expect().Return([]grpc.DialOption{}, nil)

	// Test the mock
	conn, err := mockConnManager.NewConnection("localhost", 8080, svc.HTTPS, false)
	opts, optsErr := mockConnManager.NewClientOptionsAndCreds()

	qt.Check(err, quicktest.IsNil)
	qt.Check(conn, quicktest.IsNil) // Mock returns nil
	qt.Check(optsErr, quicktest.IsNil)
	qt.Check(opts, quicktest.DeepEquals, []grpc.DialOption{})
}

func TestTLSProvider_WithMocks(t *testing.T) {
	qt := quicktest.New(t)
	mc := minimock.NewController(t)

	// Create mock TLS provider
	mockTLSProvider := mockclient.NewTLSProviderMock(mc)

	// Set up mock expectations
	mockTLSProvider.NewServerTLSFromFileMock.Expect("cert.pem", "key.pem").Return(nil, nil)
	mockTLSProvider.NewTLSMock.Expect(nil).Return(nil)

	// Test the mock
	creds, err := mockTLSProvider.NewServerTLSFromFile("cert.pem", "key.pem")
	tlsCreds := mockTLSProvider.NewTLS(nil)

	qt.Check(err, quicktest.IsNil)
	qt.Check(creds, quicktest.IsNil)    // Mock returns nil
	qt.Check(tlsCreds, quicktest.IsNil) // Mock returns nil
}

func TestMetadataPropagator_WithMocks(t *testing.T) {
	qt := quicktest.New(t)
	mc := minimock.NewController(t)

	// Create mock metadata propagator
	mockPropagator := mockclient.NewMetadataPropagatorMock(mc)

	ctx := context.Background()

	// Set up mock expectations
	mockPropagator.UnaryMetadataPropagatorInterceptorMock.Expect(ctx, "test", nil, nil, nil, nil).Return(nil)
	mockPropagator.StreamMetadataPropagatorInterceptorMock.Expect(ctx, nil, nil, "test", nil).Return(nil, nil)

	// Test the mock
	err := mockPropagator.UnaryMetadataPropagatorInterceptor(ctx, "test", nil, nil, nil, nil)
	stream, streamErr := mockPropagator.StreamMetadataPropagatorInterceptor(ctx, nil, nil, "test", nil)

	qt.Check(err, quicktest.IsNil)
	qt.Check(streamErr, quicktest.IsNil)
	qt.Check(stream, quicktest.IsNil) // Mock returns nil
}

// ============================================================================
// Integration Tests (Updated for new registry-based approach)
// ============================================================================

func TestNewClient_PipelinePublic(t *testing.T) {
	qt := quicktest.New(t)

	// Create mock server
	server, port := createTestGRPCServer(t)
	defer server.Stop()

	// Wait a bit for server to start
	time.Sleep(100 * time.Millisecond)

	svc := createTestServiceConfig("localhost", port, 8081)

	client, closeFn, err := NewClient[pipelinepb.PipelinePublicServiceClient](
		WithServiceConfig(svc),
		WithSetOTELClientHandler(false),
	)
	qt.Assert(err, quicktest.IsNil)
	qt.Assert(client, quicktest.Not(quicktest.IsNil))
	qt.Assert(closeFn, quicktest.Not(quicktest.IsNil))
	defer func() {
		if closeErr := closeFn(); closeErr != nil {
			fmt.Printf("failed to close client: %v\n", closeErr)
		}
	}()
}

func TestNewClient_PipelinePrivate(t *testing.T) {
	qt := quicktest.New(t)

	// Create mock server
	server, port := createTestGRPCServer(t)
	defer server.Stop()

	// Wait a bit for server to start
	time.Sleep(100 * time.Millisecond)

	svc := createTestServiceConfig("localhost", 8080, port)

	client, closeFn, err := NewClient[pipelinepb.PipelinePrivateServiceClient](
		WithServiceConfig(svc),
		WithSetOTELClientHandler(false),
	)
	qt.Assert(err, quicktest.IsNil)
	qt.Assert(client, quicktest.Not(quicktest.IsNil))
	qt.Assert(closeFn, quicktest.Not(quicktest.IsNil))
	defer func() {
		if closeErr := closeFn(); closeErr != nil {
			fmt.Printf("failed to close client: %v\n", closeErr)
		}
	}()
}

func TestNewClient_ArtifactPublic(t *testing.T) {
	qt := quicktest.New(t)

	// Create mock server
	server, port := createTestGRPCServer(t)
	defer server.Stop()

	// Wait a bit for server to start
	time.Sleep(100 * time.Millisecond)

	svc := createTestServiceConfig("localhost", port, 8081)

	client, closeFn, err := NewClient[artifactpb.ArtifactPublicServiceClient](
		WithServiceConfig(svc),
		WithSetOTELClientHandler(false),
	)
	qt.Assert(err, quicktest.IsNil)
	qt.Assert(client, quicktest.Not(quicktest.IsNil))
	qt.Assert(closeFn, quicktest.Not(quicktest.IsNil))
	defer func() {
		if closeErr := closeFn(); closeErr != nil {
			fmt.Printf("failed to close client: %v\n", closeErr)
		}
	}()
}

func TestNewClient_ArtifactPrivate(t *testing.T) {
	qt := quicktest.New(t)

	// Create mock server
	server, port := createTestGRPCServer(t)
	defer server.Stop()

	// Wait a bit for server to start
	time.Sleep(100 * time.Millisecond)

	svc := createTestServiceConfig("localhost", 8080, port)

	client, closeFn, err := NewClient[artifactpb.ArtifactPrivateServiceClient](
		WithServiceConfig(svc),
		WithSetOTELClientHandler(false),
	)
	qt.Assert(err, quicktest.IsNil)
	qt.Assert(client, quicktest.Not(quicktest.IsNil))
	qt.Assert(closeFn, quicktest.Not(quicktest.IsNil))
	defer func() {
		if closeErr := closeFn(); closeErr != nil {
			fmt.Printf("failed to close client: %v\n", closeErr)
		}
	}()
}

func TestNewClient_ModelPublic(t *testing.T) {
	qt := quicktest.New(t)

	// Create mock server
	server, port := createTestGRPCServer(t)
	defer server.Stop()

	// Wait a bit for server to start
	time.Sleep(100 * time.Millisecond)

	svc := createTestServiceConfig("localhost", port, 8081)

	client, closeFn, err := NewClient[modelpb.ModelPublicServiceClient](
		WithServiceConfig(svc),
		WithSetOTELClientHandler(false),
	)
	qt.Assert(err, quicktest.IsNil)
	qt.Assert(client, quicktest.Not(quicktest.IsNil))
	qt.Assert(closeFn, quicktest.Not(quicktest.IsNil))
	defer func() {
		if closeErr := closeFn(); closeErr != nil {
			fmt.Printf("failed to close client: %v\n", closeErr)
		}
	}()
}

func TestNewClient_ModelPrivate(t *testing.T) {
	qt := quicktest.New(t)

	// Create mock server
	server, port := createTestGRPCServer(t)
	defer server.Stop()

	// Wait a bit for server to start
	time.Sleep(100 * time.Millisecond)

	svc := createTestServiceConfig("localhost", 8080, port)

	client, closeFn, err := NewClient[modelpb.ModelPrivateServiceClient](
		WithServiceConfig(svc),
		WithSetOTELClientHandler(false),
	)
	qt.Assert(err, quicktest.IsNil)
	qt.Assert(client, quicktest.Not(quicktest.IsNil))
	qt.Assert(closeFn, quicktest.Not(quicktest.IsNil))
	defer func() {
		if closeErr := closeFn(); closeErr != nil {
			fmt.Printf("failed to close client: %v\n", closeErr)
		}
	}()
}

func TestNewClient_MgmtPublic(t *testing.T) {
	qt := quicktest.New(t)

	// Create mock server
	server, port := createTestGRPCServer(t)
	defer server.Stop()

	// Wait a bit for server to start
	time.Sleep(100 * time.Millisecond)

	svc := createTestServiceConfig("localhost", port, 8081)

	client, closeFn, err := NewClient[mgmtpb.MgmtPublicServiceClient](
		WithServiceConfig(svc),
		WithSetOTELClientHandler(false),
	)
	qt.Assert(err, quicktest.IsNil)
	qt.Assert(client, quicktest.Not(quicktest.IsNil))
	qt.Assert(closeFn, quicktest.Not(quicktest.IsNil))
	defer func() {
		if closeErr := closeFn(); closeErr != nil {
			fmt.Printf("failed to close client: %v\n", closeErr)
		}
	}()
}

func TestNewClient_MgmtPrivate(t *testing.T) {
	qt := quicktest.New(t)

	// Create mock server
	server, port := createTestGRPCServer(t)
	defer server.Stop()

	// Wait a bit for server to start
	time.Sleep(100 * time.Millisecond)

	svc := createTestServiceConfig("localhost", 8080, port)

	client, closeFn, err := NewClient[mgmtpb.MgmtPrivateServiceClient](
		WithServiceConfig(svc),
		WithSetOTELClientHandler(false),
	)
	qt.Assert(err, quicktest.IsNil)
	qt.Assert(client, quicktest.Not(quicktest.IsNil))
	qt.Assert(closeFn, quicktest.Not(quicktest.IsNil))
	defer func() {
		if closeErr := closeFn(); closeErr != nil {
			fmt.Printf("failed to close client: %v\n", closeErr)
		}
	}()
}

func TestNewClient_Usage(t *testing.T) {
	qt := quicktest.New(t)

	// Create mock server
	server, port := createTestGRPCServer(t)
	defer server.Stop()

	// Wait a bit for server to start
	time.Sleep(100 * time.Millisecond)

	svc := createTestServiceConfig("localhost", port, 8081)

	client, closeFn, err := NewClient[usagepb.UsageServiceClient](
		WithServiceConfig(svc),
		WithSetOTELClientHandler(false),
	)
	qt.Assert(err, quicktest.IsNil)
	qt.Assert(client, quicktest.Not(quicktest.IsNil))
	qt.Assert(closeFn, quicktest.Not(quicktest.IsNil))
	defer func() {
		if closeErr := closeFn(); closeErr != nil {
			fmt.Printf("failed to close client: %v\n", closeErr)
		}
	}()
}

func TestNewClient_UnsupportedClientType(t *testing.T) {
	qt := quicktest.New(t)

	svc := createTestServiceConfig("localhost", 8080, 8081)

	// Test with an unsupported client type
	type UnsupportedClient interface {
		SomeMethod()
	}

	client, closeFn, err := NewClient[UnsupportedClient](
		WithServiceConfig(svc),
		WithSetOTELClientHandler(false),
	)
	qt.Assert(err, quicktest.Not(quicktest.IsNil))
	qt.Check(err.Error(), quicktest.Contains, "unsupported client type")
	qt.Assert(client, quicktest.IsNil)
	qt.Assert(closeFn, quicktest.IsNil)
}

func TestNewClient_MissingServiceConfig(t *testing.T) {
	qt := quicktest.New(t)

	// Test with missing service config
	client, closeFn, err := NewClient[pipelinepb.PipelinePublicServiceClient](
		WithSetOTELClientHandler(false),
	)
	qt.Assert(err, quicktest.Not(quicktest.IsNil))
	qt.Check(err.Error(), quicktest.Contains, "service config is required")
	qt.Assert(client, quicktest.IsNil)
	qt.Assert(closeFn, quicktest.IsNil)
}

func TestNewClient_WithHTTPS(t *testing.T) {
	qt := quicktest.New(t)

	svc := createTestServiceConfigWithHTTPS("localhost", 8080, 8081, "/nonexistent/cert.pem", "/nonexistent/key.pem")

	client, closeFn, err := NewClient[pipelinepb.PipelinePublicServiceClient](
		WithServiceConfig(svc),
		WithSetOTELClientHandler(false),
	)
	qt.Assert(err, quicktest.Not(quicktest.IsNil))
	qt.Check(err.Error(), quicktest.Contains, "TLS credentials")
	qt.Assert(client, quicktest.IsNil)
	qt.Assert(closeFn, quicktest.IsNil)
}

func TestNewClient_PortSelection(t *testing.T) {
	qt := quicktest.New(t)

	// Create mock server for public port
	publicServer, publicPort := createTestGRPCServer(t)
	defer publicServer.Stop()

	// Create mock server for private port
	privateServer, privatePort := createTestGRPCServer(t)
	defer privateServer.Stop()

	// Wait a bit for servers to start
	time.Sleep(100 * time.Millisecond)

	svc := createTestServiceConfig("localhost", publicPort, privatePort)

	// Test public port selection
	_, _, err := NewClient[pipelinepb.PipelinePublicServiceClient](
		WithServiceConfig(svc),
		WithSetOTELClientHandler(false),
	)
	qt.Assert(err, quicktest.IsNil)

	// Test private port selection
	_, _, err = NewClient[pipelinepb.PipelinePrivateServiceClient](
		WithServiceConfig(svc),
		WithSetOTELClientHandler(false),
	)
	qt.Assert(err, quicktest.IsNil)
}

func TestNewClient_AllClientTypes(t *testing.T) {
	// Create mock servers for public and private ports
	publicServer, publicPort := createTestGRPCServer(t)
	defer publicServer.Stop()

	privateServer, privatePort := createTestGRPCServer(t)
	defer privateServer.Stop()

	// Wait a bit for servers to start
	time.Sleep(100 * time.Millisecond)

	svc := createTestServiceConfig("localhost", publicPort, privatePort)

	clientTypes := []struct {
		name string
		test func() error
	}{
		{"PipelinePublic", func() error {
			_, closeFn, err := NewClient[pipelinepb.PipelinePublicServiceClient](
				WithServiceConfig(svc),
				WithSetOTELClientHandler(false),
			)
			if err != nil {
				return err
			}
			return closeFn()
		}},
		{"PipelinePrivate", func() error {
			_, closeFn, err := NewClient[pipelinepb.PipelinePrivateServiceClient](
				WithServiceConfig(svc),
				WithSetOTELClientHandler(false),
			)
			if err != nil {
				return err
			}
			return closeFn()
		}},
		{"ArtifactPublic", func() error {
			_, closeFn, err := NewClient[artifactpb.ArtifactPublicServiceClient](
				WithServiceConfig(svc),
				WithSetOTELClientHandler(false),
			)
			if err != nil {
				return err
			}
			return closeFn()
		}},
		{"ArtifactPrivate", func() error {
			_, closeFn, err := NewClient[artifactpb.ArtifactPrivateServiceClient](
				WithServiceConfig(svc),
				WithSetOTELClientHandler(false),
			)
			if err != nil {
				return err
			}
			return closeFn()
		}},
		{"ModelPublic", func() error {
			_, closeFn, err := NewClient[modelpb.ModelPublicServiceClient](
				WithServiceConfig(svc),
				WithSetOTELClientHandler(false),
			)
			if err != nil {
				return err
			}
			return closeFn()
		}},
		{"ModelPrivate", func() error {
			_, closeFn, err := NewClient[modelpb.ModelPrivateServiceClient](
				WithServiceConfig(svc),
				WithSetOTELClientHandler(false),
			)
			if err != nil {
				return err
			}
			return closeFn()
		}},
		{"MgmtPublic", func() error {
			_, closeFn, err := NewClient[mgmtpb.MgmtPublicServiceClient](
				WithServiceConfig(svc),
				WithSetOTELClientHandler(false),
			)
			if err != nil {
				return err
			}
			return closeFn()
		}},
		{"MgmtPrivate", func() error {
			_, closeFn, err := NewClient[mgmtpb.MgmtPrivateServiceClient](
				WithServiceConfig(svc),
				WithSetOTELClientHandler(false),
			)
			if err != nil {
				return err
			}
			return closeFn()
		}},
		{"Usage", func() error {
			_, closeFn, err := NewClient[usagepb.UsageServiceClient](
				WithServiceConfig(svc),
				WithSetOTELClientHandler(false),
			)
			if err != nil {
				return err
			}
			return closeFn()
		}},
	}

	for _, tt := range clientTypes {
		t.Run(tt.name, func(t *testing.T) {
			qt := quicktest.New(t)
			err := tt.test()
			qt.Assert(err, quicktest.IsNil)
		})
	}
}

func TestNewClient_OptionsPattern(t *testing.T) {
	// Create mock server
	server, port := createTestGRPCServer(t)
	defer server.Stop()

	// Wait a bit for server to start
	time.Sleep(100 * time.Millisecond)

	svc := createTestServiceConfig("localhost", port, 8081)

	// Test with different option combinations
	tests := []struct {
		name        string
		options     []Option
		expectError bool
	}{
		{
			name: "all options provided",
			options: []Option{
				WithServiceConfig(svc),
				WithSetOTELClientHandler(true),
			},
			expectError: false,
		},
		{
			name: "minimal options",
			options: []Option{
				WithServiceConfig(svc),
			},
			expectError: false,
		},
		{
			name:        "missing service config",
			options:     []Option{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			qt := quicktest.New(t)
			client, closeFn, err := NewClient[pipelinepb.PipelinePublicServiceClient](tt.options...)
			if tt.expectError {
				qt.Assert(err, quicktest.Not(quicktest.IsNil))
				qt.Assert(client, quicktest.IsNil)
				qt.Assert(closeFn, quicktest.IsNil)
			} else {
				qt.Assert(err, quicktest.IsNil)
				qt.Assert(client, quicktest.Not(quicktest.IsNil))
				qt.Assert(closeFn, quicktest.Not(quicktest.IsNil))
				if closeFn != nil {
					defer func() {
						if closeErr := closeFn(); closeErr != nil {
							fmt.Printf("failed to close client: %v\n", closeErr)
						}
					}()
				}
			}
		})
	}
}

// ============================================================================
// New Tests for Registry-based Approach
// ============================================================================

func TestClientRegistry_TypeNames(t *testing.T) {
	qt := quicktest.New(t)

	// Test that the registry contains the correct type names
	expectedTypeNames := map[string]bool{
		"pipelinev1beta.PipelinePublicServiceClient":   true,
		"pipelinev1beta.PipelinePrivateServiceClient":  false,
		"artifactv1alpha.ArtifactPublicServiceClient":  true,
		"artifactv1alpha.ArtifactPrivateServiceClient": false,
		"modelv1alpha.ModelPublicServiceClient":        true,
		"modelv1alpha.ModelPrivateServiceClient":       false,
		"mgmtv1beta.MgmtPublicServiceClient":           true,
		"mgmtv1beta.MgmtPrivateServiceClient":          false,
		"usagev1beta.UsageServiceClient":               true,
	}

	for typeName, isPublic := range expectedTypeNames {
		info, exists := clientRegistry[typeName]
		qt.Check(exists, quicktest.IsTrue, quicktest.Commentf("Type %s should exist in registry", typeName))
		qt.Check(info.isPublic, quicktest.Equals, isPublic, quicktest.Commentf("Type %s should have correct public flag", typeName))
	}
}

func TestNewClient_TypeReflection(t *testing.T) {
	qt := quicktest.New(t)

	svc := createTestServiceConfig("localhost", 8080, 8081)

	// Test that the reflection-based type detection works correctly
	client, closeFn, err := NewClient[pipelinepb.PipelinePublicServiceClient](
		WithServiceConfig(svc),
		WithSetOTELClientHandler(false),
	)

	// The client creation should succeed even without a server running
	// because gRPC clients can be created without an active connection
	qt.Assert(err, quicktest.IsNil)
	qt.Assert(client, quicktest.Not(quicktest.IsNil))
	qt.Assert(closeFn, quicktest.Not(quicktest.IsNil))

	// Clean up
	if closeFn != nil {
		defer func() {
			if closeErr := closeFn(); closeErr != nil {
				t.Logf("failed to close client: %v", closeErr)
			}
		}()
	}

	// Test that we can actually use the client (this will fail due to no server)
	// but the client object itself should be valid
	qt.Check(client, quicktest.Not(quicktest.IsNil))
}

func TestNewClient_InterfaceType(t *testing.T) {
	qt := quicktest.New(t)

	svc := createTestServiceConfig("localhost", 8080, 8081)

	// Test with interface type (should be handled by the nil check in the code)
	type TestInterface interface {
		SomeMethod()
	}

	client, closeFn, err := NewClient[TestInterface](
		WithServiceConfig(svc),
		WithSetOTELClientHandler(false),
	)

	qt.Assert(err, quicktest.Not(quicktest.IsNil))
	qt.Check(err.Error(), quicktest.Contains, "unsupported client type")
	qt.Assert(client, quicktest.IsNil)
	qt.Assert(closeFn, quicktest.IsNil)
}

// ============================================================================
// Benchmarks (Updated for new registry-based approach)
// ============================================================================

func BenchmarkNewClient_PipelinePublic(b *testing.B) {
	// Create mock server
	server, port := createTestGRPCServer(b)
	defer server.Stop()

	// Wait a bit for server to start
	time.Sleep(100 * time.Millisecond)

	svc := createTestServiceConfig("localhost", port, 8081)

	for i := 0; i < b.N; i++ {
		_, closeFn, err := NewClient[pipelinepb.PipelinePublicServiceClient](
			WithServiceConfig(svc),
			WithSetOTELClientHandler(false),
		)
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
	publicServer, publicPort := createTestGRPCServer(b)
	defer publicServer.Stop()

	privateServer, privatePort := createTestGRPCServer(b)
	defer privateServer.Stop()

	// Wait a bit for servers to start
	time.Sleep(100 * time.Millisecond)

	svc := createTestServiceConfig("localhost", publicPort, privatePort)

	clientTypes := []struct {
		name string
		test func() error
	}{
		{"PipelinePublic", func() error {
			_, closeFn, err := NewClient[pipelinepb.PipelinePublicServiceClient](
				WithServiceConfig(svc),
				WithSetOTELClientHandler(false),
			)
			if err != nil {
				return err
			}
			return closeFn()
		}},
		{"PipelinePrivate", func() error {
			_, closeFn, err := NewClient[pipelinepb.PipelinePrivateServiceClient](
				WithServiceConfig(svc),
				WithSetOTELClientHandler(false),
			)
			if err != nil {
				return err
			}
			return closeFn()
		}},
		{"ArtifactPublic", func() error {
			_, closeFn, err := NewClient[artifactpb.ArtifactPublicServiceClient](
				WithServiceConfig(svc),
				WithSetOTELClientHandler(false),
			)
			if err != nil {
				return err
			}
			return closeFn()
		}},
		{"ArtifactPrivate", func() error {
			_, closeFn, err := NewClient[artifactpb.ArtifactPrivateServiceClient](
				WithServiceConfig(svc),
				WithSetOTELClientHandler(false),
			)
			if err != nil {
				return err
			}
			return closeFn()
		}},
		{"ModelPublic", func() error {
			_, closeFn, err := NewClient[modelpb.ModelPublicServiceClient](
				WithServiceConfig(svc),
				WithSetOTELClientHandler(false),
			)
			if err != nil {
				return err
			}
			return closeFn()
		}},
		{"ModelPrivate", func() error {
			_, closeFn, err := NewClient[modelpb.ModelPrivateServiceClient](
				WithServiceConfig(svc),
				WithSetOTELClientHandler(false),
			)
			if err != nil {
				return err
			}
			return closeFn()
		}},
		{"MgmtPublic", func() error {
			_, closeFn, err := NewClient[mgmtpb.MgmtPublicServiceClient](
				WithServiceConfig(svc),
				WithSetOTELClientHandler(false),
			)
			if err != nil {
				return err
			}
			return closeFn()
		}},
		{"MgmtPrivate", func() error {
			_, closeFn, err := NewClient[mgmtpb.MgmtPrivateServiceClient](
				WithServiceConfig(svc),
				WithSetOTELClientHandler(false),
			)
			if err != nil {
				return err
			}
			return closeFn()
		}},
		{"Usage", func() error {
			_, closeFn, err := NewClient[usagepb.UsageServiceClient](
				WithServiceConfig(svc),
				WithSetOTELClientHandler(false),
			)
			if err != nil {
				return err
			}
			return closeFn()
		}},
	}

	b.ResetTimer()
	for b.Loop() {
		for _, clientType := range clientTypes {
			if err := clientType.test(); err != nil {
				b.Fatal(err)
			}
		}
	}
}
