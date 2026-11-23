//nolint:revive
package util

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/sync/singleflight"
	"google.golang.org/protobuf/proto"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

const maxTLSMaterialFileSize int64 = 10 << 20

func hasTLSPath(ds *storepb.DataSource) bool {
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
	if !resolved.GetUseSsl() || !hasTLSPath(resolved) {
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
	if hasTLSPath(ds) {
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
	cfg.VerifyPeerCertificate = createCertificateVerifier(rootCertPool, ds.GetHost())
	return cfg, nil
}

// getTLSConfigForHost returns the TLS config for ds with SNI and certificate
// hostname verification bound to host when non-empty, so SRV and multi-host
// drivers verify each dialed server against its own hostname rather than ds.GetHost().
func getTLSConfigForHost(ds *storepb.DataSource, host string) (*tls.Config, error) {
	cfg, err := GetTLSConfig(ds)
	if err != nil || cfg == nil {
		return nil, err
	}
	bare := strings.TrimPrefix(strings.TrimSuffix(host, "]"), "[")
	if bare != "" {
		cfg.ServerName = bare
		if cfg.VerifyPeerCertificate != nil {
			cfg.VerifyPeerCertificate = createCertificateVerifier(cfg.RootCAs, bare)
		}
	}
	return cfg, nil
}

// createCertificateVerifier returns a verification function that properly handles intermediate certificates
// and validates the hostname matches the certificate.
func createCertificateVerifier(rootCertPool *x509.CertPool, hostname string) func([][]byte, [][]*x509.Certificate) error {
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
	if err := ValidateTLSCertKeyPair(ds); err != nil {
		return err
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

// getPGTLSConfigForHost returns Bytebase's TLS settings for one host the pg
// driver dials: the client certificate and CA from the data source, the
// verification mode it chose, and a server name and hostname verifier bound to
// host rather than to the data source's first host.
func getPGTLSConfigForHost(ds *storepb.DataSource, host string) (*tls.Config, error) {
	tlscfg, err := GetTLSConfig(ds)
	if err != nil || tlscfg == nil {
		return nil, err
	}
	// PostgreSQL proxies use SNI for routing even when certificate verification
	// is handled by our custom verifier or disabled by the user.
	tlscfg.ServerName = host
	if tlscfg.VerifyPeerCertificate != nil {
		tlscfg.VerifyPeerCertificate = createCertificateVerifier(tlscfg.RootCAs, host)
	}
	return tlscfg, nil
}

// rdsCertBundleURL is a variable so tests can point it at a local server.
var rdsCertBundleURL = "https://truststore.pki.rds.amazonaws.com/global/global-bundle.pem"

// rdsCertFetchTimeout bounds the download so a stalled fetch cannot hold a
// connection attempt open indefinitely.
const rdsCertFetchTimeout = 30 * time.Second

// maxRDSCertBundleSize bounds the downloaded bundle so a misbehaving endpoint
// cannot stream unbounded data into memory (the real global bundle is ~165 KB).
const maxRDSCertBundleSize int64 = 4 << 20

var (
	rdsCertPool  atomic.Pointer[x509.CertPool]
	rdsCertGroup singleflight.Group
)

func getRDSCertPool(ctx context.Context) (*x509.CertPool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rdsCertBundleURL, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build request for rds cert")
	}

	client := &http.Client{Timeout: rdsCertFetchTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Without this an error page would reach AppendCertsFromPEM and surface as
	// "failed to parse RDS CA certificates", hiding the real cause.
	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("failed to download RDS CA certificates: %s", resp.Status)
	}

	pem, err := io.ReadAll(io.LimitReader(resp.Body, maxRDSCertBundleSize+1))
	if err != nil {
		return nil, err
	}
	if int64(len(pem)) > maxRDSCertBundleSize {
		return nil, errors.Errorf("RDS CA bundle exceeds %d bytes", maxRDSCertBundleSize)
	}

	rootCertPool := x509.NewCertPool()
	if ok := rootCertPool.AppendCertsFromPEM(pem); !ok {
		return nil, errors.Errorf("failed to parse RDS CA certificates")
	}
	return rootCertPool, nil
}

// cachedRDSCertPool returns the AWS RDS CA bundle, downloading it once per
// process. Only successes are cached, so a transient failure can still recover.
func cachedRDSCertPool(ctx context.Context) (*x509.CertPool, error) {
	if pool := rdsCertPool.Load(); pool != nil {
		return pool, nil
	}
	// The shared fetch does not inherit any caller's context, so the caller
	// that starts it cannot cancel it for others; each caller waits on its own.
	ch := rdsCertGroup.DoChan("", func() (any, error) {
		if pool := rdsCertPool.Load(); pool != nil {
			return pool, nil
		}
		fetchCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), rdsCertFetchTimeout)
		defer cancel()
		pool, err := getRDSCertPool(fetchCtx)
		if err != nil {
			return nil, err
		}
		rdsCertPool.Store(pool)
		return pool, nil
	})
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case res := <-ch:
		if res.Err != nil {
			return nil, res.Err
		}
		pool, ok := res.Val.(*x509.CertPool)
		if !ok {
			return nil, errors.Errorf("unexpected RDS cert pool type %T", res.Val)
		}
		return pool, nil
	}
}

// RDSRootCAs returns the root CA pool for an RDS IAM connection, preferring
// an operator-supplied SslCa so deployments without outbound network access work.
func RDSRootCAs(ctx context.Context, dataSource *storepb.DataSource) (*x509.CertPool, error) {
	if ca := dataSource.GetSslCa(); ca != "" {
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM([]byte(ca)) {
			return nil, errors.Errorf("failed to parse ssl_ca certificates")
		}
		return pool, nil
	}
	return cachedRDSCertPool(ctx)
}

// RDSTLSConfig builds the TLS config for an RDS IAM connection. IAM auth always
// requires TLS, so this always returns a config.
func RDSTLSConfig(ctx context.Context, dataSource *storepb.DataSource) (*tls.Config, error) {
	if !dataSource.GetVerifyTlsCertificate() {
		// No verifier runs in this mode, so a root pool would never be consulted.
		return &tls.Config{InsecureSkipVerify: true}, nil // NOSONAR(go:S4830,go:S5527) operator disabled verification
	}
	rootCertPool, err := RDSRootCAs(ctx, dataSource)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get RDS cert pool")
	}
	return &tls.Config{ // NOSONAR(go:S4830) VerifyPeerCertificate below verifies chain and hostname
		RootCAs:               rootCertPool,
		InsecureSkipVerify:    true, // superseded by VerifyPeerCertificate below
		VerifyPeerCertificate: createCertificateVerifier(rootCertPool, dataSource.GetHost()),
	}, nil
}
