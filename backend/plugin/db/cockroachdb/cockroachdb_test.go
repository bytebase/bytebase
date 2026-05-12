package cockroachdb

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"testing"
	"time"

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

func TestGetRoutingIDFromCockroachCloudURL(t *testing.T) {
	tests := []struct {
		host     string
		expected string
	}{
		{
			host:     "routing-id.cockroachlabs.cloud",
			expected: "routing-id",
		},
		{
			host:     "bytebase-cluster-7749.6xw.aws-ap-southeast-1.cockroachlabs.cloud",
			expected: "bytebase-cluster-7749",
		},
		{
			host:     "subdomain.routing-id.cockroachlabs.cloud",
			expected: "subdomain",
		},
		{
			host:     "subdomain.routing-id.cockroachlabs.cloud",
			expected: "subdomain",
		},
		{
			host:     "cockroachlabs.cloud",
			expected: "",
		},
		{
			host:     "example.com",
			expected: "",
		},
	}

	for _, test := range tests {
		got := getRoutingIDFromCockroachCloudURL(test.host)
		require.Equal(t, test.expected, got, "host: %s", test.host)
	}
}

func TestGetCockroachConnectionConfigAddsClientCertificateForAllHosts(t *testing.T) {
	certPEM, keyPEM := generateClientCertificatePEM(t)
	connConfig, err := getCockroachConnectionConfig(db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Username:             "dba",
			Host:                 "172.18.22.61,172.18.22.62,172.18.22.63",
			Port:                 "26257",
			UseSsl:               true,
			VerifyTlsCertificate: false,
			SslCert:              certPEM,
			SslKey:               keyPEM,
		},
		ConnectionContext: db.ConnectionContext{
			DatabaseName: "defaultdb",
		},
	})
	require.NoError(t, err)
	require.NotNil(t, connConfig.TLSConfig)
	require.Len(t, connConfig.TLSConfig.Certificates, 1)
	require.Len(t, connConfig.Fallbacks, 2)
	for _, fallback := range connConfig.Fallbacks {
		require.NotNil(t, fallback.TLSConfig)
		require.True(t, fallback.TLSConfig.InsecureSkipVerify)
		require.Len(t, fallback.TLSConfig.Certificates, 1)
	}
}

func TestGetCockroachConnectionConfigVerifiesCustomCAForAllHosts(t *testing.T) {
	hosts := []string{"crdb-1.example.com", "crdb-2.example.com", "crdb-3.example.com"}
	caPEM, serverCertDERByHost := generateCAAndServerCertificates(t, hosts)
	connConfig, err := getCockroachConnectionConfig(db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Username:             "dba",
			Host:                 "crdb-1.example.com,crdb-2.example.com,crdb-3.example.com",
			Port:                 "26257",
			UseSsl:               true,
			VerifyTlsCertificate: true,
			SslCa:                caPEM,
		},
		ConnectionContext: db.ConnectionContext{
			DatabaseName: "defaultdb",
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
