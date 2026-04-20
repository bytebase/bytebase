package util

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

func TestResolveTLSMaterialReadsPathFields(t *testing.T) {
	dir := t.TempDir()
	caPath := filepath.Join(dir, "ca.pem")
	certPath := filepath.Join(dir, "cert.pem")
	keyPath := filepath.Join(dir, "key.pem")
	require.NoError(t, os.WriteFile(caPath, []byte("ca-pem"), 0o600))
	require.NoError(t, os.WriteFile(certPath, []byte("cert-pem"), 0o600))
	require.NoError(t, os.WriteFile(keyPath, []byte("key-pem"), 0o600))

	ds := &storepb.DataSource{
		UseSsl:      true,
		SslCa:       "stale-ca",
		SslCert:     "stale-cert",
		SslKey:      "stale-key",
		SslCaPath:   caPath,
		SslCertPath: certPath,
		SslKeyPath:  keyPath,
	}

	resolved, err := ResolveTLSMaterial(ds)
	require.NoError(t, err)
	require.Equal(t, "ca-pem", resolved.GetSslCa())
	require.Equal(t, "cert-pem", resolved.GetSslCert())
	require.Equal(t, "key-pem", resolved.GetSslKey())
	require.Equal(t, "stale-ca", ds.GetSslCa())
	require.Equal(t, caPath, ds.GetSslCaPath())
	require.Equal(t, certPath, ds.GetSslCertPath())
	require.Equal(t, keyPath, ds.GetSslKeyPath())
	require.Empty(t, resolved.GetSslCaPath())
	require.Empty(t, resolved.GetSslCertPath())
	require.Empty(t, resolved.GetSslKeyPath())
}

func TestResolveTLSMaterialRejectsRelativePath(t *testing.T) {
	_, err := ResolveTLSMaterial(&storepb.DataSource{
		UseSsl:    true,
		SslCaPath: "relative-ca.pem",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "CA certificate path must be absolute")
	require.NotContains(t, err.Error(), "relative-ca.pem")
}

func TestResolveTLSMaterialRedactsReadErrors(t *testing.T) {
	missingPath := filepath.Join(t.TempDir(), "missing.pem")
	_, err := ResolveTLSMaterial(&storepb.DataSource{
		UseSsl:    true,
		SslCaPath: missingPath,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to read CA certificate file")
	require.False(t, strings.Contains(err.Error(), missingPath), err.Error())
	require.NotContains(t, err.Error(), "no such file")
}
