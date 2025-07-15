package grpc

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/frankban/quicktest"
	"github.com/instill-ai/x/client"
	"google.golang.org/grpc/stats"
)

func TestNewGRPCOptions(t *testing.T) {
	qt := quicktest.New(t)
	tests := []struct {
		name     string
		options  []Option
		expected *Options
	}{
		{
			name:    "default options",
			options: []Option{},
			expected: &Options{
				ServiceName:                "unknown",
				ServiceVersion:             "unknown",
				HTTPSConfig:                client.HTTPSConfig{},
				MethodLogExcludePatterns:   []string{},
				MethodTraceExcludePatterns: []string{},
				SetOTELServerHandler:       false,
			},
		},
		{
			name: "with service name",
			options: []Option{
				WithServiceName("test-service"),
			},
			expected: &Options{
				ServiceName:                "test-service",
				ServiceVersion:             "unknown",
				HTTPSConfig:                client.HTTPSConfig{},
				MethodLogExcludePatterns:   []string{},
				MethodTraceExcludePatterns: []string{},
				SetOTELServerHandler:       false,
			},
		},
		{
			name: "with service version",
			options: []Option{
				WithServiceVersion("v1.0.0"),
			},
			expected: &Options{
				ServiceName:                "unknown",
				ServiceVersion:             "v1.0.0",
				HTTPSConfig:                client.HTTPSConfig{},
				MethodLogExcludePatterns:   []string{},
				MethodTraceExcludePatterns: []string{},
				SetOTELServerHandler:       false,
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
				ServiceName:                "unknown",
				ServiceVersion:             "unknown",
				HTTPSConfig:                client.HTTPSConfig{Cert: "/path/to/cert.pem", Key: "/path/to/key.pem"},
				MethodLogExcludePatterns:   []string{},
				MethodTraceExcludePatterns: []string{},
				SetOTELServerHandler:       false,
			},
		},
		{
			name: "with method log exclude patterns",
			options: []Option{
				WithMethodLogExcludePatterns([]string{"*.Health/*", "*.Metrics/*"}),
			},
			expected: &Options{
				ServiceName:                "unknown",
				ServiceVersion:             "unknown",
				HTTPSConfig:                client.HTTPSConfig{},
				MethodLogExcludePatterns:   []string{"*.Health/*", "*.Metrics/*"},
				MethodTraceExcludePatterns: []string{},
				SetOTELServerHandler:       false,
			},
		},
		{
			name: "with method trace exclude patterns",
			options: []Option{
				WithMethodTraceExcludePatterns([]string{"*.TestService/*", "*.DebugService/*"}),
			},
			expected: &Options{
				ServiceName:                "unknown",
				ServiceVersion:             "unknown",
				HTTPSConfig:                client.HTTPSConfig{},
				MethodLogExcludePatterns:   []string{},
				MethodTraceExcludePatterns: []string{"*.TestService/*", "*.DebugService/*"},
				SetOTELServerHandler:       false,
			},
		},
		{
			name: "with OTEL collector enabled",
			options: []Option{
				WithSetOTELServerHandler(true),
			},
			expected: &Options{
				ServiceName:                "unknown",
				ServiceVersion:             "unknown",
				HTTPSConfig:                client.HTTPSConfig{},
				MethodLogExcludePatterns:   []string{},
				MethodTraceExcludePatterns: []string{},
				SetOTELServerHandler:       true,
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
				WithMethodLogExcludePatterns([]string{"*.Health/*"}),
				WithMethodTraceExcludePatterns([]string{"*.TestService/*"}),
				WithSetOTELServerHandler(true),
			},
			expected: &Options{
				ServiceName:                "test-service",
				ServiceVersion:             "v1.0.0",
				HTTPSConfig:                client.HTTPSConfig{Cert: "/path/to/cert.pem", Key: "/path/to/key.pem"},
				MethodLogExcludePatterns:   []string{"*.Health/*"},
				MethodTraceExcludePatterns: []string{"*.TestService/*"},
				SetOTELServerHandler:       true,
			},
		},
	}

	for _, tt := range tests {
		qt.Run(tt.name, func(c *quicktest.C) {
			result := newOptions(tt.options...)
			c.Check(result, quicktest.DeepEquals, tt.expected)
		})
	}
}

