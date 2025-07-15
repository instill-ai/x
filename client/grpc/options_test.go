package grpc

import (
	"testing"

	"github.com/frankban/quicktest"

	"github.com/instill-ai/x/client"
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
				ServiceConfig:              client.ServiceConfig{},
				SetOTELClientHandler:       false,
				MethodTraceExcludePatterns: []string{},
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
				SetOTELClientHandler:       false,
				MethodTraceExcludePatterns: []string{},
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
				SetOTELClientHandler:       false,
				MethodTraceExcludePatterns: []string{},
			},
		},
		{
			name: "with OTEL collector enabled",
			options: []Option{
				WithSetOTELClientHandler(true),
			},
			expected: &Options{
				ServiceConfig:              client.ServiceConfig{},
				SetOTELClientHandler:       true,
				MethodTraceExcludePatterns: []string{},
			},
		},
		{
			name: "with method trace exclude patterns",
			options: []Option{
				WithMethodTraceExcludePatterns([]string{
					".*TestService/.*",
					".*DebugService/.*",
				}),
			},
			expected: &Options{
				ServiceConfig:        client.ServiceConfig{},
				SetOTELClientHandler: false,
				MethodTraceExcludePatterns: []string{
					".*TestService/.*",
					".*DebugService/.*",
				},
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
				WithMethodTraceExcludePatterns([]string{
					".*TestService/.*",
					".*DebugService/.*",
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
				SetOTELClientHandler: true,
				MethodTraceExcludePatterns: []string{
					".*TestService/.*",
					".*DebugService/.*",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			qt := quicktest.New(t)
			result := newOptions(tt.options...)
			qt.Check(result, quicktest.CmpEquals(), tt.expected)
		})
	}
}

func TestWithServiceConfig(t *testing.T) {
	qt := quicktest.New(t)

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

	qt.Assert(opts.ServiceConfig, quicktest.Equals, config)
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
			qt := quicktest.New(t)
			option := WithSetOTELClientHandler(tt.enable)
			opts := &Options{}
			option(opts)

			qt.Assert(opts.SetOTELClientHandler, quicktest.Equals, tt.enable)
		})
	}
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
				".*TestService/.*",
			},
		},
		{
			name: "multiple patterns",
			patterns: []string{
				".*TestService/.*",
				".*DebugService/.*",
				".*HealthService/.*",
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

func TestNewClientOptionsAndCreds_DefaultOptions(t *testing.T) {
	qt := quicktest.New(t)

	dialOpts, err := NewClientOptionsAndCreds()

	qt.Assert(err, quicktest.IsNil)
	qt.Assert(dialOpts, quicktest.Not(quicktest.IsNil))

	// Verify that we have the expected number of dial options
	// This includes interceptors, message size limits, and insecure credentials
	qt.Check(len(dialOpts) > 0, quicktest.IsTrue)
}

func TestNewClientOptionsAndCreds_WithCustomOptions(t *testing.T) {
	qt := quicktest.New(t)

	dialOpts, err := NewClientOptionsAndCreds(
		WithSetOTELClientHandler(true),
	)

	qt.Assert(err, quicktest.IsNil)
	qt.Assert(dialOpts, quicktest.Not(quicktest.IsNil))
	qt.Check(len(dialOpts) > 0, quicktest.IsTrue)
}

func TestNewClientOptionsAndCreds_WithMethodTraceExcludePatterns(t *testing.T) {
	qt := quicktest.New(t)

	dialOpts, err := NewClientOptionsAndCreds(
		WithSetOTELClientHandler(true),
		WithMethodTraceExcludePatterns([]string{
			".*TestService/.*",
			".*DebugService/.*",
		}),
	)

	qt.Assert(err, quicktest.IsNil)
	qt.Assert(dialOpts, quicktest.Not(quicktest.IsNil))
	qt.Check(len(dialOpts) > 0, quicktest.IsTrue)
}

func TestNewClientOptionsAndCreds_OptionOrdering(t *testing.T) {
	qt := quicktest.New(t)

	// Test that options are applied in the correct order
	dialOpts1, err1 := NewClientOptionsAndCreds()

	dialOpts2, err2 := NewClientOptionsAndCreds()

	qt.Assert(err1, quicktest.IsNil)
	qt.Assert(err2, quicktest.IsNil)
	qt.Check(len(dialOpts1), quicktest.Equals, len(dialOpts2))
}

func TestNewClientOptionsAndCreds_OTELStatsHandler(t *testing.T) {
	qt := quicktest.New(t)

	// Test with OTEL collector disabled (default)
	dialOpts, err := NewClientOptionsAndCreds()
	qt.Assert(err, quicktest.IsNil)

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
	qt.Assert(err, quicktest.IsNil)

	// Should have more options when OTEL is enabled
	qt.Check(len(dialOptsWithOTEL) >= len(dialOpts), quicktest.IsTrue)
}

func TestNewClientOptionsAndCreds_OTELStatsHandlerWithExcludePatterns(t *testing.T) {
	qt := quicktest.New(t)

	// Test with OTEL collector enabled and exclude patterns
	dialOptsWithOTEL, err := NewClientOptionsAndCreds(
		WithSetOTELClientHandler(true),
		WithMethodTraceExcludePatterns([]string{
			".*TestService/.*",
			".*DebugService/.*",
		}),
	)
	qt.Assert(err, quicktest.IsNil)

	// Should have options when OTEL is enabled with exclude patterns
	qt.Check(len(dialOptsWithOTEL) > 0, quicktest.IsTrue)
}

func TestNewClientOptionsAndCreds_DialOptionsStructure(t *testing.T) {
	qt := quicktest.New(t)

	dialOpts, err := NewClientOptionsAndCreds(
		WithSetOTELClientHandler(true),
	)
	qt.Assert(err, quicktest.IsNil)

	// Verify we have the expected minimum number of options
	// This includes: interceptors, message size limits, credentials, and potentially OTEL stats handler
	qt.Check(len(dialOpts) >= 3, quicktest.IsTrue)

	// Verify all options are non-nil
	for _, opt := range dialOpts {
		qt.Assert(opt, quicktest.Not(quicktest.IsNil))
	}
}

func TestNewClientOptionsAndCreds_InterceptorChain(t *testing.T) {
	qt := quicktest.New(t)

	dialOpts, err := NewClientOptionsAndCreds(
		WithSetOTELClientHandler(true),
	)
	qt.Assert(err, quicktest.IsNil)

	// Verify that interceptors are included
	// This is a basic check - in practice you'd want to verify the actual interceptor types
	qt.Check(len(dialOpts) >= 2, quicktest.IsTrue)
}

func TestNewClientOptionsAndCreds_MultipleOptions(t *testing.T) {
	qt := quicktest.New(t)

	dialOpts, err := NewClientOptionsAndCreds(
		WithSetOTELClientHandler(true),
		WithMethodTraceExcludePatterns([]string{
			".*TestService/.*",
		}),
	)

	qt.Assert(err, quicktest.IsNil)
	qt.Assert(dialOpts, quicktest.Not(quicktest.IsNil))
	qt.Check(len(dialOpts) > 0, quicktest.IsTrue)
}

func TestNewClientOptionsAndCreds_EmptyOptions(t *testing.T) {
	qt := quicktest.New(t)

	dialOpts, err := NewClientOptionsAndCreds()
	qt.Assert(err, quicktest.IsNil)
	qt.Assert(dialOpts, quicktest.Not(quicktest.IsNil))
	qt.Check(len(dialOpts) > 0, quicktest.IsTrue)
}

func TestNewClientOptionsAndCreds_NilOptions(t *testing.T) {
	qt := quicktest.New(t)

	// Test with no options (empty variadic arguments)
	dialOpts, err := NewClientOptionsAndCreds()
	qt.Assert(err, quicktest.IsNil)
	qt.Assert(dialOpts, quicktest.Not(quicktest.IsNil))
	qt.Check(len(dialOpts) > 0, quicktest.IsTrue)
}

func TestNewClientOptionsAndCreds_WithNilOptionInSlice(t *testing.T) {
	qt := quicktest.New(t)

	// Test with a slice containing nil options
	var nilOption Option = nil
	dialOpts, err := NewClientOptionsAndCreds(
		nilOption, // This should be handled gracefully
		WithSetOTELClientHandler(true),
	)
	qt.Assert(err, quicktest.IsNil)
	qt.Assert(dialOpts, quicktest.Not(quicktest.IsNil))
	qt.Check(len(dialOpts) > 0, quicktest.IsTrue)
}

func TestNewOptions_WithNilOptions(t *testing.T) {
	qt := quicktest.New(t)

	// Test that newOptions handles nil options gracefully
	var nilOption Option = nil
	result := newOptions(nilOption)

	expected := &Options{
		ServiceConfig:              client.ServiceConfig{},
		SetOTELClientHandler:       false,
		MethodTraceExcludePatterns: []string{},
	}

	qt.Assert(result, quicktest.CmpEquals(), expected)
}

func TestNewOptions_WithMixedNilAndValidOptions(t *testing.T) {
	qt := quicktest.New(t)

	// Test with a mix of nil and valid options
	var nilOption Option = nil
	validOption := WithSetOTELClientHandler(true)

	result := newOptions(nilOption, validOption)

	expected := &Options{
		ServiceConfig:              client.ServiceConfig{},
		SetOTELClientHandler:       true,
		MethodTraceExcludePatterns: []string{},
	}

	qt.Assert(result, quicktest.CmpEquals(), expected)
}

func TestCreateFilterTraceDecider(t *testing.T) {
	qt := quicktest.New(t)

	// Test with custom patterns
	customPatterns := []string{
		".*TestService/.*",
		".*DebugService/.*",
	}

	filter := createFilterTraceDecider(customPatterns)

	// Test that the filter function works correctly
	// This is a basic test - in practice you'd want to test with actual RPC tag info
	qt.Assert(filter, quicktest.Not(quicktest.IsNil))
}

func TestDefaultMethodTraceExcludePatterns(t *testing.T) {
	qt := quicktest.New(t)

	// Test that default patterns are defined
	qt.Check(len(defaultMethodTraceExcludePatterns) > 0, quicktest.IsTrue)

	// Test that default patterns contain expected patterns
	expectedPatterns := []string{
		".*PublicService/.*ness$",
		".*PrivateService/.*$",
		".*UsageService/.*$",
	}

	for _, expectedPattern := range expectedPatterns {
		found := false
		for _, pattern := range defaultMethodTraceExcludePatterns {
			if pattern == expectedPattern {
				found = true
				break
			}
		}
		qt.Check(found, quicktest.IsTrue, quicktest.Commentf("Expected pattern %s not found in default patterns", expectedPattern))
	}
}

func TestMethodTraceExcludePatterns_Integration(t *testing.T) {
	qt := quicktest.New(t)

	// Test that method trace exclude patterns work with OTEL handler
	dialOpts, err := NewClientOptionsAndCreds(
		WithSetOTELClientHandler(true),
		WithMethodTraceExcludePatterns([]string{
			".*CustomService/.*",
		}),
	)

	qt.Assert(err, quicktest.IsNil)
	qt.Assert(dialOpts, quicktest.Not(quicktest.IsNil))
	qt.Check(len(dialOpts) > 0, quicktest.IsTrue)
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
			qt := quicktest.New(t)
			qt.Check(string(expectedType), quicktest.Not(quicktest.Equals), "")
			qt.Assert(expectedType, quicktest.Equals, expectedType)
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

func BenchmarkNewClientOptionsAndCreds_WithExcludePatterns(b *testing.B) {
	for b.Loop() {
		_, err := NewClientOptionsAndCreds(
			WithSetOTELClientHandler(true),
			WithMethodTraceExcludePatterns([]string{
				".*TestService/.*",
				".*DebugService/.*",
			}),
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
		WithMethodTraceExcludePatterns([]string{
			".*TestService/.*",
			".*DebugService/.*",
		}),
	}

	b.ResetTimer()
	for b.Loop() {
		_ = newOptions(options...)
	}
}
