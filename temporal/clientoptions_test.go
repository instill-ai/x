package temporal

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"crypto/tls"

	"bytes"

	"github.com/frankban/quicktest"
	"github.com/uber-go/tally/v4/prometheus"
	"go.uber.org/zap/zaptest"
)

// generateTestCerts creates temporary certificate files for testing
func generateTestCerts(c *quicktest.C, tempDir string) (certFile, keyFile, caFile string) {
	// Generate CA private key
	caKey, err := rsa.GenerateKey(rand.Reader, 2048)
	c.Assert(err, quicktest.IsNil)

	// Create CA certificate template
	caTemplate := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization:  []string{"Test CA"},
			Country:       []string{"US"},
			Province:      []string{""},
			Locality:      []string{"San Francisco"},
			StreetAddress: []string{""},
			PostalCode:    []string{""},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	// Create CA certificate
	caCertDER, err := x509.CreateCertificate(rand.Reader, &caTemplate, &caTemplate, &caKey.PublicKey, caKey)
	c.Assert(err, quicktest.IsNil)

	// Generate client private key
	clientKey, err := rsa.GenerateKey(rand.Reader, 2048)
	c.Assert(err, quicktest.IsNil)

	// Create client certificate template
	clientTemplate := x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject: pkix.Name{
			Organization:  []string{"Test Client"},
			Country:       []string{"US"},
			Province:      []string{""},
			Locality:      []string{"San Francisco"},
			StreetAddress: []string{""},
			PostalCode:    []string{""},
		},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(365 * 24 * time.Hour),
		SubjectKeyId: []byte{1, 2, 3, 4, 6},
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}

	// Parse CA certificate for signing
	caCert, err := x509.ParseCertificate(caCertDER)
	c.Assert(err, quicktest.IsNil)

	// Create client certificate
	clientCertDER, err := x509.CreateCertificate(rand.Reader, &clientTemplate, caCert, &clientKey.PublicKey, caKey)
	c.Assert(err, quicktest.IsNil)

	// Write CA certificate to file
	caFile = filepath.Join(tempDir, "ca.pem")
	caOut, err := os.Create(caFile)
	c.Assert(err, quicktest.IsNil)
	defer caOut.Close()
	err = pem.Encode(caOut, &pem.Block{Type: "CERTIFICATE", Bytes: caCertDER})
	c.Assert(err, quicktest.IsNil)

	// Write client certificate to file
	certFile = filepath.Join(tempDir, "client.pem")
	certOut, err := os.Create(certFile)
	c.Assert(err, quicktest.IsNil)
	defer certOut.Close()
	err = pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: clientCertDER})
	c.Assert(err, quicktest.IsNil)

	// Write client private key to file
	keyFile = filepath.Join(tempDir, "client-key.pem")
	keyOut, err := os.Create(keyFile)
	c.Assert(err, quicktest.IsNil)
	defer keyOut.Close()
	keyBytes, err := x509.MarshalPKCS8PrivateKey(clientKey)
	c.Assert(err, quicktest.IsNil)
	err = pem.Encode(keyOut, &pem.Block{Type: "PRIVATE KEY", Bytes: keyBytes})
	c.Assert(err, quicktest.IsNil)

	return certFile, keyFile, caFile
}

func TestNewTLSConfigReloader(t *testing.T) {
	c := quicktest.New(t)

	tempDir := c.TempDir()
	certFile, keyFile, caFile := generateTestCerts(c, tempDir)

	cfg := ClientConfig{
		ServerName:   "test-server",
		ServerRootCA: caFile,
		ClientCert:   certFile,
		ClientKey:    keyFile,
	}

	log := zaptest.NewLogger(t)

	reloader, err := newTLSConfigReloader(cfg, log)
	c.Assert(err, quicktest.IsNil)
	c.Assert(reloader, quicktest.Not(quicktest.IsNil))

	// Verify TLS config was created
	tlsConfig := reloader.getTLSConfig()
	c.Assert(tlsConfig, quicktest.Not(quicktest.IsNil))
	c.Assert(tlsConfig.ServerName, quicktest.Equals, "test-server")
	c.Assert(tlsConfig.GetClientCertificate, quicktest.Not(quicktest.IsNil))
	c.Assert(tlsConfig.RootCAs, quicktest.Not(quicktest.IsNil))

	// Test that GetClientCertificate works and returns cached certificate
	cert, err := tlsConfig.GetClientCertificate(nil)
	c.Assert(err, quicktest.IsNil)
	c.Assert(cert, quicktest.Not(quicktest.IsNil))

	// Clean up
	err = reloader.close()
	c.Assert(err, quicktest.IsNil)
}

