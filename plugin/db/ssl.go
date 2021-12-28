package db

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
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
	pem, err := os.ReadFile(tc.SslCA)
	if err != nil {
		return nil, err
	}
	if ok := rootCertPool.AppendCertsFromPEM(pem); !ok {
		return nil, fmt.Errorf("rootCertPool.AppendCertsFromPEM() failed to append server CA pem")
	}

	cfg := &tls.Config{
		RootCAs: rootCertPool,
	}
	if (tc.SslCert == "" && tc.SslKey != "") || (tc.SslCert != "" && tc.SslKey == "") {
		return nil, fmt.Errorf("ssl-cert and ssl-key must be both set or unset")
	}
	if tc.SslCert != "" && tc.SslKey != "" {
		clientCert := make([]tls.Certificate, 0, 1)
		certs, err := tls.LoadX509KeyPair(tc.SslCert, tc.SslKey)
		if err != nil {
			return nil, err
		}
		clientCert = append(clientCert, certs)

		cfg.Certificates = clientCert
	}

	cfg.InsecureSkipVerify = true
	cfg.VerifyPeerCertificate = func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
		if len(rawCerts) == 0 {
			return fmt.Errorf("empty certificate to verify")
		}
		cert, err := x509.ParseCertificate(rawCerts[0])
		if err != nil {
			return err
		}
		opts := x509.VerifyOptions{Roots: rootCertPool}
		if _, err = cert.Verify(opts); err != nil {
			return fmt.Errorf("SSL cert failed to verify: %v", err)
		}
		return nil
	}
	return cfg, nil
}
