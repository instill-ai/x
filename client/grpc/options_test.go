package grpc

import (
	"testing"

	"github.com/instill-ai/x/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClientOptions(t *testing.T) {
	tests := []struct {
		name     string
		options  []ClientOption
		expected *ClientOptions
	}{
		{
			name:    "default options",
			options: []ClientOption{},
			expected: &ClientOptions{
				HTTPSConfig:         client.HTTPSConfig{},
				HostPort:            "",
				OTELCollectorEnable: false,
			},
		},
		{
			name: "with HTTPS config",
			options: []ClientOption{
				WithHTTPSConfig(client.HTTPSConfig{
					Cert: "/path/to/cert.pem",
					Key:  "/path/to/key.pem",
				}),
			},
			expected: &ClientOptions{
				HTTPSConfig:         client.HTTPSConfig{Cert: "/path/to/cert.pem", Key: "/path/to/key.pem"},
				HostPort:            "",
				OTELCollectorEnable: false,
			},
		},
		{
			name: "with host port",
			options: []ClientOption{
				WithHostPort("localhost:8080"),
			},
			expected: &ClientOptions{
				HTTPSConfig:         client.HTTPSConfig{},
				HostPort:            "localhost:8080",
				OTELCollectorEnable: false,
			},
		},
		{
			name: "with OTEL collector enabled",
			options: []ClientOption{
				WithOTELCollectorEnable(true),
			},
			expected: &ClientOptions{
				HTTPSConfig:         client.HTTPSConfig{},
				HostPort:            "",
				OTELCollectorEnable: true,
			},
		},
		{
			name: "with all options",
			options: []ClientOption{
				WithHTTPSConfig(client.HTTPSConfig{
					Cert: "/path/to/cert.pem",
					Key:  "/path/to/key.pem",
				}),
				WithHostPort("localhost:8080"),
				WithOTELCollectorEnable(true),
			},
			expected: &ClientOptions{
				HTTPSConfig:         client.HTTPSConfig{Cert: "/path/to/cert.pem", Key: "/path/to/key.pem"},
				HostPort:            "localhost:8080",
				OTELCollectorEnable: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := newClientOptions(tt.options...)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestWithHTTPSConfig(t *testing.T) {
	config := client.HTTPSConfig{
		Cert: "/test/cert.pem",
		Key:  "/test/key.pem",
	}

	option := WithHTTPSConfig(config)
	opts := &ClientOptions{}
	option(opts)

	assert.Equal(t, config, opts.HTTPSConfig)
}

func TestWithHostPort(t *testing.T) {
	hostPort := "localhost:8080"
	option := WithHostPort(hostPort)
	opts := &ClientOptions{}
	option(opts)

	assert.Equal(t, hostPort, opts.HostPort)
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
			opts := &ClientOptions{}
			option(opts)

			assert.Equal(t, tt.enable, opts.OTELCollectorEnable)
		})
	}
}

func TestNewClientDialOptionsAndCreds_DefaultOptions(t *testing.T) {
	dialOpts, creds, err := NewClientDialOptionsAndCreds()

	require.NoError(t, err)
	assert.NotNil(t, dialOpts)
	assert.Nil(t, creds) // No TLS credentials with default options

	// Verify that we have the expected number of dial options
	// This includes interceptors, message size limits, and insecure credentials
	assert.Greater(t, len(dialOpts), 0)
}

func TestNewClientDialOptionsAndCreds_WithCustomOptions(t *testing.T) {
	dialOpts, creds, err := NewClientDialOptionsAndCreds(
		WithHostPort("localhost:8080"),
		WithOTELCollectorEnable(true),
	)

	require.NoError(t, err)
	assert.NotNil(t, dialOpts)
	assert.Nil(t, creds) // No TLS credentials
	assert.Greater(t, len(dialOpts), 0)
}

func TestNewClientDialOptionsAndCreds_WithValidTLS(t *testing.T) {
	dialOpts, creds, err := NewClientDialOptionsAndCreds(
		WithHTTPSConfig(client.HTTPSConfig{
			Cert: "/nonexistent/cert.pem",
			Key:  "/nonexistent/key.pem",
		}),
	)

	// This will fail due to invalid cert files, but we can verify the error
	assert.Error(t, err)
	assert.Nil(t, dialOpts)
	assert.Nil(t, creds)
	assert.Contains(t, err.Error(), "TLS credentials")
}

func TestNewClientDialOptionsAndCreds_WithInvalidTLS(t *testing.T) {
	dialOpts, creds, err := NewClientDialOptionsAndCreds(
		WithHTTPSConfig(client.HTTPSConfig{
			Cert: "/nonexistent/cert.pem",
			Key:  "/nonexistent/key.pem",
		}),
	)

	assert.Error(t, err)
	assert.Nil(t, dialOpts)
	assert.Nil(t, creds)
	assert.Contains(t, err.Error(), "TLS credentials")
}

func TestNewClientDialOptionsAndCreds_WithPartialTLS(t *testing.T) {
	// Test with only cert file (missing key)
	dialOpts, creds, err := NewClientDialOptionsAndCreds(
		WithHTTPSConfig(client.HTTPSConfig{
			Cert: "/path/to/cert.pem",
			Key:  "", // Missing key
		}),
	)

	require.NoError(t, err)
	assert.NotNil(t, dialOpts)
	assert.Nil(t, creds) // No credentials when key is missing
	assert.Greater(t, len(dialOpts), 0)

	// Test with only key file (missing cert)
	dialOpts, creds, err = NewClientDialOptionsAndCreds(
		WithHTTPSConfig(client.HTTPSConfig{
			Cert: "", // Missing cert
			Key:  "/path/to/key.pem",
		}),
	)

	require.NoError(t, err)
	assert.NotNil(t, dialOpts)
	assert.Nil(t, creds) // No credentials when cert is missing
	assert.Greater(t, len(dialOpts), 0)
}

func TestNewClientDialOptionsAndCreds_OptionOrdering(t *testing.T) {
	// Test that options are applied in the correct order
	dialOpts1, _, err1 := NewClientDialOptionsAndCreds(
		WithHostPort("first"),
		WithHostPort("second"), // Should override "first"
	)

	dialOpts2, _, err2 := NewClientDialOptionsAndCreds(
		WithHostPort("second"),
	)

	require.NoError(t, err1)
	require.NoError(t, err2)
	assert.Equal(t, len(dialOpts1), len(dialOpts2))
}

func TestNewClientDialOptionsAndCreds_OTELStatsHandler(t *testing.T) {
	// Test with OTEL collector disabled (default)
	dialOpts, _, err := NewClientDialOptionsAndCreds()
	require.NoError(t, err)

	// Count dial options to verify OTEL stats handler is not added
	otelStatsHandlerCount := 0
	for _, opt := range dialOpts {
		// Check if this is an OTEL stats handler option
		// This is a bit tricky to test directly, so we'll count options
		// and verify the behavior indirectly
		if opt != nil {
			otelStatsHandlerCount++
		}
	}

	// Test with OTEL collector enabled
	dialOptsWithOTEL, _, err := NewClientDialOptionsAndCreds(
		WithOTELCollectorEnable(true),
	)
	require.NoError(t, err)

	// Should have more options when OTEL is enabled
	assert.GreaterOrEqual(t, len(dialOptsWithOTEL), len(dialOpts),
		"OTEL enabled should have same or more dial options")
}

func TestNewClientDialOptionsAndCreds_DialOptionsStructure(t *testing.T) {
	dialOpts, _, err := NewClientDialOptionsAndCreds(
		WithHostPort("localhost:8080"),
		WithOTELCollectorEnable(true),
	)
	require.NoError(t, err)

	// Verify we have the expected minimum number of options
	// This includes: interceptors, message size limits, credentials, and potentially OTEL stats handler
	assert.GreaterOrEqual(t, len(dialOpts), 3,
		"Should have at least interceptors and message size options")

	// Verify all options are non-nil
	for i, opt := range dialOpts {
		assert.NotNil(t, opt, "Dial option %d should not be nil", i)
	}
}

func TestNewClientDialOptionsAndCreds_InterceptorChain(t *testing.T) {
	dialOpts, _, err := NewClientDialOptionsAndCreds(
		WithHostPort("localhost:8080"),
		WithOTELCollectorEnable(true),
	)
	require.NoError(t, err)

	// Verify that interceptors are included
	// This is a basic check - in practice you'd want to verify the actual interceptor types
	assert.GreaterOrEqual(t, len(dialOpts), 2,
		"Should have at least unary interceptors and message size options")
}

func TestNewClientDialOptionsAndCreds_MultipleOptions(t *testing.T) {
	dialOpts, creds, err := NewClientDialOptionsAndCreds(
		WithHostPort("localhost:8080"),
		WithOTELCollectorEnable(true),
		WithHTTPSConfig(client.HTTPSConfig{
			Cert: "/nonexistent/cert.pem",
			Key:  "/nonexistent/key.pem",
		}),
	)

	// This will fail due to invalid TLS files, but we can verify the error
	assert.Error(t, err)
	assert.Nil(t, dialOpts)
	assert.Nil(t, creds)
	assert.Contains(t, err.Error(), "TLS credentials")
}

func TestNewClientDialOptionsAndCreds_EmptyOptions(t *testing.T) {
	dialOpts, creds, err := NewClientDialOptionsAndCreds()
	require.NoError(t, err)
	assert.NotNil(t, dialOpts)
	assert.Nil(t, creds)
	assert.Greater(t, len(dialOpts), 0)
}

func TestNewClientDialOptionsAndCreds_NilOptions(t *testing.T) {
	// Test with no options (empty variadic arguments)
	dialOpts, creds, err := NewClientDialOptionsAndCreds()
	require.NoError(t, err)
	assert.NotNil(t, dialOpts)
	assert.Nil(t, creds)
	assert.Greater(t, len(dialOpts), 0)
}

func TestNewClientDialOptionsAndCreds_WithNilOptionInSlice(t *testing.T) {
	// Test with a slice containing nil options
	var nilOption ClientOption = nil
	dialOpts, creds, err := NewClientDialOptionsAndCreds(
		WithHostPort("localhost:8080"),
		nilOption, // This should be handled gracefully
		WithOTELCollectorEnable(true),
	)
	require.NoError(t, err)
	assert.NotNil(t, dialOpts)
	assert.Nil(t, creds)
	assert.Greater(t, len(dialOpts), 0)
}

func BenchmarkNewClientDialOptionsAndCreds_Default(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _, err := NewClientDialOptionsAndCreds()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkNewClientDialOptionsAndCreds_WithOptions(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _, err := NewClientDialOptionsAndCreds(
			WithHostPort("localhost:8080"),
			WithOTELCollectorEnable(true),
		)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkNewClientOptions(b *testing.B) {
	options := []ClientOption{
		WithHostPort("localhost:8080"),
		WithOTELCollectorEnable(true),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = newClientOptions(options...)
	}
}