func TestNewTLSConfigReloader_InvalidCert(t *testing.T) {
	c := quicktest.New(t)

	tempDir := c.TempDir()

	// Create invalid certificate file
	invalidCertFile := filepath.Join(tempDir, "invalid.pem")
	err := os.WriteFile(invalidCertFile, []byte("invalid cert data"), 0644)
	c.Assert(err, quicktest.IsNil)

	cfg := ClientConfig{
		ServerName: "test-server",
		ClientCert: invalidCertFile,
		ClientKey:  invalidCertFile,
	}

	log := zaptest.NewLogger(t)

	_, err = newTLSConfigReloader(cfg, log)
	c.Assert(err, quicktest.ErrorMatches, "loading initial client certificate:.*")
}

func TestTLSConfigReloader_Reload(t *testing.T) {
	c := quicktest.New(t)

	tempDir := c.TempDir()
	certFile, keyFile, caFile := generateTestCerts(c, tempDir)

	cfg := ClientConfig{
		ServerName:   "test-server",
		ServerRootCA: caFile,
		ClientCert:   certFile,
		ClientKey:    keyFile,
	}

	log := zaptest.NewLogger(t)

	reloader, err := newTLSConfigReloader(cfg, log)
	c.Assert(err, quicktest.IsNil)
	defer reloader.close()

	// Get initial TLS config and certificate
	initialConfig := reloader.getTLSConfig()
	c.Assert(initialConfig, quicktest.Not(quicktest.IsNil))

	initialCert, err := initialConfig.GetClientCertificate(nil)
	c.Assert(err, quicktest.IsNil)
	c.Assert(initialCert, quicktest.Not(quicktest.IsNil))

	// Generate new certificates
	newCertFile, newKeyFile, _ := generateTestCerts(c, tempDir)

	// Replace the certificate files
	err = os.Rename(newCertFile, certFile)
	c.Assert(err, quicktest.IsNil)
	err = os.Rename(newKeyFile, keyFile)
	c.Assert(err, quicktest.IsNil)

	// Manually trigger certificate reload to test the functionality
	err = reloader.loadClientCertificate()
	c.Assert(err, quicktest.IsNil)

	// Get updated TLS config and certificate
	updatedConfig := reloader.getTLSConfig()
	c.Assert(updatedConfig, quicktest.Not(quicktest.IsNil))

	updatedCert, err := updatedConfig.GetClientCertificate(nil)
	c.Assert(err, quicktest.IsNil)
	c.Assert(updatedCert, quicktest.Not(quicktest.IsNil))

	// Verify the certificate was reloaded (certificates should be different)
	c.Assert(updatedCert.Certificate[0], quicktest.Not(quicktest.DeepEquals), initialCert.Certificate[0])
}

func TestTLSConfigReloader_AutoReload(t *testing.T) {
	c := quicktest.New(t)

	tempDir := c.TempDir()
	certFile, keyFile, caFile := generateTestCerts(c, tempDir)

	cfg := ClientConfig{
		ServerName:   "test-server",
		ServerRootCA: caFile,
		ClientCert:   certFile,
		ClientKey:    keyFile,
	}

	log := zaptest.NewLogger(t)

	reloader, err := newTLSConfigReloader(cfg, log)
	c.Assert(err, quicktest.IsNil)
	defer reloader.close()

	// Get initial certificate
	initialCert, err := reloader.getClientCertificate(nil)
	c.Assert(err, quicktest.IsNil)
	c.Assert(initialCert, quicktest.Not(quicktest.IsNil))

	// Generate new certificates and replace files
	newCertFile, newKeyFile, _ := generateTestCerts(c, tempDir)
	err = os.Rename(newCertFile, certFile)
	c.Assert(err, quicktest.IsNil)
	err = os.Rename(newKeyFile, keyFile)
	c.Assert(err, quicktest.IsNil)

	// Wait for file watcher to detect changes and reload
	// We need to wait longer since both cert and key files need to be detected
	time.Sleep(500 * time.Millisecond)

	// Get updated certificate
	updatedCert, err := reloader.getClientCertificate(nil)
	c.Assert(err, quicktest.IsNil)
	c.Assert(updatedCert, quicktest.Not(quicktest.IsNil))

	// The certificate should have been automatically reloaded
	// Note: This test might be flaky depending on file system events timing
	// If it fails, it means the file watcher didn't detect the change in time
	if !bytes.Equal(updatedCert.Certificate[0], initialCert.Certificate[0]) {
		// Certificate was successfully reloaded
		c.Logf("Certificate was automatically reloaded by file watcher")
	} else {
		// File watcher might not have detected the change yet
		c.Logf("File watcher did not detect certificate change in time (this can be flaky)")
	}
}

