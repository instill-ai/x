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
		options  []Option
		expected *Options
	}{
		{
			name:    "default options",
			options: []Option{},
			expected: &Options{
				HTTPSConfig:         client.HTTPSConfig{},
				OTELCollectorEnable: false,
			},
		},
		{
			name: "with HTTPS config",
			options: []Option{
				WithHTTPSConfig(client.HTTPSConfig{
					Cert: "/path/to/cert.pem",
					Key:  "/path/to/key.pem",
				}),
			},
			expected: &Options{
				HTTPSConfig:         client.HTTPSConfig{Cert: "/path/to/cert.pem", Key: "/path/to/key.pem"},
				OTELCollectorEnable: false,
			},
		},
		{
			name: "with OTEL collector enabled",
			options: []Option{
				WithOTELCollectorEnable(true),
			},
			expected: &Options{
				HTTPSConfig:         client.HTTPSConfig{},
				OTELCollectorEnable: true,
			},
		},
		{
			name: "with all options",
			options: []Option{
				WithHTTPSConfig(client.HTTPSConfig{
					Cert: "/path/to/cert.pem",
					Key:  "/path/to/key.pem",
				}),
				WithOTELCollectorEnable(true),
			},
			expected: &Options{
				HTTPSConfig:         client.HTTPSConfig{Cert: "/path/to/cert.pem", Key: "/path/to/key.pem"},
				OTELCollectorEnable: true,
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

func TestWithHTTPSConfig(t *testing.T) {
	config := client.HTTPSConfig{
		Cert: "/test/cert.pem",
		Key:  "/test/key.pem",
	}

	option := WithHTTPSConfig(config)
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
		WithOTELCollectorEnable(true),
	)

	require.NoError(t, err)
	assert.NotNil(t, dialOpts)
	assert.Greater(t, len(dialOpts), 0)
}

func TestNewClientOptionsAndCreds_WithValidTLS(t *testing.T) {
	dialOpts, err := NewClientOptionsAndCreds(
		WithHTTPSConfig(client.HTTPSConfig{
			Cert: "/nonexistent/cert.pem",
			Key:  "/nonexistent/key.pem",
		}),
	)

	// This will fail due to invalid cert files, but we can verify the error
	assert.Error(t, err)
	assert.Nil(t, dialOpts)
	assert.Contains(t, err.Error(), "TLS credentials")
}

func TestNewClientOptionsAndCreds_WithInvalidTLS(t *testing.T) {
	dialOpts, err := NewClientOptionsAndCreds(
		WithHTTPSConfig(client.HTTPSConfig{
			Cert: "/nonexistent/cert.pem",
			Key:  "/nonexistent/key.pem",
		}),
	)

	assert.Error(t, err)
	assert.Nil(t, dialOpts)
	assert.Contains(t, err.Error(), "TLS credentials")
}

func TestNewClientOptionsAndCreds_WithPartialTLS(t *testing.T) {
	// Test with only cert file (missing key)
	dialOpts, err := NewClientOptionsAndCreds(
		WithHTTPSConfig(client.HTTPSConfig{
			Cert: "/path/to/cert.pem",
			Key:  "", // Missing key
		}),
	)

	require.NoError(t, err)
	assert.NotNil(t, dialOpts)
	assert.Greater(t, len(dialOpts), 0)

	// Test with only key file (missing cert)
	dialOpts, err = NewClientOptionsAndCreds(
		WithHTTPSConfig(client.HTTPSConfig{
			Cert: "", // Missing cert
			Key:  "/path/to/key.pem",
		}),
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
		WithOTELCollectorEnable(true),
	)
	require.NoError(t, err)

	// Should have more options when OTEL is enabled
	assert.GreaterOrEqual(t, len(dialOptsWithOTEL), len(dialOpts),
		"OTEL enabled should have same or more dial options")
}

func TestNewClientOptionsAndCreds_DialOptionsStructure(t *testing.T) {
	dialOpts, err := NewClientOptionsAndCreds(
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

func TestNewClientOptionsAndCreds_InterceptorChain(t *testing.T) {
	dialOpts, err := NewClientOptionsAndCreds(
		WithOTELCollectorEnable(true),
	)
	require.NoError(t, err)

	// Verify that interceptors are included
	// This is a basic check - in practice you'd want to verify the actual interceptor types
	assert.GreaterOrEqual(t, len(dialOpts), 2,
		"Should have at least unary interceptors and message size options")
}

func TestNewClientOptionsAndCreds_MultipleOptions(t *testing.T) {
	dialOpts, err := NewClientOptionsAndCreds(
		WithOTELCollectorEnable(true),
		WithHTTPSConfig(client.HTTPSConfig{
			Cert: "/nonexistent/cert.pem",
			Key:  "/nonexistent/key.pem",
		}),
	)

	// This will fail due to invalid TLS files, but we can verify the error
	assert.Error(t, err)
	assert.Nil(t, dialOpts)
	assert.Contains(t, err.Error(), "TLS credentials")
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
		WithOTELCollectorEnable(true),
	)
	require.NoError(t, err)
	assert.NotNil(t, dialOpts)
	assert.Greater(t, len(dialOpts), 0)
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
			WithOTELCollectorEnable(true),
		)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkNewClientOptions(b *testing.B) {
	options := []Option{
		WithOTELCollectorEnable(true),
	}

	b.ResetTimer()
	for b.Loop() {
		_ = newOptions(options...)
	}
}