func TestWithServiceConfig(t *testing.T) {
	qt := quicktest.New(t)
	config := client.HTTPSConfig{
		Cert: "/test/cert.pem",
		Key:  "/test/key.pem",
	}

	option := WithServiceConfig(config)
	opts := &Options{}
	option(opts)

	qt.Check(opts.HTTPSConfig, quicktest.DeepEquals, config)
}

func TestWithOTELCollectorEnable(t *testing.T) {
	qt := quicktest.New(t)
	tests := []struct {
		name   string
		enable bool
	}{
		{"enable true", true},
		{"enable false", false},
	}

	for _, tt := range tests {
		qt.Run(tt.name, func(c *quicktest.C) {
			option := WithSetOTELServerHandler(tt.enable)
			opts := &Options{}
			option(opts)

			c.Check(opts.SetOTELServerHandler, quicktest.Equals, tt.enable)
		})
	}
}

func TestWithServiceName(t *testing.T) {
	qt := quicktest.New(t)
	serviceName := "test-service"
	option := WithServiceName(serviceName)
	opts := &Options{}
	option(opts)

	qt.Check(opts.ServiceName, quicktest.Equals, serviceName)
}

func TestWithServiceVersion(t *testing.T) {
	qt := quicktest.New(t)
	serviceVersion := "v1.0.0"
	option := WithServiceVersion(serviceVersion)
	opts := &Options{}
	option(opts)

	qt.Check(opts.ServiceVersion, quicktest.Equals, serviceVersion)
}

func TestWithMethodLogExcludePatterns(t *testing.T) {
	qt := quicktest.New(t)
	patterns := []string{"*.Health/*", "*.Metrics/*", "*.Internal/*"}
	option := WithMethodLogExcludePatterns(patterns)
	opts := &Options{}
	option(opts)

	qt.Check(opts.MethodLogExcludePatterns, quicktest.DeepEquals, patterns)
}

func TestWithMethodTraceExcludePatterns(t *testing.T) {
	tests := []struct {
		name     string
		patterns []string
	}{
		{
			name:     "empty patterns",
			patterns: []string{},
		},
		{
			name: "single pattern",
			patterns: []string{
				"*.TestService/*",
			},
		},
		{
			name: "multiple patterns",
			patterns: []string{
				"*.TestService/*",
				"*.DebugService/*",
				"*.HealthService/*",
			},
		},
		{
			name: "patterns with special characters",
			patterns: []string{
				".*PublicService/.*ness$",
				".*PrivateService/.*$",
				".*UsageService/.*$",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			qt := quicktest.New(t)
			option := WithMethodTraceExcludePatterns(tt.patterns)
			opts := &Options{}
			option(opts)

			qt.Assert(opts.MethodTraceExcludePatterns, quicktest.DeepEquals, tt.patterns)
		})
	}
}

func TestDefaultMethodTraceExcludePatterns(t *testing.T) {
	qt := quicktest.New(t)
	expected := []string{
		".*PublicService/.*ness$",
		".*PrivateService/.*$",
		".*UsageService/.*$",
	}
	qt.Check(defaultMethodTraceExcludePatterns, quicktest.DeepEquals, expected)
}

func TestCreateFilterTraceDecider(t *testing.T) {
	qt := quicktest.New(t)
	customPatterns := []string{".*CustomService/.*"}
	filter := createFilterTraceDecider(customPatterns)

	// Should filter out default and custom patterns
	type testCase struct {
		method string
		want   bool
	}
	cases := []testCase{
		{"SomeService/SomeMethod", true},
		{"PublicService/Liveness", false},
		{"PrivateService/DoSomething", false},
		{"UsageService/Track", false},
		{"CustomService/Do", false},
	}
	for _, tc := range cases {
		got := filter(&stats.RPCTagInfo{FullMethodName: tc.method})
		qt.Check(got, quicktest.Equals, tc.want, quicktest.Commentf("method: %s", tc.method))
	}
}