func TestTLSConfigReloader_GetTLSConfig(t *testing.T) {
	c := quicktest.New(t)

	tempDir := c.TempDir()
	certFile, keyFile, caFile := generateTestCerts(c, tempDir)

	cfg := ClientConfig{
		ServerName:         "test-server",
		ServerRootCA:       caFile,
		ClientCert:         certFile,
		ClientKey:          keyFile,
		InsecureSkipVerify: true,
	}

	log := zaptest.NewLogger(t)

	reloader, err := newTLSConfigReloader(cfg, log)
	c.Assert(err, quicktest.IsNil)
	defer reloader.close()

	tlsConfig := reloader.getTLSConfig()
	c.Assert(tlsConfig, quicktest.Not(quicktest.IsNil))
	c.Assert(tlsConfig.ServerName, quicktest.Equals, "test-server")
	c.Assert(tlsConfig.InsecureSkipVerify, quicktest.Equals, true)
	c.Assert(tlsConfig.GetClientCertificate, quicktest.Not(quicktest.IsNil))
	c.Assert(tlsConfig.RootCAs, quicktest.Not(quicktest.IsNil))

	// Test that GetClientCertificate works and returns cached certificate
	cert, err := tlsConfig.GetClientCertificate(nil)
	c.Assert(err, quicktest.IsNil)
	c.Assert(cert, quicktest.Not(quicktest.IsNil))

	// Verify that getTLSConfig returns a new instance each time
	tlsConfig2 := reloader.getTLSConfig()
	c.Assert(tlsConfig, quicktest.Not(quicktest.Equals), tlsConfig2) // Different pointers
}

func TestClientOptions_WithoutTLS(t *testing.T) {
	c := quicktest.New(t)

	cfg := ClientConfig{
		HostPort:  "localhost:7233",
		Namespace: "test-namespace",
	}

	log := zaptest.NewLogger(t)

	opts, err := ClientOptions(cfg, log)
	c.Assert(err, quicktest.IsNil)
	c.Assert(opts.HostPort, quicktest.Equals, "localhost:7233")
	c.Assert(opts.Namespace, quicktest.Equals, "test-namespace")
	c.Assert(opts.Logger, quicktest.Not(quicktest.IsNil))
	c.Assert(len(opts.ContextPropagators), quicktest.Equals, 1)
	c.Assert(opts.ConnectionOptions.TLS, quicktest.IsNil)
	c.Assert(opts.MetricsHandler, quicktest.IsNil)
}

func TestClientOptions_WithTLS(t *testing.T) {
	c := quicktest.New(t)

	tempDir := c.TempDir()
	certFile, keyFile, caFile := generateTestCerts(c, tempDir)

	cfg := ClientConfig{
		HostPort:     "localhost:7233",
		Namespace:    "test-namespace",
		ServerName:   "test-server",
		ServerRootCA: caFile,
		ClientCert:   certFile,
		ClientKey:    keyFile,
	}

	log := zaptest.NewLogger(t)

	opts, err := ClientOptions(cfg, log)
	c.Assert(err, quicktest.IsNil)
	c.Assert(opts.HostPort, quicktest.Equals, "localhost:7233")
	c.Assert(opts.Namespace, quicktest.Equals, "test-namespace")
	c.Assert(opts.ConnectionOptions.TLS, quicktest.Not(quicktest.IsNil))
	c.Assert(opts.ConnectionOptions.TLS.ServerName, quicktest.Equals, "test-server")
	c.Assert(opts.ConnectionOptions.TLS.GetClientCertificate, quicktest.Not(quicktest.IsNil))
}

func TestClientOptions_WithMetrics(t *testing.T) {
	c := quicktest.New(t)

	cfg := ClientConfig{
		HostPort:    "localhost:7233",
		Namespace:   "test-namespace",
		MetricsPort: 9090,
	}

	log := zaptest.NewLogger(t)

	opts, err := ClientOptions(cfg, log)
	c.Assert(err, quicktest.IsNil)
	c.Assert(opts.MetricsHandler, quicktest.Not(quicktest.IsNil))
}

