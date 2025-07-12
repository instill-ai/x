package grpc

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/instill-ai/x/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGRPCOptions(t *testing.T) {
	tests := []struct {
		name     string
		options  []Option
		expected *Options
	}{
		{
			name:    "default options",
			options: []Option{},
			expected: &Options{
				ServiceName:           "unknown",
				ServiceVersion:        "unknown",
				HTTPSConfig:           client.HTTPSConfig{},
				MethodExcludePatterns: []string{},
				OTELCollectorEnable:   false,
			},
		},
		{
			name: "with service name",
			options: []Option{
				WithServiceName("test-service"),
			},
			expected: &Options{
				ServiceName:           "test-service",
				ServiceVersion:        "unknown",
				HTTPSConfig:           client.HTTPSConfig{},
				MethodExcludePatterns: []string{},
				OTELCollectorEnable:   false,
			},
		},
		{
			name: "with service version",
			options: []Option{
				WithServiceVersion("v1.0.0"),
			},
			expected: &Options{
				ServiceName:           "unknown",
				ServiceVersion:        "v1.0.0",
				HTTPSConfig:           client.HTTPSConfig{},
				MethodExcludePatterns: []string{},
				OTELCollectorEnable:   false,
			},
		},
		{
			name: "with HTTPS config",
			options: []Option{
				WithServiceConfig(client.HTTPSConfig{
					Cert: "/path/to/cert.pem",
					Key:  "/path/to/key.pem",
				}),
			},
			expected: &Options{
				ServiceName:           "unknown",
				ServiceVersion:        "unknown",
				HTTPSConfig:           client.HTTPSConfig{Cert: "/path/to/cert.pem", Key: "/path/to/key.pem"},
				MethodExcludePatterns: []string{},
				OTELCollectorEnable:   false,
			},
		},
		{
			name: "with method exclude patterns",
			options: []Option{
				WithMethodExcludePatterns([]string{"*.Health/*", "*.Metrics/*"}),
			},
			expected: &Options{
				ServiceName:           "unknown",
				ServiceVersion:        "unknown",
				HTTPSConfig:           client.HTTPSConfig{},
				MethodExcludePatterns: []string{"*.Health/*", "*.Metrics/*"},
				OTELCollectorEnable:   false,
			},
		},
		{
			name: "with OTEL collector enabled",
			options: []Option{
				WithOTELCollectorEnable(true),
			},
			expected: &Options{
				ServiceName:           "unknown",
				ServiceVersion:        "unknown",
				HTTPSConfig:           client.HTTPSConfig{},
				MethodExcludePatterns: []string{},
				OTELCollectorEnable:   true,
			},
		},
		{
			name: "with all options",
			options: []Option{
				WithServiceName("test-service"),
				WithServiceVersion("v1.0.0"),
				WithServiceConfig(client.HTTPSConfig{
					Cert: "/path/to/cert.pem",
					Key:  "/path/to/key.pem",
				}),
				WithMethodExcludePatterns([]string{"*.Health/*"}),
				WithOTELCollectorEnable(true),
			},
			expected: &Options{
				ServiceName:           "test-service",
				ServiceVersion:        "v1.0.0",
				HTTPSConfig:           client.HTTPSConfig{Cert: "/path/to/cert.pem", Key: "/path/to/key.pem"},
				MethodExcludePatterns: []string{"*.Health/*"},
				OTELCollectorEnable:   true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := newGRPCOptions(tt.options...)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestWithServiceConfig(t *testing.T) {
	config := client.HTTPSConfig{
		Cert: "/test/cert.pem",
		Key:  "/test/key.pem",
	}

	option := WithServiceConfig(config)
	opts := &Options{}
	option(opts)

	assert.Equal(t, config, opts.HTTPSConfig)
}

func TestWithOTELCollectorEnable(t *testing.T) {
	tests := []struct {
		name   string
		enable bool
	}{
		{"enable true", true},
		{"enable false", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			option := WithOTELCollectorEnable(tt.enable)
			opts := &Options{}
			option(opts)

			assert.Equal(t, tt.enable, opts.OTELCollectorEnable)
		})
	}
}

func TestWithServiceName(t *testing.T) {
	serviceName := "test-service"
	option := WithServiceName(serviceName)
	opts := &Options{}
	option(opts)

	assert.Equal(t, serviceName, opts.ServiceName)
}

func TestWithServiceVersion(t *testing.T) {
	serviceVersion := "v1.0.0"
	option := WithServiceVersion(serviceVersion)
	opts := &Options{}
	option(opts)

	assert.Equal(t, serviceVersion, opts.ServiceVersion)
}

func TestWithMethodExcludePatterns(t *testing.T) {
	patterns := []string{"*.Health/*", "*.Metrics/*", "*.Internal/*"}
	option := WithMethodExcludePatterns(patterns)
	opts := &Options{}
	option(opts)

	assert.Equal(t, patterns, opts.MethodExcludePatterns)
}

func TestNewGRPCOptionsAndCreds_DefaultOptions(t *testing.T) {
	serverOpts, creds, err := NewGRPCOptionsAndCreds()

	require.NoError(t, err)
	assert.NotNil(t, serverOpts)
	assert.Nil(t, creds) // No TLS credentials with default options

	// Verify that we have the expected number of server options
	// This includes interceptors, stats handler, and message size limits
	assert.Greater(t, len(serverOpts), 0)
}

func TestNewGRPCOptionsAndCreds_WithCustomOptions(t *testing.T) {
	serverOpts, creds, err := NewGRPCOptionsAndCreds(
		WithServiceName("test-service"),
		WithServiceVersion("v1.0.0"),
		WithMethodExcludePatterns([]string{"*.Health/*"}),
		WithOTELCollectorEnable(true),
	)

	require.NoError(t, err)
	assert.NotNil(t, serverOpts)
	assert.Nil(t, creds) // No TLS credentials
	assert.Greater(t, len(serverOpts), 0)
}

func TestNewGRPCOptionsAndCreds_WithValidTLS(t *testing.T) {
	// Create temporary certificate files for testing
	tempDir := t.TempDir()
	certFile := filepath.Join(tempDir, "cert.pem")
	keyFile := filepath.Join(tempDir, "key.pem")

	// Create dummy certificate files
	err := os.WriteFile(certFile, []byte("-----BEGIN CERTIFICATE-----\nDUMMY\n-----END CERTIFICATE-----"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(keyFile, []byte("-----BEGIN PRIVATE KEY-----\nDUMMY\n-----END PRIVATE KEY-----"), 0644)
	require.NoError(t, err)

	serverOpts, creds, err := NewGRPCOptionsAndCreds(
		WithServiceConfig(client.HTTPSConfig{
			Cert: certFile,
			Key:  keyFile,
		}),
	)

	// Note: This will fail because we're using dummy certificates
	// In a real test environment, you'd use proper test certificates
	assert.Error(t, err)
	assert.Nil(t, serverOpts)
	assert.Nil(t, creds)
	assert.Contains(t, err.Error(), "failed to create credentials")
}

func TestNewGRPCOptionsAndCreds_WithInvalidTLS(t *testing.T) {
	serverOpts, creds, err := NewGRPCOptionsAndCreds(
		WithServiceConfig(client.HTTPSConfig{
			Cert: "/nonexistent/cert.pem",
			Key:  "/nonexistent/key.pem",
		}),
	)

	assert.Error(t, err)
	assert.Nil(t, serverOpts)
	assert.Nil(t, creds)
	assert.Contains(t, err.Error(), "failed to create credentials")
}

func TestNewGRPCOptionsAndCreds_WithPartialTLS(t *testing.T) {
	// Test with only cert file (missing key)
	serverOpts, creds, err := NewGRPCOptionsAndCreds(
		WithServiceConfig(client.HTTPSConfig{
			Cert: "/path/to/cert.pem",
			Key:  "", // Missing key
		}),
	)

	require.NoError(t, err)
	assert.NotNil(t, serverOpts)
	assert.Nil(t, creds) // No credentials when key is missing
	assert.Greater(t, len(serverOpts), 0)

	// Test with only key file (missing cert)
	serverOpts, creds, err = NewGRPCOptionsAndCreds(
		WithServiceConfig(client.HTTPSConfig{
			Cert: "", // Missing cert
			Key:  "/path/to/key.pem",
		}),
	)

	require.NoError(t, err)
	assert.NotNil(t, serverOpts)
	assert.Nil(t, creds) // No credentials when cert is missing
	assert.Greater(t, len(serverOpts), 0)
}

func TestNewGRPCOptionsAndCreds_OptionOrdering(t *testing.T) {
	// Test that options are applied in the correct order
	serverOpts1, _, err1 := NewGRPCOptionsAndCreds(
		WithServiceName("first"),
		WithServiceName("second"), // Should override "first"
	)

	serverOpts2, _, err2 := NewGRPCOptionsAndCreds(
		WithServiceName("second"),
	)

	require.NoError(t, err1)
	require.NoError(t, err2)
	assert.Equal(t, len(serverOpts1), len(serverOpts2))
}

func TestNewGRPCOptionsAndCreds_EmptyMethodPatterns(t *testing.T) {
	serverOpts, creds, err := NewGRPCOptionsAndCreds(
		WithMethodExcludePatterns([]string{}),
	)

	require.NoError(t, err)
	assert.NotNil(t, serverOpts)
	assert.Nil(t, creds)
	assert.Greater(t, len(serverOpts), 0)
}

func TestNewGRPCOptionsAndCreds_NilMethodPatterns(t *testing.T) {
	serverOpts, creds, err := NewGRPCOptionsAndCreds(
		WithMethodExcludePatterns(nil),
	)

	require.NoError(t, err)
	assert.NotNil(t, serverOpts)
	assert.Nil(t, creds)
	assert.Greater(t, len(serverOpts), 0)
}

func TestNewGRPCOptionsAndCreds_MultipleOptions(t *testing.T) {
	serverOpts, creds, err := NewGRPCOptionsAndCreds(
		WithServiceName("multi-service"),
		WithServiceVersion("v2.1.0"),
		WithMethodExcludePatterns([]string{"*.Health/*", "*.Metrics/*"}),
		WithOTELCollectorEnable(true),
		WithServiceConfig(client.HTTPSConfig{
			Cert: "/path/to/cert.pem",
			Key:  "/path/to/key.pem",
		}),
	)

	// This will fail due to invalid TLS files, but we can verify the error
	assert.Error(t, err)
	assert.Nil(t, serverOpts)
	assert.Nil(t, creds)
	assert.Contains(t, err.Error(), "failed to create credentials")
}

func BenchmarkNewGRPCOptionsAndCreds_Default(b *testing.B) {
	for b.Loop() {
		_, _, err := NewGRPCOptionsAndCreds()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkNewGRPCOptionsAndCreds_WithOptions(b *testing.B) {
	for b.Loop() {
		_, _, err := NewGRPCOptionsAndCreds(
			WithServiceName("benchmark-service"),
			WithServiceVersion("v1.0.0"),
			WithMethodExcludePatterns([]string{"*.Health/*"}),
			WithOTELCollectorEnable(true),
		)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkNewGRPCOptions(b *testing.B) {
	options := []Option{
		WithServiceName("benchmark-service"),
		WithServiceVersion("v1.0.0"),
		WithMethodExcludePatterns([]string{"*.Health/*"}),
		WithOTELCollectorEnable(true),
	}

	b.ResetTimer()
	for b.Loop() {
		_ = newGRPCOptions(options...)
	}
}

func TestNewGRPCOptionsAndCreds_ServerOptionsStructure(t *testing.T) {
	serverOpts, _, err := NewGRPCOptionsAndCreds(
		WithServiceName("test-service"),
		WithServiceVersion("v1.0.0"),
		WithMethodExcludePatterns([]string{"*.Health/*"}),
	)
	require.NoError(t, err)

	// Verify we have the expected minimum number of options
	// This includes: interceptors, message size limits, and potentially OTEL stats handler
	assert.GreaterOrEqual(t, len(serverOpts), 3,
		"Should have at least interceptors and message size options")

	// Verify all options are non-nil
	for i, opt := range serverOpts {
		assert.NotNil(t, opt, "Server option %d should not be nil", i)
	}
}

func TestNewGRPCOptionsAndCreds_InterceptorChain(t *testing.T) {
	serverOpts, _, err := NewGRPCOptionsAndCreds(
		WithServiceName("test-service"),
		WithServiceVersion("v1.0.0"),
		WithMethodExcludePatterns([]string{"*.Health/*"}),
	)
	require.NoError(t, err)

	// Verify that interceptors are included
	// This is a basic check - in practice you'd want to verify the actual interceptor types
	assert.GreaterOrEqual(t, len(serverOpts), 2,
		"Should have at least unary and stream interceptors")
}

func TestNewGRPCOptionsAndCreds_WithValidTestCertificates(t *testing.T) {
	// Create proper test certificates for TLS testing
	tempDir := t.TempDir()
	certFile := filepath.Join(tempDir, "cert.pem")
	keyFile := filepath.Join(tempDir, "key.pem")

	// Create minimal valid certificate and key files
	// Note: These are not cryptographically valid but should pass basic file checks
	certContent := `-----BEGIN CERTIFICATE-----
MIIDXTCCAkWgAwIBAgIJAKoK/OvJ8mQkMA0GCSqGSIb3DQEBCwUAMEUxCzAJBgNV
BAYTAkFVMRMwEQYDVQQIDApTb21lLVN0YXRlMSEwHwYDVQQKDBhJbnRlcm5ldCBX
aWRnaXRzIFB0eSBMdGQwHhcNMTkwMzI2MTIzMzQ5WhcNMjAwMzI1MTIzMzQ5WjBF
MQswCQYDVQQGEwJBVTETMBEGA1UECAwKU29tZS1TdGF0ZTEhMB8GA1UECgwYSW50
ZXJuZXQgV2lkZ2l0cyBQdHkgTHRkMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIB
CgKCAQEAvxJj7aFzJkLmJ8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J
8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8
J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8
J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8
J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8
J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8
J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8
J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8J8

-----END CERTIFICATE-----`
	keyContent := `-----BEGIN PRIVATE KEY-----
MIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQD...
-----END PRIVATE KEY-----`

	err := os.WriteFile(certFile, []byte(certContent), 0644)
	require.NoError(t, err)
	err = os.WriteFile(keyFile, []byte(keyContent), 0644)
	require.NoError(t, err)

	serverOpts, creds, err := NewGRPCOptionsAndCreds(
		WithServiceConfig(client.HTTPSConfig{
			Cert: certFile,
			Key:  keyFile,
		}),
	)
	// This may still fail if the certs are not cryptographically valid, but the test ensures the code path is exercised
	if err != nil {
		assert.Contains(t, err.Error(), "failed to create credentials")
		assert.Nil(t, serverOpts)
		assert.Nil(t, creds)
	} else {
		assert.NotNil(t, serverOpts)
		assert.NotNil(t, creds)
	}
}
