package db

import (
	"crypto/tls"
	"crypto/x509"

	"github.com/pkg/errors"
)

// TLSConfig is the configuration for SSL connection.
type TLSConfig struct {
	SslCA   string
	SslCert string
	SslKey  string
}

// GetSslConfig gets the SSL config for connection.
func (tc TLSConfig) GetSslConfig() (*tls.Config, error) {
	if tc.SslCA == "" {
		return nil, nil
	}
	rootCertPool := x509.NewCertPool()
	if ok := rootCertPool.AppendCertsFromPEM([]byte(tc.SslCA)); !ok {
		return nil, errors.Errorf("rootCertPool.AppendCertsFromPEM() failed to append server CA pem")
	}

	cfg := &tls.Config{
		RootCAs: rootCertPool,
	}
	if (tc.SslCert == "" && tc.SslKey != "") || (tc.SslCert != "" && tc.SslKey == "") {
		return nil, errors.Errorf("ssl-cert and ssl-key must be both set or unset")
	}
	if tc.SslCert != "" && tc.SslKey != "" {
		var clientCert []tls.Certificate
		certs, err := tls.X509KeyPair([]byte(tc.SslCert), []byte(tc.SslKey))
		if err != nil {
			return nil, err
		}
		clientCert = append(clientCert, certs)

		cfg.Certificates = clientCert
	}

	cfg.InsecureSkipVerify = true
	cfg.VerifyPeerCertificate = func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
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
