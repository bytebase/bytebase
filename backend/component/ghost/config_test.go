package ghost

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

func TestNewMigrationContextWritesTLSMaterialToTempFiles(t *testing.T) {
	certPEM, keyPEM := generateSelfSignedPEM(t)

	ctx := context.Background()
	database := &store.DatabaseMessage{DatabaseName: "ghostdb"}
	dataSource := &storepb.DataSource{
		Host:                 "127.0.0.1",
		Port:                 "3306",
		Username:             "root",
		UseSsl:               true,
		VerifyTlsCertificate: false,
		SslCa:                certPEM,
		SslCert:              certPEM,
		SslKey:               keyPEM,
		AuthenticationType:   storepb.DataSource_PASSWORD,
	}

	migrationContext, err := NewMigrationContext(ctx, 1, database, dataSource, "t", "_suffix", "ALTER TABLE t ADD COLUMN c INT", false, nil, 0)
	require.NoError(t, err)
	require.True(t, migrationContext.UseTLS)
	require.True(t, migrationContext.TLSAllowInsecure)
	require.True(t, filepath.IsAbs(migrationContext.TLSCACertificate))
	require.True(t, filepath.IsAbs(migrationContext.TLSCertificate))
	require.True(t, filepath.IsAbs(migrationContext.TLSKey))
	require.NotEqual(t, certPEM, migrationContext.TLSCACertificate)
	require.NotEqual(t, certPEM, migrationContext.TLSCertificate)
	require.NotEqual(t, keyPEM, migrationContext.TLSKey)
	require.NoFileExists(t, migrationContext.TLSCACertificate)
	require.NoFileExists(t, migrationContext.TLSCertificate)
	require.NoFileExists(t, migrationContext.TLSKey)
}

func TestNewMigrationContextRespectsVerifyTlsCertificate(t *testing.T) {
	certPEM, keyPEM := generateSelfSignedPEM(t)

	ctx := context.Background()
	database := &store.DatabaseMessage{DatabaseName: "ghostdb"}
	dataSource := &storepb.DataSource{
		Host:                 "127.0.0.1",
		Port:                 "3306",
		Username:             "root",
		UseSsl:               true,
		VerifyTlsCertificate: true,
		SslCa:                certPEM,
		SslCert:              certPEM,
		SslKey:               keyPEM,
	}

	migrationContext, err := NewMigrationContext(ctx, 1, database, dataSource, "t", "_suffix", "ALTER TABLE t ADD COLUMN c INT", false, nil, 0)
	require.NoError(t, err)
	require.False(t, migrationContext.TLSAllowInsecure)
}

func generateSelfSignedPEM(t *testing.T) (string, string) {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName:   "bytebase-test",
			Organization: []string{"Bytebase"},
		},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	require.NotEmpty(t, certPEM)
	require.NotEmpty(t, keyPEM)

	return string(certPEM), string(keyPEM)
}
