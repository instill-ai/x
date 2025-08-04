package temporal

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"time"

	"github.com/uber-go/tally/v4"
	"github.com/uber-go/tally/v4/prometheus"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/workflow"
	"go.uber.org/zap"

	prom "github.com/prometheus/client_golang/prometheus"
	sdktally "go.temporal.io/sdk/contrib/tally"

	logx "github.com/instill-ai/x/log"
)

// ClientConfig contains the configuration parameters for a Temporal client.
type ClientConfig struct {
	HostPort  string `koanf:"hostport"`
	Namespace string `koanf:"namespace"`
	Retention string `koanf:"retention"`

	MetricsPort int `koanf:"metricsport"` // Listener address for the Temporal metrics to be scraped

	// Temporal Cloud specific config.
	UseTemporalCloud bool   `koanf:"usetemporalcloud"` // Use Temporal Cloud
	APIKey           string `koanf:"apikey"`           // API key to use for authentication

	// Secure communication config.
	// DEPRECATED: These configurations are deprecated in favor of using the API key
	// for authentication. They are maintained for backward compatibility and will
	// be removed in a future version.
	ServerName   string `koanf:"servername"`   // Server name to use for verifying the server certificate and as metrics prefix
	ServerRootCA string `koanf:"serverrootca"` // Path to the server root CA certificate
	ClientCert   string `koanf:"clientcert"`   // Path to the client certificate
	ClientKey    string `koanf:"clientkey"`    // Path to the client

	// InsecureSkipVerify controls whether a client verifies the server's
	// certificate chain and host name. If InsecureSkipVerify is true, crypto/tls
	// accepts any certificate presented by the server and any host name in that
	// certificate. In this mode, TLS is susceptible to machine-in-the-middle
	// attacks unless custom verification is used. This should be used only for
	// testing or in combination with VerifyConnection or VerifyPeerCertificate.
	InsecureSkipVerify bool `koanf:"insecureskipverify"`
}

// ClientOptions returns a Temporal client configuration based on the provided
// configuration.
func ClientOptions(cfg ClientConfig, log *zap.Logger) (client.Options, error) {
	opts := client.Options{
		HostPort:           cfg.HostPort,
		Namespace:          cfg.Namespace,
		Logger:             logx.NewZapAdapter(log),
		ContextPropagators: []workflow.ContextPropagator{NewContextPropagator()},
	}
	if cfg.UseTemporalCloud {
		opts.ConnectionOptions.TLS = &tls.Config{}
		opts.Credentials = client.NewAPIKeyDynamicCredentials(func(ctx context.Context) (string, error) {
			// TODO: use dynamic api key here in case we want to rotate the api key in the future
			return cfg.APIKey, nil
		})
	}

	if cfg.ServerRootCA != "" && cfg.ClientCert != "" && cfg.ClientKey != "" {
		connOpts, err := getTLSConnOptions(cfg)
		if err != nil {
			return opts, fmt.Errorf("getting Temporal options: %w", err)
		}

		opts.ConnectionOptions = connOpts
	}

	if cfg.MetricsPort != 0 {
		ps, err := newPrometheusScope(prometheus.Configuration{
			ListenAddress: fmt.Sprintf("0.0.0.0:%d", cfg.MetricsPort),
			TimerType:     "histogram",
		}, log)
		if err != nil {
			return opts, fmt.Errorf("creating Prometheus metrics scope: %w", err)
		}

		opts.MetricsHandler = sdktally.NewMetricsHandler(ps)
	}

	return opts, nil
}

func getTLSConnOptions(cfg ClientConfig) (opts client.ConnectionOptions, err error) {
	cert, err := tls.LoadX509KeyPair(cfg.ClientCert, cfg.ClientKey)
	if err != nil {
		return opts, fmt.Errorf("loading client cert and key: %w", err)
	}

	opts.TLS = &tls.Config{
		Certificates:       []tls.Certificate{cert},
		ServerName:         cfg.ServerName,
		InsecureSkipVerify: cfg.InsecureSkipVerify,
	}

	if cfg.ServerRootCA != "" {
		serverCAPool := x509.NewCertPool()
		b, err := os.ReadFile(cfg.ServerRootCA)
		if err != nil {
			return opts, fmt.Errorf("failed reading server CA: %w", err)
		}

		if !serverCAPool.AppendCertsFromPEM(b) {
			return opts, fmt.Errorf("server CA PEM file invalid")
		}

		opts.TLS.RootCAs = serverCAPool
	}

	return opts, nil
}

func newPrometheusScope(c prometheus.Configuration, log *zap.Logger) (tally.Scope, error) {
	reporter, err := c.NewReporter(
		prometheus.ConfigurationOptions{
			Registry: prom.NewRegistry(),
			OnError: func(err error) {
				log.Error("Error in prometheus reporter", zap.Error(err))
			},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("creating prometheus reporter: %w", err)
	}

	// By convention, metrics will use snake_case.
	scopeOpts := tally.ScopeOptions{
		CachedReporter:  reporter,
		Separator:       prometheus.DefaultSeparator,
		SanitizeOptions: &sdktally.PrometheusSanitizeOptions,
		// A prefix can be set here, but metrics are already grouped by namespace.
	}
	scope, _ := tally.NewRootScope(scopeOpts, time.Second)
	scope = sdktally.NewPrometheusNamingScope(scope)

	log.Info("Prometheus metrics scope created")
	return scope, nil
}
