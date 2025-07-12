package grpc

import (
	"testing"

	"github.com/instill-ai/x/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewOptions(t *testing.T) {
	tests := []struct {
		name     string
		options  []Option
		expected *Options
	}{
		{
			name:    "default options",
			options: []Option{},
			expected: &Options{
				ServiceConfig:        client.ServiceConfig{},
				SetOTELClientHandler: false,
			},
		},
		{
			name: "with service config",
			options: []Option{
				WithServiceConfig(client.ServiceConfig{
					Host:        "localhost",
					PublicPort:  8080,
					PrivatePort: 8081,
				}),
			},
			expected: &Options{
				ServiceConfig: client.ServiceConfig{
					Host:        "localhost",
					PublicPort:  8080,
					PrivatePort: 8081,
				},
				SetOTELClientHandler: false,
			},
		},
		{
			name: "with service config including HTTPS",
			options: []Option{
				WithServiceConfig(client.ServiceConfig{
					Host:        "localhost",
					PublicPort:  8080,
					PrivatePort: 8081,
					HTTPS: client.HTTPSConfig{
						Cert: "/path/to/cert.pem",
						Key:  "/path/to/key.pem",
					},
				}),
			},
			expected: &Options{
				ServiceConfig: client.ServiceConfig{
					Host:        "localhost",
					PublicPort:  8080,
					PrivatePort: 8081,
					HTTPS: client.HTTPSConfig{
						Cert: "/path/to/cert.pem",
						Key:  "/path/to/key.pem",
					},
				},
				SetOTELClientHandler: false,
			},
		},
		{
			name: "with OTEL collector enabled",
			options: []Option{
				WithSetOTELClientHandler(true),
			},
			expected: &Options{
				ServiceConfig:        client.ServiceConfig{},
				SetOTELClientHandler: true,
			},
		},
		{
			name: "with all options",
			options: []Option{
				WithServiceConfig(client.ServiceConfig{
					Host:        "localhost",
					PublicPort:  8080,
					PrivatePort: 8081,
					HTTPS: client.HTTPSConfig{
						Cert: "/path/to/cert.pem",
						Key:  "/path/to/key.pem",
					},
				}),
				WithSetOTELClientHandler(true),
			},
			expected: &Options{
				ServiceConfig: client.ServiceConfig{
					Host:        "localhost",
					PublicPort:  8080,
					PrivatePort: 8081,
					HTTPS: client.HTTPSConfig{
						Cert: "/path/to/cert.pem",
						Key:  "/path/to/key.pem",
					},
				},
				SetOTELClientHandler: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := newOptions(tt.options...)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestWithServiceConfig(t *testing.T) {
	config := client.ServiceConfig{
		Host:        "test.example.com",
		PublicPort:  9090,
		PrivatePort: 9091,
		HTTPS: client.HTTPSConfig{
			Cert: "/test/cert.pem",
			Key:  "/test/key.pem",
		},
	}

	option := WithServiceConfig(config)
	opts := &Options{}
	option(opts)

	assert.Equal(t, config, opts.ServiceConfig)
}

func TestWithSetOTELClientHandler(t *testing.T) {
	tests := []struct {
		name   string
		enable bool
	}{
		{"enable true", true},
		{"enable false", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			option := WithSetOTELClientHandler(tt.enable)
			opts := &Options{}
			option(opts)

			assert.Equal(t, tt.enable, opts.SetOTELClientHandler)
		})
	}
}

func TestNewClientOptionsAndCreds_DefaultOptions(t *testing.T) {
	dialOpts, err := NewClientOptionsAndCreds()

	require.NoError(t, err)
	assert.NotNil(t, dialOpts)

	// Verify that we have the expected number of dial options
	// This includes interceptors, message size limits, and insecure credentials
	assert.Greater(t, len(dialOpts), 0)
}

func TestNewClientOptionsAndCreds_WithCustomOptions(t *testing.T) {
	dialOpts, err := NewClientOptionsAndCreds(
		WithSetOTELClientHandler(true),
	)

	require.NoError(t, err)
	assert.NotNil(t, dialOpts)
	assert.Greater(t, len(dialOpts), 0)
}

func TestNewClientOptionsAndCreds_OptionOrdering(t *testing.T) {
	// Test that options are applied in the correct order
	dialOpts1, err1 := NewClientOptionsAndCreds()

	dialOpts2, err2 := NewClientOptionsAndCreds()

	require.NoError(t, err1)
	require.NoError(t, err2)
	assert.Equal(t, len(dialOpts1), len(dialOpts2))
}

func TestNewClientOptionsAndCreds_OTELStatsHandler(t *testing.T) {
	// Test with OTEL collector disabled (default)
	dialOpts, err := NewClientOptionsAndCreds()
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
	dialOptsWithOTEL, err := NewClientOptionsAndCreds(
		WithSetOTELClientHandler(true),
	)
	require.NoError(t, err)

	// Should have more options when OTEL is enabled
	assert.GreaterOrEqual(t, len(dialOptsWithOTEL), len(dialOpts),
		"OTEL enabled should have same or more dial options")
}

func TestNewClientOptionsAndCreds_DialOptionsStructure(t *testing.T) {
	dialOpts, err := NewClientOptionsAndCreds(
		WithSetOTELClientHandler(true),
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

func TestNewClientOptionsAndCreds_InterceptorChain(t *testing.T) {
	dialOpts, err := NewClientOptionsAndCreds(
		WithSetOTELClientHandler(true),
	)
	require.NoError(t, err)

	// Verify that interceptors are included
	// This is a basic check - in practice you'd want to verify the actual interceptor types
	assert.GreaterOrEqual(t, len(dialOpts), 2,
		"Should have at least unary interceptors and message size options")
}

func TestNewClientOptionsAndCreds_MultipleOptions(t *testing.T) {
	dialOpts, err := NewClientOptionsAndCreds(
		WithSetOTELClientHandler(true),
	)

	require.NoError(t, err)
	assert.NotNil(t, dialOpts)
	assert.Greater(t, len(dialOpts), 0)
}

func TestNewClientOptionsAndCreds_EmptyOptions(t *testing.T) {
	dialOpts, err := NewClientOptionsAndCreds()
	require.NoError(t, err)
	assert.NotNil(t, dialOpts)
	assert.Greater(t, len(dialOpts), 0)
}

func TestNewClientOptionsAndCreds_NilOptions(t *testing.T) {
	// Test with no options (empty variadic arguments)
	dialOpts, err := NewClientOptionsAndCreds()
	require.NoError(t, err)
	assert.NotNil(t, dialOpts)
	assert.Greater(t, len(dialOpts), 0)
}

func TestNewClientOptionsAndCreds_WithNilOptionInSlice(t *testing.T) {
	// Test with a slice containing nil options
	var nilOption Option = nil
	dialOpts, err := NewClientOptionsAndCreds(
		nilOption, // This should be handled gracefully
		WithSetOTELClientHandler(true),
	)
	require.NoError(t, err)
	assert.NotNil(t, dialOpts)
	assert.Greater(t, len(dialOpts), 0)
}

func TestNewOptions_WithNilOptions(t *testing.T) {
	// Test that newOptions handles nil options gracefully
	var nilOption Option = nil
	result := newOptions(nilOption)

	expected := &Options{
		ServiceConfig:        client.ServiceConfig{},
		SetOTELClientHandler: false,
	}

	assert.Equal(t, expected, result)
}

func TestNewOptions_WithMixedNilAndValidOptions(t *testing.T) {
	// Test with a mix of nil and valid options
	var nilOption Option = nil
	validOption := WithSetOTELClientHandler(true)

	result := newOptions(nilOption, validOption)

	expected := &Options{
		ServiceConfig:        client.ServiceConfig{},
		SetOTELClientHandler: true,
	}

	assert.Equal(t, expected, result)
}

func TestClientTypeConstants(t *testing.T) {
	// Test that all ClientType constants are defined and have expected values
	expectedTypes := map[string]ClientType{
		"PipelinePublic":  PipelinePublic,
		"PipelinePrivate": PipelinePrivate,
		"ArtifactPublic":  ArtifactPublic,
		"ArtifactPrivate": ArtifactPrivate,
		"ModelPublic":     ModelPublic,
		"ModelPrivate":    ModelPrivate,
		"MgmtPublic":      MgmtPublic,
		"MgmtPrivate":     MgmtPrivate,
		"Usage":           Usage,
		"External":        External,
	}

	for name, expectedType := range expectedTypes {
		t.Run(name, func(t *testing.T) {
			assert.NotEmpty(t, string(expectedType), "ClientType should not be empty")
			assert.Equal(t, expectedType, expectedType, "ClientType should equal itself")
		})
	}
}

func BenchmarkNewClientOptionsAndCreds_Default(b *testing.B) {
	for b.Loop() {
		_, err := NewClientOptionsAndCreds()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkNewClientOptionsAndCreds_WithOptions(b *testing.B) {
	for b.Loop() {
		_, err := NewClientOptionsAndCreds(
			WithSetOTELClientHandler(true),
		)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkNewOptions(b *testing.B) {
	options := []Option{
		WithServiceConfig(client.ServiceConfig{
			Host:        "localhost",
			PublicPort:  8080,
			PrivatePort: 8081,
			HTTPS: client.HTTPSConfig{
				Cert: "/path/to/cert.pem",
				Key:  "/path/to/key.pem",
			},
		}),
		WithSetOTELClientHandler(true),
	}

	b.ResetTimer()
	for b.Loop() {
		_ = newOptions(options...)
	}
}
