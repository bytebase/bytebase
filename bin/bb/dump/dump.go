// dump is a library for dumping database schemas provided by bytebase.com.
package dump

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/bytebase/bytebase/bin/bb/dump/mysqldump"
	"github.com/bytebase/bytebase/bin/bb/dump/pgdump"
)

// Dump exports the schema of a database instance.
// All non-system databases will be exported to the input directory in the format of database_name.sql for each database.
// When directory isn't specified, the schema will be exported to stdout.
func Dump(databaseType, username, password, hostname, port, database, directory string, tlsCfg TlsConfig) error {
	if directory != "" {
		dirInfo, err := os.Stat(directory)
		if os.IsNotExist(err) {
			return fmt.Errorf("directory %q does not exist", directory)
		}
		if !dirInfo.IsDir() {
			return fmt.Errorf("path %q isn't a directory", directory)
		}
	}

	switch databaseType {
	case "mysql":
		if username == "" && port == "" {
			username, port = "root", "3306"
		}
		tlsConfig, err := tlsCfg.GetSslConfig()
		if err != nil {
			return fmt.Errorf("TlsConfig.GetSslConfig() got error: %v", err)
		}
		dp, err := mysqldump.New(username, password, hostname, port, tlsConfig)
		if err != nil {
			return fmt.Errorf("mysqldump.New(%q, %q, %q, %q) got error: %v", username, password, hostname, port, err)
		}
		defer dp.Close()

		return dp.Dump(database, directory)
	case "pg":
		dp, err := pgdump.New(username, password, hostname, port, database, tlsCfg.SslCA, tlsCfg.SslCert, tlsCfg.SslKey)
		if err != nil {
			return fmt.Errorf("pgdump.New(%q, %q, %q, %q) got error: %v", username, password, hostname, port, err)
		}
		defer dp.Close()
		return dp.Dump(database, directory)
	default:
		return fmt.Errorf("database type %q not supported; supported types: mysql, pg.", databaseType)
	}
}

// TlsConfig is the configuration for SSL connection.
type TlsConfig struct {
	SslCA   string
	SslCert string
	SslKey  string
}

// GetSslConfig gets the SSL config for connection.
func (tc TlsConfig) GetSslConfig() (*tls.Config, error) {
	if tc.SslCA == "" {
		return nil, nil
	}
	rootCertPool := x509.NewCertPool()
	pem, err := ioutil.ReadFile(tc.SslCA)
	if err != nil {
		return nil, err
	}
	if ok := rootCertPool.AppendCertsFromPEM(pem); !ok {
		return nil, fmt.Errorf("rootCertPool.AppendCertsFromPEM() failed to append server CA pem.")
	}

	cfg := &tls.Config{
		RootCAs: rootCertPool,
	}
	if (tc.SslCert == "" && tc.SslKey != "") || (tc.SslCert != "" && tc.SslKey == "") {
		return nil, fmt.Errorf("ssl-cert and ssl-key must be both set or unset.")
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
			return fmt.Errorf("empty certificate to verify.")
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
