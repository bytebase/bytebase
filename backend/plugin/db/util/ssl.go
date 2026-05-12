//nolint:revive
package util

import (
	"crypto/tls"
	"crypto/x509"
	"io"
	"os"
	"path/filepath"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

const maxTLSMaterialFileSize int64 = 10 << 20

func HasTLSPath(ds *storepb.DataSource) bool {
	if ds == nil {
		return false
	}
	return ds.GetSslCaPath() != "" || ds.GetSslCertPath() != "" || ds.GetSslKeyPath() != ""
}

func ResolveTLSMaterial(ds *storepb.DataSource) (*storepb.DataSource, error) {
	if ds == nil {
		return nil, nil
	}
	resolved := proto.Clone(ds).(*storepb.DataSource)
	if !resolved.GetUseSsl() || !HasTLSPath(resolved) {
		return resolved, nil
	}

	if resolved.GetSslCaPath() != "" {
		content, err := readTLSPathFile("CA certificate", resolved.GetSslCaPath())
		if err != nil {
			return nil, err
		}
		resolved.SslCa = content
		resolved.SslCaPath = ""
	}
	if resolved.GetSslCertPath() != "" {
		content, err := readTLSPathFile("client certificate", resolved.GetSslCertPath())
		if err != nil {
			return nil, err
		}
		resolved.SslCert = content
		resolved.SslCertPath = ""
	}
	if resolved.GetSslKeyPath() != "" {
		content, err := readTLSPathFile("client key", resolved.GetSslKeyPath())
		if err != nil {
			return nil, err
		}
		resolved.SslKey = content
		resolved.SslKeyPath = ""
	}

	return resolved, nil
}

func readTLSPathFile(label, path string) (string, error) {
	if !filepath.IsAbs(path) {
		return "", errors.Errorf("%s path must be absolute", label)
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", errors.Errorf("failed to read %s file", label)
	}
	if !info.Mode().IsRegular() {
		return "", errors.Errorf("%s path must point to a regular file", label)
	}
	if info.Size() == 0 {
		return "", errors.Errorf("%s file is empty", label)
	}
	if info.Size() > maxTLSMaterialFileSize {
		return "", errors.Errorf("%s file is too large", label)
	}

	file, err := os.Open(path)
	if err != nil {
		return "", errors.Errorf("failed to read %s file", label)
	}
	defer file.Close()

	content, err := io.ReadAll(io.LimitReader(file, maxTLSMaterialFileSize+1))
	if err != nil {
		return "", errors.Errorf("failed to read %s file", label)
	}
	if len(content) == 0 {
		return "", errors.Errorf("%s file is empty", label)
	}
	if int64(len(content)) > maxTLSMaterialFileSize {
		return "", errors.Errorf("%s file is too large", label)
	}
	return string(content), nil
}

// GetTLSConfig gets the TLS config for connection.
// The datasource must already have path-backed TLS material resolved via ResolveTLSMaterial.
func GetTLSConfig(ds *storepb.DataSource) (*tls.Config, error) {
	if !ds.GetUseSsl() {
		return nil, nil
	}
	if HasTLSPath(ds) {
		return nil, errors.New("TLS material must be resolved before building TLS config")
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
	sslModeRequire    SSLMode = "require"
	sslModeVerifyCA   SSLMode = "verify-ca"
	sslModeVerifyFull SSLMode = "verify-full"
)

// GetPGSSLMode is used only when SSL is enabled.
// We should consider allowing user to override this default in the future even if SSL is enabled.
func GetPGSSLMode(ds *storepb.DataSource) SSLMode {
	if !ds.GetVerifyTlsCertificate() {
		return sslModeRequire
	}
	sslMode := sslModeVerifyFull
	if ds.GetSslCa() != "" {
		if ds.GetSshHost() != "" {
			sslMode = sslModeVerifyCA
		}
	}
	return sslMode
}

// ApplyPGTLSConfig applies Bytebase TLS settings to pgx primary and fallback TLS configs.
func ApplyPGTLSConfig(tlscfg *tls.Config, host string, fallbacks []*pgconn.FallbackConfig) {
	if tlscfg == nil {
		return
	}
	applyPGTLSConfigForHost(tlscfg, host, tlscfg)
	for _, fallback := range fallbacks {
		if fallback != nil && fallback.TLSConfig != nil {
			applyPGTLSConfigForHost(fallback.TLSConfig, fallback.Host, tlscfg)
		}
	}
}

func applyPGTLSConfigForHost(dst *tls.Config, host string, src *tls.Config) {
	if len(src.Certificates) > 0 {
		dst.Certificates = append([]tls.Certificate(nil), src.Certificates...)
	}
	if src.VerifyPeerCertificate != nil {
		dst.RootCAs = src.RootCAs
		dst.InsecureSkipVerify = src.InsecureSkipVerify
		dst.VerifyPeerCertificate = CreateCertificateVerifier(src.RootCAs, host)
	}
}