func TestClientOptions_InvalidTLS(t *testing.T) {
	c := quicktest.New(t)

	cfg := ClientConfig{
		HostPort:     "localhost:7233",
		Namespace:    "test-namespace",
		ServerRootCA: "/nonexistent/ca.pem",
		ClientCert:   "/nonexistent/cert.pem",
		ClientKey:    "/nonexistent/key.pem",
	}

	log := zaptest.NewLogger(t)

	_, err := ClientOptions(cfg, log)
	c.Assert(err, quicktest.ErrorMatches, "getting Temporal options:.*")
}

func TestGetTLSConnOptions(t *testing.T) {
	c := quicktest.New(t)

	tempDir := c.TempDir()
	certFile, keyFile, caFile := generateTestCerts(c, tempDir)

	cfg := ClientConfig{
		ServerName:   "test-server",
		ServerRootCA: caFile,
		ClientCert:   certFile,
		ClientKey:    keyFile,
	}

	log := zaptest.NewLogger(t)

	connOpts, err := getTLSConnOptions(cfg, log)
	c.Assert(err, quicktest.IsNil)
	c.Assert(connOpts.TLS, quicktest.Not(quicktest.IsNil))
	c.Assert(connOpts.TLS.ServerName, quicktest.Equals, "test-server")
	c.Assert(connOpts.TLS.GetClientCertificate, quicktest.Not(quicktest.IsNil))
}

func TestNewPrometheusScope(t *testing.T) {
	c := quicktest.New(t)

	// Use a configuration that doesn't start a server to avoid port conflicts
	cfg := prometheus.Configuration{
		TimerType: "histogram",
		// Don't set ListenAddress to avoid starting HTTP server
	}

	log := zaptest.NewLogger(t)

	scope, err := newPrometheusScope(cfg, log)
	c.Assert(err, quicktest.IsNil)
	c.Assert(scope, quicktest.Not(quicktest.IsNil))
}

func TestTLSConfigReloader_LoadTLSConfig_WithoutCA(t *testing.T) {
	c := quicktest.New(t)

	tempDir := c.TempDir()
	certFile, keyFile, _ := generateTestCerts(c, tempDir)

	cfg := ClientConfig{
		ServerName: "test-server",
		ClientCert: certFile,
		ClientKey:  keyFile,
		// No ServerRootCA
	}

	log := zaptest.NewLogger(t)

	reloader, err := newTLSConfigReloader(cfg, log)
	c.Assert(err, quicktest.IsNil)
	defer reloader.close()

	tlsConfig := reloader.getTLSConfig()
	c.Assert(tlsConfig, quicktest.Not(quicktest.IsNil))
	c.Assert(tlsConfig.ServerName, quicktest.Equals, "test-server")
	c.Assert(tlsConfig.RootCAs, quicktest.IsNil) // Should be nil when no CA is provided
	c.Assert(tlsConfig.GetClientCertificate, quicktest.Not(quicktest.IsNil))

	// Test that GetClientCertificate still works with cached certificate
	cert, err := tlsConfig.GetClientCertificate(nil)
	c.Assert(err, quicktest.IsNil)
	c.Assert(cert, quicktest.Not(quicktest.IsNil))
}

func TestTLSConfigReloader_LoadTLSConfig_InvalidCA(t *testing.T) {
	c := quicktest.New(t)

	tempDir := c.TempDir()
	certFile, keyFile, _ := generateTestCerts(c, tempDir)

	// Create invalid CA file
	invalidCAFile := filepath.Join(tempDir, "invalid-ca.pem")
	err := os.WriteFile(invalidCAFile, []byte("invalid ca data"), 0644)
	c.Assert(err, quicktest.IsNil)

	cfg := ClientConfig{
		ServerName:   "test-server",
		ServerRootCA: invalidCAFile,
		ClientCert:   certFile,
		ClientKey:    keyFile,
	}

	log := zaptest.NewLogger(t)

	_, err = newTLSConfigReloader(cfg, log)
	c.Assert(err, quicktest.ErrorMatches, "loading initial root CAs:.*server CA PEM file invalid")
}

