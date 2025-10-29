//nolint:revive
package util

import (
	"crypto/tls"
	"crypto/x509"

	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// GetTLSConfig gets the TLS config for connection.
func GetTLSConfig(ds *storepb.DataSource) (*tls.Config, error) {
	if !ds.GetUseSsl() {
		return nil, nil
	}

	cfg := &tls.Config{}

	// Handle client certificates for mutual TLS authentication
	// Client certificates can be used with or without server verification
	if err := configureClientCertificates(ds, cfg); err != nil {
		return nil, err
	}

	// Handle server certificate verification
	if !ds.GetVerifyTlsCertificate() {
		// Certificate verification is disabled (default for backward compatibility)
		// This accepts any certificate presented by the server but still uses encryption
		cfg.InsecureSkipVerify = true
		return cfg, nil
	}

	// Server certificate verification is enabled
	// Set up the root CA pool for verifying the server's certificate
	var rootCertPool *x509.CertPool
	if ds.GetSslCa() == "" {
		// No custom CA provided, use system's default trusted CAs
		p, err := x509.SystemCertPool()
		if err != nil {
			return nil, err
		}
		rootCertPool = p
	} else {
		// Use the provided CA certificate
		rootCertPool = x509.NewCertPool()
		if ok := rootCertPool.AppendCertsFromPEM([]byte(ds.GetSslCa())); !ok {
			return nil, errors.Errorf("rootCertPool.AppendCertsFromPEM() failed to append server CA pem")
		}
	}

	cfg.RootCAs = rootCertPool

	// Expected behavior:
	// - InsecureSkipVerify = true disables Go's default verification
	// - VerifyPeerCertificate implements custom verification that handles intermediate certificates
	// - This pattern allows proper verification even when servers send certificates out of order
	cfg.InsecureSkipVerify = true
	cfg.VerifyPeerCertificate = CreateCertificateVerifier(rootCertPool, ds.GetHost())
	return cfg, nil
}

// CreateCertificateVerifier returns a verification function that properly handles intermediate certificates
// and validates the hostname matches the certificate.
// This is exported for use by database drivers that need custom certificate verification.
func CreateCertificateVerifier(rootCertPool *x509.CertPool, hostname string) func([][]byte, [][]*x509.Certificate) error {
	return func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
		if len(rawCerts) == 0 {
			return errors.Errorf("empty certificate to verify")
		}
		cert, err := x509.ParseCertificate(rawCerts[0])
		if err != nil {
			return err
		}

		// Build intermediate certificate pool from the certificate chain
		intermediatePool := x509.NewCertPool()
		for _, intermediate := range rawCerts[1:] {
			cert, err := x509.ParseCertificate(intermediate)
			if err != nil {
				return err
			}
			intermediatePool.AddCert(cert)
		}

		// Verify the certificate chain and hostname
		opts := x509.VerifyOptions{
			Roots:         rootCertPool,
			Intermediates: intermediatePool,
			DNSName:       hostname,
		}
		if _, err = cert.Verify(opts); err != nil {
			return errors.Wrap(err, "SSL cert failed to verify")
		}
		return nil
	}
}

// configureClientCertificates sets up client certificates for mutual TLS authentication
func configureClientCertificates(ds *storepb.DataSource, cfg *tls.Config) error {
	// Validate that both cert and key are provided together
	if (ds.GetSslCert() == "" && ds.GetSslKey() != "") || (ds.GetSslCert() != "" && ds.GetSslKey() == "") {
		return errors.Errorf("ssl-cert and ssl-key must be both set or unset")
	}

	// Configure client certificate if both cert and key are provided
	if ds.GetSslCert() != "" && ds.GetSslKey() != "" {
		certs, err := tls.X509KeyPair([]byte(ds.GetSslCert()), []byte(ds.GetSslKey()))
		if err != nil {
			return err
		}
		cfg.Certificates = []tls.Certificate{certs}
	}

	return nil
}

// SSLMode is the PGSSLMode type.
// https://www.postgresql.org/docs/current/libpq-connect.html#LIBPQ-CONNECT-SSLMODE
type SSLMode string

const (
	sslModeVerifyCA   SSLMode = "verify-ca"
	sslModeVerifyFull SSLMode = "verify-full"
)

// GetPGSSLMode is used only when SSL is enabled.
// We should consider allowing user to override this default in the future even if SSL is enabled.
func GetPGSSLMode(ds *storepb.DataSource) SSLMode {
	sslMode := sslModeVerifyFull
	if ds.GetSslCa() != "" {
		if ds.GetSshHost() != "" {
			sslMode = sslModeVerifyCA
		}
	}
	return sslMode
}
