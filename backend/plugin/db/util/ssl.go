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
	var rootCertPool *x509.CertPool
	if ds.GetSslCa() == "" {
		p, err := x509.SystemCertPool()
		if err != nil {
			return nil, err
		}
		rootCertPool = p
	} else {
		rootCertPool = x509.NewCertPool()
		if ok := rootCertPool.AppendCertsFromPEM([]byte(ds.GetSslCa())); !ok {
			return nil, errors.Errorf("rootCertPool.AppendCertsFromPEM() failed to append server CA pem")
		}
	}

	cfg := &tls.Config{
		RootCAs: rootCertPool,
	}
	if (ds.GetSslCert() == "" && ds.GetSslKey() != "") || (ds.GetSslCert() != "" && ds.GetSslKey() == "") {
		return nil, errors.Errorf("ssl-cert and ssl-key must be both set or unset")
	}
	if ds.GetSslCert() != "" && ds.GetSslKey() != "" {
		var clientCert []tls.Certificate
		certs, err := tls.X509KeyPair([]byte(ds.GetSslCert()), []byte(ds.GetSslKey()))
		if err != nil {
			return nil, err
		}
		clientCert = append(clientCert, certs)

		cfg.Certificates = clientCert
	}

	cfg.InsecureSkipVerify = true
	cfg.VerifyPeerCertificate = func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
		if len(rawCerts) == 0 {
			return errors.Errorf("empty certificate to verify")
		}
		cert, err := x509.ParseCertificate(rawCerts[0])
		if err != nil {
			return err
		}

		// Add the intermediates.
		intermediatePool := x509.NewCertPool()
		for _, intermediate := range rawCerts[1:] {
			cert, err := x509.ParseCertificate(intermediate)
			if err != nil {
				return err
			}
			intermediatePool.AddCert(cert)
		}

		opts := x509.VerifyOptions{
			Roots:         rootCertPool,
			Intermediates: intermediatePool,
		}
		if _, err = cert.Verify(opts); err != nil {
			return errors.Wrap(err, "SSL cert failed to verify")
		}
		return nil
	}
	return cfg, nil
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