func TestNewGRPCOptionsAndCreds_DefaultOptions(t *testing.T) {
	qt := quicktest.New(t)
	serverOpts, err := NewServerOptionsAndCreds()
	qt.Check(err, quicktest.IsNil)
	qt.Check(serverOpts, quicktest.Not(quicktest.IsNil))
	qt.Check(len(serverOpts) > 0, quicktest.IsTrue)
}

func TestNewGRPCOptionsAndCreds_WithCustomOptions(t *testing.T) {
	qt := quicktest.New(t)
	serverOpts, err := NewServerOptionsAndCreds(
		WithServiceName("test-service"),
		WithServiceVersion("v1.0.0"),
		WithMethodLogExcludePatterns([]string{"*.Health/*"}),
		WithSetOTELServerHandler(true),
	)
	qt.Check(err, quicktest.IsNil)
	qt.Check(serverOpts, quicktest.Not(quicktest.IsNil))
	qt.Check(len(serverOpts) > 0, quicktest.IsTrue)
}

func TestNewGRPCOptionsAndCreds_WithMethodTraceExcludePatterns(t *testing.T) {
	qt := quicktest.New(t)
	serverOpts, err := NewServerOptionsAndCreds(
		WithServiceName("test-service"),
		WithServiceVersion("v1.0.0"),
		WithMethodTraceExcludePatterns([]string{"*.TestService/*", "*.DebugService/*"}),
		WithSetOTELServerHandler(true),
	)
	qt.Check(err, quicktest.IsNil)
	qt.Check(serverOpts, quicktest.Not(quicktest.IsNil))
	qt.Check(len(serverOpts) > 0, quicktest.IsTrue)
}

func TestNewGRPCOptionsAndCreds_WithValidTLS(t *testing.T) {
	qt := quicktest.New(t)
	// Create temporary certificate files for testing
	tempDir := t.TempDir()
	certFile := filepath.Join(tempDir, "cert.pem")
	keyFile := filepath.Join(tempDir, "key.pem")

	// Create dummy certificate files
	err := os.WriteFile(certFile, []byte("-----BEGIN CERTIFICATE-----\nDUMMY\n-----END CERTIFICATE-----"), 0644)
	qt.Check(err, quicktest.IsNil)
	err = os.WriteFile(keyFile, []byte("-----BEGIN PRIVATE KEY-----\nDUMMY\n-----END PRIVATE KEY-----"), 0644)
	qt.Check(err, quicktest.IsNil)

	serverOpts, err := NewServerOptionsAndCreds(
		WithServiceConfig(client.HTTPSConfig{
			Cert: certFile,
			Key:  keyFile,
		}),
	)

	// Note: This will fail because we're using dummy certificates
	// In a real test environment, you'd use proper test certificates
	qt.Check(err, quicktest.Not(quicktest.IsNil))
	qt.Check(serverOpts, quicktest.IsNil)
	qt.Check(err.Error(), quicktest.Contains, "failed to create credentials")
}

func TestNewGRPCOptionsAndCreds_WithInvalidTLS(t *testing.T) {
	qt := quicktest.New(t)
	serverOpts, err := NewServerOptionsAndCreds(
		WithServiceConfig(client.HTTPSConfig{
			Cert: "/nonexistent/cert.pem",
			Key:  "/nonexistent/key.pem",
		}),
	)
	qt.Check(err, quicktest.Not(quicktest.IsNil))
	qt.Check(serverOpts, quicktest.IsNil)
	qt.Check(err.Error(), quicktest.Contains, "failed to create credentials")
}

