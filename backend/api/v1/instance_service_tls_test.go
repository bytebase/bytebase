package v1

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

func TestValidateDataSourceTLSWriteRejectsMixedExplicitMaterial(t *testing.T) {
	err := validateDataSourceTLSWrite(
		&storepb.DataSource{UseSsl: true, SslCa: "inline-ca", SslCaPath: "/tmp/ca.pem"},
		&storepb.DataSource{UseSsl: true, SslCa: "inline-ca", SslCaPath: "/tmp/ca.pem"},
		[]string{"ssl_ca", "ssl_ca_path"},
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "cannot set both inline TLS material and TLS file paths")
}

func TestNormalizeDataSourceTLSClearsInlineWhenPathWins(t *testing.T) {
	ds := &storepb.DataSource{
		UseSsl:    true,
		SslCa:     "inline-ca",
		SslCert:   "inline-cert",
		SslKey:    "inline-key",
		SslCaPath: "/tmp/ca.pem",
	}
	normalizeDataSourceTLS(ds, []string{"ssl_ca_path"})
	require.Empty(t, ds.GetSslCa())
	require.Empty(t, ds.GetSslCert())
	require.Empty(t, ds.GetSslKey())
	require.Equal(t, "/tmp/ca.pem", ds.GetSslCaPath())
}

func TestNormalizeDataSourceTLSClearsStaleInlineWhenPathExists(t *testing.T) {
	ds := &storepb.DataSource{
		UseSsl:    true,
		SslCa:     "inline-ca",
		SslCert:   "inline-cert",
		SslKey:    "inline-key",
		SslCaPath: "/tmp/ca.pem",
	}
	normalizeDataSourceTLS(ds, []string{"verify_tls_certificate"})
	require.Empty(t, ds.GetSslCa())
	require.Empty(t, ds.GetSslCert())
	require.Empty(t, ds.GetSslKey())
	require.Equal(t, "/tmp/ca.pem", ds.GetSslCaPath())
}

func TestNormalizeDataSourceTLSClearsPathsWhenSwitchingToInline(t *testing.T) {
	ds := &storepb.DataSource{
		UseSsl:      true,
		SslCa:       "inline-ca",
		SslCaPath:   "/tmp/ca.pem",
		SslCertPath: "/tmp/cert.pem",
		SslKeyPath:  "/tmp/key.pem",
	}
	normalizeDataSourceTLS(ds, []string{"ssl_ca", "ssl_ca_path", "ssl_cert_path", "ssl_key_path"})
	require.Equal(t, "inline-ca", ds.GetSslCa())
	require.Empty(t, ds.GetSslCaPath())
	require.Empty(t, ds.GetSslCertPath())
	require.Empty(t, ds.GetSslKeyPath())
}

func TestNormalizeDataSourceTLSClearsAllWhenDisabled(t *testing.T) {
	ds := &storepb.DataSource{
		UseSsl:      false,
		SslCa:       "inline-ca",
		SslCert:     "inline-cert",
		SslKey:      "inline-key",
		SslCaPath:   "/tmp/ca.pem",
		SslCertPath: "/tmp/cert.pem",
		SslKeyPath:  "/tmp/key.pem",
	}
	normalizeDataSourceTLS(ds, []string{"use_ssl"})
	require.Empty(t, ds.GetSslCa())
	require.Empty(t, ds.GetSslCert())
	require.Empty(t, ds.GetSslKey())
	require.Empty(t, ds.GetSslCaPath())
	require.Empty(t, ds.GetSslCertPath())
	require.Empty(t, ds.GetSslKeyPath())
}

func TestTLSMaskPaths(t *testing.T) {
	mask := &fieldmaskpb.FieldMask{Paths: []string{"ssl_ca_path", "host"}}
	require.True(t, tlsMaskContains(mask.GetPaths(), "ssl_ca_path"))
	require.False(t, tlsMaskContains(mask.GetPaths(), "ssl_cert_path"))
}

func TestValidateDataSourceTLSWriteRejectsInactiveInlineWithExistingPath(t *testing.T) {
	err := validateDataSourceTLSWrite(
		&storepb.DataSource{UseSsl: true, SslCa: "inline-ca"},
		&storepb.DataSource{UseSsl: true, SslCa: "inline-ca", SslCaPath: "/tmp/ca.pem"},
		[]string{"ssl_ca"},
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "cannot set inline TLS material while TLS file paths are configured")
}

func TestValidateDataSourceTLSConfigRejectsRelativePath(t *testing.T) {
	err := validateDataSourceTLSConfig(&storepb.DataSource{UseSsl: true, SslCaPath: "ca.pem"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "ssl_ca_path must be an absolute path")
}
