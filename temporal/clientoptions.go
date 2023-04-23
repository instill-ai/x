package temporal

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	"go.temporal.io/sdk/client"

	"github.com/instill-ai/x/zapadapter"
)

// GetClientOption takes
// hostport: host:port of the Temporal server
// namespace: namespace to use
// logger: a zap logger
func GetClientOption(hostport string, namespace string, logger *zapadapter.ZapAdapter) (client.Options, error) {
	return client.Options{
		HostPort:  hostport,
		Namespace: namespace,
		Logger:    logger,
	}, nil
}

// GetTLSClientOption takes
// hostport: host:port of the Temporal server
// namespace: namespace to use
// logger: a zap logger
// serverRootCACert: path to the server's root CA cert
// clientCert: path to the client's cert
// clientKey: path to the client's key
// serverName: server name to use for verifying the server's certificate
// insecureSkipVerify: skip verification of the server's certificate and host name
func GetTLSClientOption(hostport string, namespace string, logger *zapadapter.ZapAdapter, serverRootCACert string, clientCert string, clientKey string, serverName string, insecureSkipVerify bool) (client.Options, error) {
	// Load client cert
	cert, err := tls.LoadX509KeyPair(clientCert, clientKey)
	if err != nil {
		return client.Options{}, fmt.Errorf("failed loading client cert and key: %w", err)
	}

	// Load server CA if given
	var serverCAPool *x509.CertPool
	if serverRootCACert != "" {
		serverCAPool = x509.NewCertPool()
		b, err := os.ReadFile(serverRootCACert)
		if err != nil {
			return client.Options{}, fmt.Errorf("failed reading server CA: %w", err)
		} else if !serverCAPool.AppendCertsFromPEM(b) {
			return client.Options{}, fmt.Errorf("server CA PEM file invalid")
		}
	}

	return client.Options{
		HostPort:  hostport,
		Namespace: namespace,
		Logger:    logger,
		ConnectionOptions: client.ConnectionOptions{
			TLS: &tls.Config{
				Certificates:       []tls.Certificate{cert},
				RootCAs:            serverCAPool,
				ServerName:         serverName,
				InsecureSkipVerify: true,
			},
		},
	}, nil
}
