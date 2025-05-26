package temporal

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/uber-go/tally/v4"
	"github.com/uber-go/tally/v4/prometheus"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/workflow"
	"go.uber.org/zap"

	prom "github.com/prometheus/client_golang/prometheus"
	sdktally "go.temporal.io/sdk/contrib/tally"

	"github.com/instill-ai/x/zapadapter"
)

// ClientConfig contains the configuration parameters for a Temporal client.
type ClientConfig struct {
	HostPort  string `koanf:"hostport"`
	Namespace string `koanf:"namespace"`
	Retention string `koanf:"retention"`

	MetricsPort int `koanf:"metricsport"` // Listener address for the Temporal metrics to be scraped

	// Secure communication config.
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

// tlsConfigReloader handles dynamic reloading of TLS certificates
type tlsConfigReloader struct {
	cfg        ClientConfig
	log        *zap.Logger
	mu         sync.RWMutex
	rootCAs    *x509.CertPool
	clientCert *tls.Certificate
	watcher    *fsnotify.Watcher
}

// newTLSConfigReloader creates a new TLS config reloader that watches for file changes
func newTLSConfigReloader(cfg ClientConfig, log *zap.Logger) (*tlsConfigReloader, error) {
	reloader := &tlsConfigReloader{
		cfg: cfg,
		log: log,
	}

	// Load initial root CAs and client certificate
	if err := reloader.loadRootCAs(); err != nil {
		return nil, fmt.Errorf("loading initial root CAs: %w", err)
	}

	if err := reloader.loadClientCertificate(); err != nil {
		return nil, fmt.Errorf("loading initial client certificate: %w", err)
	}

	// Set up file watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("creating file watcher: %w", err)
	}
	reloader.watcher = watcher

	// Watch certificate files
	if cfg.ClientCert != "" {
		if err := watcher.Add(cfg.ClientCert); err != nil {
			watcher.Close()
			return nil, fmt.Errorf("watching client cert file: %w", err)
		}
	}
	if cfg.ClientKey != "" {
		if err := watcher.Add(cfg.ClientKey); err != nil {
			watcher.Close()
			return nil, fmt.Errorf("watching client key file: %w", err)
		}
	}
	if cfg.ServerRootCA != "" {
		if err := watcher.Add(cfg.ServerRootCA); err != nil {
			watcher.Close()
			return nil, fmt.Errorf("watching server root CA file: %w", err)
		}
	}

	// Start watching for file changes
	go reloader.watchFiles()

	return reloader, nil
}

// loadRootCAs loads the root CA certificates
func (r *tlsConfigReloader) loadRootCAs() error {
	if r.cfg.ServerRootCA == "" {
		r.mu.Lock()
		r.rootCAs = nil
		r.mu.Unlock()
		return nil
	}

	serverCAPool := x509.NewCertPool()
	b, err := os.ReadFile(r.cfg.ServerRootCA)
	if err != nil {
		return fmt.Errorf("failed reading server CA: %w", err)
	}

	if !serverCAPool.AppendCertsFromPEM(b) {
		return fmt.Errorf("server CA PEM file invalid")
	}

	r.mu.Lock()
	r.rootCAs = serverCAPool
	r.mu.Unlock()

	r.log.Info("Root CA certificates loaded/reloaded")
	return nil
}

// loadClientCertificate loads and caches the client certificate
func (r *tlsConfigReloader) loadClientCertificate() error {
	cert, err := tls.LoadX509KeyPair(r.cfg.ClientCert, r.cfg.ClientKey)
	if err != nil {
		return fmt.Errorf("loading client cert and key: %w", err)
	}

	r.mu.Lock()
	r.clientCert = &cert
	r.mu.Unlock()

	r.log.Info("Client certificate loaded/reloaded")
	return nil
}

// getClientCertificate returns the cached client certificate for TLS handshake
func (r *tlsConfigReloader) getClientCertificate(info *tls.CertificateRequestInfo) (*tls.Certificate, error) {
	r.mu.RLock()
	cert := r.clientCert
	r.mu.RUnlock()

	if cert == nil {
		return nil, fmt.Errorf("no client certificate available")
	}

	r.log.Debug("Client certificate retrieved from cache")
	return cert, nil
}

// getTLSConfig returns the current TLS configuration with GetClientCertificate callback
func (r *tlsConfigReloader) getTLSConfig() *tls.Config {
	r.mu.RLock()
	rootCAs := r.rootCAs
	r.mu.RUnlock()

	tlsConfig := &tls.Config{
		GetClientCertificate: r.getClientCertificate,
		ServerName:           r.cfg.ServerName,
		InsecureSkipVerify:   r.cfg.InsecureSkipVerify,
		RootCAs:              rootCAs,
	}

	return tlsConfig
}

// watchFiles watches for file system events and reloads TLS config when files change
func (r *tlsConfigReloader) watchFiles() {
	for {
		select {
		case event, ok := <-r.watcher.Events:
			if !ok {
				return
			}

			// Check if it's a write or create event
			if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
				r.log.Info("TLS file changed, reloading configuration", zap.String("file", event.Name))

				// Add a small delay to ensure file write is complete
				time.Sleep(100 * time.Millisecond)

				// Reload the appropriate component based on which file changed
				if event.Name == r.cfg.ServerRootCA {
					if err := r.loadRootCAs(); err != nil {
						r.log.Error("Failed to reload root CA certificates", zap.Error(err))
					}
				} else if event.Name == r.cfg.ClientCert || event.Name == r.cfg.ClientKey {
					if err := r.loadClientCertificate(); err != nil {
						r.log.Error("Failed to reload client certificate", zap.Error(err))
					}
				}
			}

		case err, ok := <-r.watcher.Errors:
			if !ok {
				return
			}
			r.log.Error("File watcher error", zap.Error(err))
		}
	}
}

// close stops the file watcher
func (r *tlsConfigReloader) close() error {
	if r.watcher != nil {
		return r.watcher.Close()
	}
	return nil
}

// ClientOptions returns a Temporal client configuration based on the provided
// configuration.
func ClientOptions(cfg ClientConfig, log *zap.Logger) (client.Options, error) {
	opts := client.Options{
		HostPort:           cfg.HostPort,
		Namespace:          cfg.Namespace,
		Logger:             zapadapter.NewZapAdapter(log),
		ContextPropagators: []workflow.ContextPropagator{NewContextPropagator()},
	}

	if cfg.ServerRootCA != "" && cfg.ClientCert != "" && cfg.ClientKey != "" {
		connOpts, err := getTLSConnOptions(cfg, log)
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

func getTLSConnOptions(cfg ClientConfig, log *zap.Logger) (opts client.ConnectionOptions, err error) {
	reloader, err := newTLSConfigReloader(cfg, log)
	if err != nil {
		return opts, fmt.Errorf("creating TLS config reloader: %w", err)
	}

	opts.TLS = reloader.getTLSConfig()

	// Note: The reloader will continue running in the background.
	// In a production environment, you might want to store the reloader
	// reference somewhere to properly close it during application shutdown.

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