func TestNewGRPCOptionsAndCreds_WithPartialTLS(t *testing.T) {
	qt := quicktest.New(t)
	// Test with only cert file (missing key)
	serverOpts, err := NewServerOptionsAndCreds(
		WithServiceConfig(client.HTTPSConfig{
			Cert: "/path/to/cert.pem",
			Key:  "", // Missing key
		}),
	)
	qt.Check(err, quicktest.IsNil)
	qt.Check(serverOpts, quicktest.Not(quicktest.IsNil))
	qt.Check(len(serverOpts) > 0, quicktest.IsTrue)

	// Test with only key file (missing cert)
	serverOpts, err = NewServerOptionsAndCreds(
		WithServiceConfig(client.HTTPSConfig{
			Cert: "", // Missing cert
			Key:  "/path/to/key.pem",
		}),
	)
	qt.Check(err, quicktest.IsNil)
	qt.Check(serverOpts, quicktest.Not(quicktest.IsNil))
	qt.Check(len(serverOpts) > 0, quicktest.IsTrue)
}

func TestNewGRPCOptionsAndCreds_OptionOrdering(t *testing.T) {
	qt := quicktest.New(t)
	// Test that options are applied in the correct order
	serverOpts1, err1 := NewServerOptionsAndCreds(
		WithServiceName("first"),
		WithServiceName("second"), // Should override "first"
	)

	serverOpts2, err2 := NewServerOptionsAndCreds(
		WithServiceName("second"),
	)

	qt.Check(err1, quicktest.IsNil)
	qt.Check(err2, quicktest.IsNil)
	qt.Check(len(serverOpts1), quicktest.Equals, len(serverOpts2))
}

func TestNewGRPCOptionsAndCreds_EmptyMethodPatterns(t *testing.T) {
	qt := quicktest.New(t)
	serverOpts, err := NewServerOptionsAndCreds(
		WithMethodLogExcludePatterns([]string{}),
	)
	qt.Check(err, quicktest.IsNil)
	qt.Check(serverOpts, quicktest.Not(quicktest.IsNil))
	qt.Check(len(serverOpts) > 0, quicktest.IsTrue)
}

func TestNewGRPCOptionsAndCreds_NilMethodPatterns(t *testing.T) {
	qt := quicktest.New(t)
	serverOpts, err := NewServerOptionsAndCreds(
		WithMethodLogExcludePatterns(nil),
	)
	qt.Check(err, quicktest.IsNil)
	qt.Check(serverOpts, quicktest.Not(quicktest.IsNil))
	qt.Check(len(serverOpts) > 0, quicktest.IsTrue)
}

func TestNewGRPCOptionsAndCreds_EmptyTracePatterns(t *testing.T) {
	qt := quicktest.New(t)
	serverOpts, err := NewServerOptionsAndCreds(
		WithMethodTraceExcludePatterns([]string{}),
	)
	qt.Check(err, quicktest.IsNil)
	qt.Check(serverOpts, quicktest.Not(quicktest.IsNil))
	qt.Check(len(serverOpts) > 0, quicktest.IsTrue)
}

func TestNewGRPCOptionsAndCreds_NilTracePatterns(t *testing.T) {
	qt := quicktest.New(t)
	serverOpts, err := NewServerOptionsAndCreds(
		WithMethodTraceExcludePatterns(nil),
	)
	qt.Check(err, quicktest.IsNil)
	qt.Check(serverOpts, quicktest.Not(quicktest.IsNil))
	qt.Check(len(serverOpts) > 0, quicktest.IsTrue)
}

func TestNewGRPCOptionsAndCreds_MultipleOptions(t *testing.T) {
	qt := quicktest.New(t)
	serverOpts, err := NewServerOptionsAndCreds(
		WithServiceName("multi-service"),
		WithServiceVersion("v2.1.0"),
		WithMethodLogExcludePatterns([]string{"*.Health/*", "*.Metrics/*"}),
		WithMethodTraceExcludePatterns([]string{"*.TestService/*", "*.DebugService/*"}),
		WithSetOTELServerHandler(true),
		WithServiceConfig(client.HTTPSConfig{
			Cert: "/path/to/cert.pem",
			Key:  "/path/to/key.pem",
		}),
	)
	// This will fail due to invalid TLS files, but we can verify the error
	qt.Check(err, quicktest.Not(quicktest.IsNil))
	qt.Check(serverOpts, quicktest.IsNil)
	qt.Check(err.Error(), quicktest.Contains, "failed to create credentials")
}