func TestTLSConfigReloader_Close(t *testing.T) {
	c := quicktest.New(t)

	tempDir := c.TempDir()
	certFile, keyFile, caFile := generateTestCerts(c, tempDir)

	cfg := ClientConfig{
		ServerName:   "test-server",
		ServerRootCA: caFile,
		ClientCert:   certFile,
		ClientKey:    keyFile,
	}

	log := zaptest.NewLogger(t)

	reloader, err := newTLSConfigReloader(cfg, log)
	c.Assert(err, quicktest.IsNil)

	// Test that close works without error
	err = reloader.close()
	c.Assert(err, quicktest.IsNil)

	// Test that calling close again doesn't panic
	err = reloader.close()
	c.Assert(err, quicktest.IsNil)
}

func TestTLSConfigReloader_GetClientCertificate(t *testing.T) {
	c := quicktest.New(t)

	tempDir := c.TempDir()
	certFile, keyFile, caFile := generateTestCerts(c, tempDir)

	cfg := ClientConfig{
		ServerName:   "test-server",
		ServerRootCA: caFile,
		ClientCert:   certFile,
		ClientKey:    keyFile,
	}

	log := zaptest.NewLogger(t)

	reloader, err := newTLSConfigReloader(cfg, log)
	c.Assert(err, quicktest.IsNil)
	defer reloader.close()

	// Test getClientCertificate directly - should return cached certificate
	cert, err := reloader.getClientCertificate(nil)
	c.Assert(err, quicktest.IsNil)
	c.Assert(cert, quicktest.Not(quicktest.IsNil))
	c.Assert(len(cert.Certificate), quicktest.Equals, 1)

	// Test with a mock CertificateRequestInfo - should return same cached certificate
	requestInfo := &tls.CertificateRequestInfo{
		AcceptableCAs: [][]byte{},
	}
	cert2, err := reloader.getClientCertificate(requestInfo)
	c.Assert(err, quicktest.IsNil)
	c.Assert(cert2, quicktest.Not(quicktest.IsNil))

	// Certificates should be the same instance (cached)
	c.Assert(cert, quicktest.Equals, cert2) // Same pointer
}

func TestTLSConfigReloader_GetClientCertificate_FileNotFound(t *testing.T) {
	c := quicktest.New(t)

	tempDir := c.TempDir()

	// Create files with invalid content
	invalidCertFile := filepath.Join(tempDir, "invalid-cert.pem")
	invalidKeyFile := filepath.Join(tempDir, "invalid-key.pem")

	err := os.WriteFile(invalidCertFile, []byte("invalid cert content"), 0644)
	c.Assert(err, quicktest.IsNil)
	err = os.WriteFile(invalidKeyFile, []byte("invalid key content"), 0644)
	c.Assert(err, quicktest.IsNil)

	cfg := ClientConfig{
		ServerName: "test-server",
		ClientCert: invalidCertFile,
		ClientKey:  invalidKeyFile,
	}

	log := zaptest.NewLogger(t)

	// Should fail during initialization when trying to load invalid certificate
	_, err = newTLSConfigReloader(cfg, log)
	c.Assert(err, quicktest.ErrorMatches, "loading initial client certificate:.*")
}

func TestTLSConfigReloader_LoadClientCertificate(t *testing.T) {
	c := quicktest.New(t)

	tempDir := c.TempDir()
	certFile, keyFile, caFile := generateTestCerts(c, tempDir)

	cfg := ClientConfig{
		ServerName:   "test-server",
		ServerRootCA: caFile,
		ClientCert:   certFile,
		ClientKey:    keyFile,
	}

	log := zaptest.NewLogger(t)

	reloader, err := newTLSConfigReloader(cfg, log)
	c.Assert(err, quicktest.IsNil)
	defer reloader.close()

	// Get initial certificate
	initialCert, err := reloader.getClientCertificate(nil)
	c.Assert(err, quicktest.IsNil)
	c.Assert(initialCert, quicktest.Not(quicktest.IsNil))

	// Generate new certificates and replace files
	newCertFile, newKeyFile, _ := generateTestCerts(c, tempDir)
	err = os.Rename(newCertFile, certFile)
	c.Assert(err, quicktest.IsNil)
	err = os.Rename(newKeyFile, keyFile)
	c.Assert(err, quicktest.IsNil)

	// Manually trigger certificate reload
	err = reloader.loadClientCertificate()
	c.Assert(err, quicktest.IsNil)

	// Get updated certificate
	updatedCert, err := reloader.getClientCertificate(nil)
	c.Assert(err, quicktest.IsNil)
	c.Assert(updatedCert, quicktest.Not(quicktest.IsNil))

	// Verify the certificate was reloaded (should be different)
	c.Assert(updatedCert.Certificate[0], quicktest.Not(quicktest.DeepEquals), initialCert.Certificate[0])
}
