package pg

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
)

func TestGetDatabaseInCreateDatabaseStatement(t *testing.T) {
	tests := []struct {
		createDatabaseStatement string
		want                    string
		wantErr                 bool
	}{
		{
			`CREATE DATABASE "hello" ENCODING "UTF8";`,
			"hello",
			false,
		},
		{
			`CREATE DATABASE "hello";`,
			"hello",
			false,
		},
		{
			`CREATE DATABASE hello;`,
			"hello",
			false,
		},
		{
			`CREATE DATABASE hello ENCODING "UTF8";`,
			"hello",
			false,
		},
		{
			`CREATE DATABASE;`,
			"",
			true,
		},
	}

	for _, test := range tests {
		got, err := getDatabaseInCreateDatabaseStatement(test.createDatabaseStatement)
		if test.wantErr {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
		}
		require.Equal(t, test.want, got)
	}
}

func TestGetPGConnectionConfigUsesPgBouncerCompatibleQueryMode(t *testing.T) {
	connConfig, err := getPGConnectionConfig(db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Username: "dba",
			Host:     "pgbouncer.example.com",
			Port:     "6432",
		},
		ConnectionContext: db.ConnectionContext{
			DatabaseName: "postgres",
		},
	})
	require.NoError(t, err)
	require.Equal(t, pgx.QueryExecModeExec, connConfig.DefaultQueryExecMode)
	require.Zero(t, connConfig.StatementCacheCapacity)
}

func TestGetPGConnectionConfigPreservesExplicitQueryExecMode(t *testing.T) {
	connConfig, err := getPGConnectionConfig(db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Username: "dba",
			Host:     "proxy.example.com",
			Port:     "6432",
			ExtraConnectionParameters: map[string]string{
				"default_query_exec_mode":  "simple_protocol",
				"statement_cache_capacity": "128",
			},
		},
		ConnectionContext: db.ConnectionContext{
			DatabaseName: "postgres",
		},
	})
	require.NoError(t, err)
	require.Equal(t, pgx.QueryExecModeSimpleProtocol, connConfig.DefaultQueryExecMode)
	require.Equal(t, 128, connConfig.StatementCacheCapacity)
}

func TestGetPGConnectionConfigDisablesTLSVerificationForAllHosts(t *testing.T) {
	connConfig, err := getPGConnectionConfig(db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Username:             "dba",
			Host:                 "172.18.22.61,172.18.22.62,172.18.22.63",
			Port:                 "5432",
			UseSsl:               true,
			VerifyTlsCertificate: false,
		},
		ConnectionContext: db.ConnectionContext{
			DatabaseName: "lapidlive",
		},
	})
	require.NoError(t, err)
	require.NotNil(t, connConfig.TLSConfig)
	require.True(t, connConfig.TLSConfig.InsecureSkipVerify)
	require.Len(t, connConfig.Fallbacks, 2)
	for _, fallback := range connConfig.Fallbacks {
		require.NotNil(t, fallback.TLSConfig)
		require.True(t, fallback.TLSConfig.InsecureSkipVerify)
	}
}

func TestGetPGConnectionConfigAddsClientCertificateForAllHosts(t *testing.T) {
	certPEM, keyPEM := generateClientCertificatePEM(t)
	connConfig, err := getPGConnectionConfig(db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Username:             "dba",
			Host:                 "172.18.22.61,172.18.22.62,172.18.22.63",
			Port:                 "5432",
			UseSsl:               true,
			VerifyTlsCertificate: false,
			SslCert:              certPEM,
			SslKey:               keyPEM,
		},
		ConnectionContext: db.ConnectionContext{
			DatabaseName: "lapidlive",
		},
	})
	require.NoError(t, err)
	require.NotNil(t, connConfig.TLSConfig)
	require.Len(t, connConfig.TLSConfig.Certificates, 1)
	require.Len(t, connConfig.Fallbacks, 2)
	for _, fallback := range connConfig.Fallbacks {
		require.NotNil(t, fallback.TLSConfig)
		require.Len(t, fallback.TLSConfig.Certificates, 1)
	}
}

func TestGetPGConnectionConfigVerifiesCustomCAForAllHosts(t *testing.T) {
	hosts := []string{"pg-1.example.com", "pg-2.example.com", "pg-3.example.com"}
	caPEM, serverCertDERByHost := generateCAAndServerCertificates(t, hosts)
	connConfig, err := getPGConnectionConfig(db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Username:             "dba",
			Host:                 "pg-1.example.com,pg-2.example.com,pg-3.example.com",
			Port:                 "5432",
			UseSsl:               true,
			VerifyTlsCertificate: true,
			SslCa:                caPEM,
		},
		ConnectionContext: db.ConnectionContext{
			DatabaseName: "lapidlive",
		},
	})
	require.NoError(t, err)
	require.NotNil(t, connConfig.TLSConfig)
	require.NotNil(t, connConfig.TLSConfig.RootCAs)
	require.NotNil(t, connConfig.TLSConfig.VerifyPeerCertificate)
	require.NoError(t, connConfig.TLSConfig.VerifyPeerCertificate([][]byte{serverCertDERByHost[hosts[0]]}, nil))
	require.Len(t, connConfig.Fallbacks, 2)
	for i, fallback := range connConfig.Fallbacks {
		require.NotNil(t, fallback.TLSConfig)
		require.NotNil(t, fallback.TLSConfig.RootCAs)
		require.NotNil(t, fallback.TLSConfig.VerifyPeerCertificate)
		require.NoError(t, fallback.TLSConfig.VerifyPeerCertificate([][]byte{serverCertDERByHost[hosts[i+1]]}, nil))
	}
}

func generateClientCertificatePEM(t *testing.T) (string, string) {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: "bytebase-test-client",
		},
		NotBefore: time.Now().Add(-time.Hour),
		NotAfter:  time.Now().Add(time.Hour),
		KeyUsage:  x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageClientAuth,
		},
	}
	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	return string(certPEM), string(keyPEM)
}

func generateCAAndServerCertificates(t *testing.T, hosts []string) (string, map[string][]byte) {
	t.Helper()

	caKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	caTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "bytebase-test-ca"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	require.NoError(t, err)

	serverCertDERByHost := make(map[string][]byte, len(hosts))
	for i, host := range hosts {
		serverKey, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)
		serverTemplate := &x509.Certificate{
			SerialNumber: big.NewInt(int64(i + 2)),
			Subject:      pkix.Name{CommonName: host},
			DNSNames:     []string{host},
			NotBefore:    time.Now().Add(-time.Hour),
			NotAfter:     time.Now().Add(time.Hour),
			KeyUsage:     x509.KeyUsageDigitalSignature,
			ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		}
		serverDER, err := x509.CreateCertificate(rand.Reader, serverTemplate, caTemplate, &serverKey.PublicKey, caKey)
		require.NoError(t, err)
		serverCertDERByHost[host] = serverDER
	}

	caPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER})
	return string(caPEM), serverCertDERByHost
}