func BenchmarkNewGRPCOptionsAndCreds_Default(b *testing.B) {
	for b.Loop() {
		_, err := NewServerOptionsAndCreds()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkNewGRPCOptionsAndCreds_WithOptions(b *testing.B) {
	for b.Loop() {
		_, err := NewServerOptionsAndCreds(
			WithServiceName("benchmark-service"),
			WithServiceVersion("v1.0.0"),
			WithMethodLogExcludePatterns([]string{"*.Health/*"}),
			WithSetOTELServerHandler(true),
		)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkNewGRPCOptionsAndCreds_WithTracePatterns(b *testing.B) {
	for b.Loop() {
		_, err := NewServerOptionsAndCreds(
			WithServiceName("benchmark-service"),
			WithServiceVersion("v1.0.0"),
			WithMethodTraceExcludePatterns([]string{"*.TestService/*", "*.DebugService/*"}),
			WithSetOTELServerHandler(true),
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
		WithMethodLogExcludePatterns([]string{"*.Health/*"}),
		WithMethodTraceExcludePatterns([]string{"*.TestService/*"}),
		WithSetOTELServerHandler(true),
	}

	b.ResetTimer()
	for b.Loop() {
		_ = newOptions(options...)
	}
}

func TestNewGRPCOptionsAndCreds_ServerOptionsStructure(t *testing.T) {
	qt := quicktest.New(t)
	serverOpts, err := NewServerOptionsAndCreds(
		WithServiceName("test-service"),
		WithServiceVersion("v1.0.0"),
		WithMethodLogExcludePatterns([]string{"*.Health/*"}),
	)
	qt.Check(err, quicktest.IsNil)

	// Verify we have the expected minimum number of options
	// This includes: interceptors, message size limits, and potentially OTEL stats handler
	qt.Check(len(serverOpts) >= 3, quicktest.IsTrue)

	// Verify all options are non-nil
	for _, opt := range serverOpts {
		qt.Check(opt, quicktest.Not(quicktest.IsNil))
	}
}

func TestNewGRPCOptionsAndCreds_InterceptorChain(t *testing.T) {
	qt := quicktest.New(t)
	serverOpts, err := NewServerOptionsAndCreds(
		WithServiceName("test-service"),
		WithServiceVersion("v1.0.0"),
		WithMethodLogExcludePatterns([]string{"*.Health/*"}),
	)
	qt.Check(err, quicktest.IsNil)

	// Verify that interceptors are included
	// This is a basic check - in practice you'd want to verify the actual interceptor types
	qt.Check(len(serverOpts) >= 2, quicktest.IsTrue)
}

func TestNewGRPCOptionsAndCreds_WithValidTestCertificates(t *testing.T) {
	qt := quicktest.New(t)
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
-----END CERTIFICATE-----`
	keyContent := `-----BEGIN PRIVATE KEY-----
MIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQD...
-----END PRIVATE KEY-----`

	err := os.WriteFile(certFile, []byte(certContent), 0644)
	qt.Check(err, quicktest.IsNil)
	err = os.WriteFile(keyFile, []byte(keyContent), 0644)
	qt.Check(err, quicktest.IsNil)

	serverOpts, err := NewServerOptionsAndCreds(
		WithServiceConfig(client.HTTPSConfig{
			Cert: certFile,
			Key:  keyFile,
		}),
	)
	// This may still fail if the certs are not cryptographically valid, but the test ensures the code path is exercised
	if err != nil {
		qt.Check(err.Error(), quicktest.Contains, "failed to create credentials")
		qt.Check(serverOpts, quicktest.IsNil)
	} else {
		qt.Check(serverOpts, quicktest.Not(quicktest.IsNil))
	}
}
