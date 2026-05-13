package ghost

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"database/sql"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net"
	"path/filepath"
	"strings"
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

	migrationContext, cleanup, err := NewMigrationContext(ctx, 1, database, dataSource, "t", "_suffix", "ALTER TABLE t ADD COLUMN c INT", false, nil, 0)
	require.NoError(t, err)
	t.Cleanup(cleanup)
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

	migrationContext, cleanup, err := NewMigrationContext(ctx, 1, database, dataSource, "t", "_suffix", "ALTER TABLE t ADD COLUMN c INT", false, nil, 0)
	require.NoError(t, err)
	t.Cleanup(cleanup)
	require.False(t, migrationContext.TLSAllowInsecure)
}

func TestNewMigrationContextUsesSSHNetworkDialersAndCleanup(t *testing.T) {
	originalGetSSHDialer := getSSHDialer
	fakeDialer := &fakeSSHDialer{}
	getSSHDialer = func(_ *storepb.DataSource) (sshDialer, error) {
		return fakeDialer, nil
	}
	t.Cleanup(func() {
		getSSHDialer = originalGetSSHDialer
	})

	ctx := context.Background()
	database := &store.DatabaseMessage{DatabaseName: "ghostdb"}
	dataSource := &storepb.DataSource{
		Host:               "172.29.0.10",
		Port:               "3306",
		Username:           "root",
		Password:           "root",
		SshHost:            "127.0.0.1",
		SshPort:            "2222",
		SshUser:            "bb",
		AuthenticationType: storepb.DataSource_PASSWORD,
	}

	migrationContext, cleanup, err := NewMigrationContext(ctx, 1, database, dataSource, "t", "_suffix", "ALTER TABLE t ADD COLUMN c INT", false, nil, 0)
	require.NoError(t, err)
	cleanedUp := false
	t.Cleanup(func() {
		if !cleanedUp {
			cleanup()
		}
	})
	require.True(t, strings.HasPrefix(migrationContext.InspectorConnectionConfig.Network, "mysql-tcp-"))
	require.NotNil(t, migrationContext.InspectorConnectionConfig.Dialer)

	sqlCtx := context.WithValue(ctx, dialContextKey{}, "sql")
	db, err := sql.Open("mysql", fmt.Sprintf("%s@%s(%s)/%s", dataSource.GetUsername(), migrationContext.InspectorConnectionConfig.Network, dataSource.GetHost()+":"+dataSource.GetPort(), database.DatabaseName))
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = db.Close()
	})
	require.ErrorIs(t, db.PingContext(sqlCtx), errFakeDial)
	require.Len(t, fakeDialer.calls, 1)
	require.Equal(t, "sql", fakeDialer.calls[0].contextValue)
	require.Equal(t, "tcp", fakeDialer.calls[0].network)
	require.Equal(t, dataSource.GetHost()+":"+dataSource.GetPort(), fakeDialer.calls[0].address)

	binlogCtx := context.WithValue(ctx, dialContextKey{}, "binlog")
	_, err = migrationContext.InspectorConnectionConfig.Dialer(binlogCtx, "tcp", dataSource.GetHost()+":"+dataSource.GetPort())
	require.ErrorIs(t, err, errFakeDial)
	require.Len(t, fakeDialer.calls, 2)
	require.Equal(t, "binlog", fakeDialer.calls[1].contextValue)

	cleanup()
	cleanedUp = true
	require.Equal(t, 1, fakeDialer.closeCount)

	dbAfterCleanup, err := sql.Open("mysql", fmt.Sprintf("%s@%s(%s)/%s", dataSource.GetUsername(), migrationContext.InspectorConnectionConfig.Network, dataSource.GetHost()+":"+dataSource.GetPort(), database.DatabaseName))
	require.NoError(t, err)
	defer dbAfterCleanup.Close()
	require.Error(t, dbAfterCleanup.PingContext(ctx))
	require.Len(t, fakeDialer.calls, 2)
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

type dialContextKey struct{}

var errFakeDial = errors.New("fake dial")

type fakeSSHDialer struct {
	calls      []fakeSSHDialCall
	closeCount int
}

type fakeSSHDialCall struct {
	contextValue any
	network      string
	address      string
}

func (d *fakeSSHDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	d.calls = append(d.calls, fakeSSHDialCall{
		contextValue: ctx.Value(dialContextKey{}),
		network:      network,
		address:      address,
	})
	return nil, errFakeDial
}

func (d *fakeSSHDialer) Close() error {
	d.closeCount++
	return nil
}
