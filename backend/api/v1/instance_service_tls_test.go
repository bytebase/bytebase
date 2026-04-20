package v1

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

const validCAPEM = `-----BEGIN CERTIFICATE-----
MIIDOTCCAiGgAwIBAgIQSRJrEpBGFc7tNb1fb5pKFzANBgkqhkiG9w0BAQsFADAS
MRAwDgYDVQQKEwdBY21lIENvMCAXDTcwMDEwMTAwMDAwMFoYDzIwODQwMTI5MTYw
MDAwWjASMRAwDgYDVQQKEwdBY21lIENvMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8A
MIIBCgKCAQEA6Gba5tHV1dAKouAaXO3/ebDUU4rvwCUg/CNaJ2PT5xLD4N1Vcb8r
bFSW2HXKq+MPfVdwIKR/1DczEoAGf/JWQTW7EgzlXrCd3rlajEX2D73faWJekD0U
aUgz5vtrTXZ90BQL7WvRICd7FlEZ6FPOcPlumiyNmzUqtwGhO+9ad1W5BqJaRI6P
YfouNkwR6Na4TzSj5BrqUfP0FwDizKSJ0XXmh8g8G9mtwxOSN3Ru1QFc61Xyeluk
POGKBV/q6RBNklTNe0gI8usUMlYyoC7ytppNMW7X2vodAelSu25jgx2anj9fDVZu
h7AXF5+4nJS4AAt0n1lNY7nGSsdZas8PbQIDAQABo4GIMIGFMA4GA1UdDwEB/wQE
AwICpDATBgNVHSUEDDAKBggrBgEFBQcDATAPBgNVHRMBAf8EBTADAQH/MB0GA1Ud
DgQWBBStsdjh3/JCXXYlQryOrL4Sh7BW5TAuBgNVHREEJzAlggtleGFtcGxlLmNv
bYcEfwAAAYcQAAAAAAAAAAAAAAAAAAAAATANBgkqhkiG9w0BAQsFAAOCAQEAxWGI
5NhpF3nwwy/4yB4i/CwwSpLrWUa70NyhvprUBC50PxiXav1TeDzwzLx/o5HyNwsv
cxv3HdkLW59i/0SlJSrNnWdfZ19oTcS+6PtLoVyISgtyN6DpkKpdG1cOkW3Cy2P2
+tK/tKHRP1Y/Ra0RiDpOAmqn0gCOFGz8+lqDIor/T7MTpibL3IxqWfPrvfVRHL3B
grw/ZQTTIVjjh4JBSW3WyWgNo/ikC1lrVxzl4iPUGptxT36Cr7Zk2Bsg0XqwbOvK
5d+NTDREkSnUbie4GeutujmX3Dsx88UiV6UY/4lHJa6I5leHUNOHahRbpbWeOfs/
WkBKOclmOV2xlTVuPw==
-----END CERTIFICATE-----`

func TestValidateDataSourceTLSWriteRejectsSameSlotMixedMaterial(t *testing.T) {
	tests := []struct {
		name string
		ds   *storepb.DataSource
		mask []string
		want string
	}{
		{
			name: "ca",
			ds:   &storepb.DataSource{UseSsl: true, SslCa: "inline-ca", SslCaPath: "/tmp/ca.pem"},
			mask: []string{"ssl_ca", "ssl_ca_path"},
			want: "cannot set both ssl_ca and ssl_ca_path",
		},
		{
			name: "cert",
			ds:   &storepb.DataSource{UseSsl: true, SslCert: "inline-cert", SslCertPath: "/tmp/cert.pem"},
			mask: []string{"ssl_cert", "ssl_cert_path"},
			want: "cannot set both ssl_cert and ssl_cert_path",
		},
		{
			name: "key",
			ds:   &storepb.DataSource{UseSsl: true, SslKey: "inline-key", SslKeyPath: "/tmp/key.pem"},
			mask: []string{"ssl_key", "ssl_key_path"},
			want: "cannot set both ssl_key and ssl_key_path",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateDataSourceTLSWrite(tc.ds, tc.ds, tc.mask)
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.want)
		})
	}
}

func TestValidateDataSourceTLSWriteRejectsSameSlotMixedMaterialWithPartialMask(t *testing.T) {
	requested := &storepb.DataSource{UseSsl: true, SslCa: "inline-ca", SslCaPath: "/tmp/ca.pem"}
	merged := &storepb.DataSource{UseSsl: true, SslCa: "inline-ca"}

	err := validateDataSourceTLSWrite(requested, merged, []string{"ssl_ca"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "cannot set both ssl_ca and ssl_ca_path")
}

func TestValidateDataSourceTLSWriteAllowsCrossSlotMixedMaterial(t *testing.T) {
	err := validateDataSourceTLSWrite(
		&storepb.DataSource{UseSsl: true, SslCa: validCAPEM, SslCertPath: "/tmp/cert.pem", SslKeyPath: "/tmp/key.pem"},
		&storepb.DataSource{UseSsl: true, SslCa: validCAPEM, SslCertPath: "/tmp/cert.pem", SslKeyPath: "/tmp/key.pem"},
		[]string{"ssl_ca", "ssl_cert_path", "ssl_key_path"},
	)
	require.NoError(t, err)
}

func TestValidateDataSourceTLSWriteAllowsSourceSwitchFromInlineToPath(t *testing.T) {
	requested := &storepb.DataSource{UseSsl: true, SslCaPath: "/tmp/ca.pem"}
	merged := &storepb.DataSource{UseSsl: true, SslCa: validCAPEM, SslCaPath: "/tmp/ca.pem"}

	err := validateDataSourceTLSWrite(requested, merged, []string{"ssl_ca_path"})
	require.NoError(t, err)

	normalizeDataSourceTLS(merged, []string{"ssl_ca_path"})
	require.Empty(t, merged.GetSslCa())
	require.Equal(t, "/tmp/ca.pem", merged.GetSslCaPath())
}

func TestValidateDataSourceTLSWriteAllowsSourceSwitchFromPathToInline(t *testing.T) {
	requested := &storepb.DataSource{UseSsl: true, SslCa: validCAPEM}
	merged := &storepb.DataSource{UseSsl: true, SslCa: validCAPEM, SslCaPath: "/tmp/ca.pem"}

	err := validateDataSourceTLSWrite(requested, merged, []string{"ssl_ca"})
	require.NoError(t, err)

	normalizeDataSourceTLS(merged, []string{"ssl_ca"})
	require.Equal(t, validCAPEM, merged.GetSslCa())
	require.Empty(t, merged.GetSslCaPath())
}

func TestValidateDataSourceTLSWriteAllowsCertPathAndInlineKey(t *testing.T) {
	err := validateDataSourceTLSWrite(
		&storepb.DataSource{UseSsl: true, SslCertPath: "/tmp/cert.pem", SslKey: "inline-key"},
		&storepb.DataSource{UseSsl: true, SslCertPath: "/tmp/cert.pem", SslKey: "inline-key"},
		[]string{"ssl_cert_path", "ssl_key"},
	)
	require.NoError(t, err)
}

func TestValidateDataSourceTLSWriteAllowsCertInlineAndKeyPath(t *testing.T) {
	err := validateDataSourceTLSWrite(
		&storepb.DataSource{UseSsl: true, SslCert: validCAPEM, SslKeyPath: "/tmp/key.pem"},
		&storepb.DataSource{UseSsl: true, SslCert: validCAPEM, SslKeyPath: "/tmp/key.pem"},
		[]string{"ssl_cert", "ssl_key_path"},
	)
	require.NoError(t, err)
}

func TestNormalizeDataSourceTLSClearsSameSlotConflictsOnly(t *testing.T) {
	ds := &storepb.DataSource{
		UseSsl:     true,
		SslCa:      "inline-ca",
		SslCaPath:  "/tmp/ca.pem",
		SslCert:    "inline-cert",
		SslKeyPath: "/tmp/key.pem",
	}
	normalizeDataSourceTLS(ds, []string{"ssl_ca_path", "ssl_key_path"})
	require.Empty(t, ds.GetSslCa())
	require.Equal(t, "/tmp/ca.pem", ds.GetSslCaPath())
	require.Equal(t, "inline-cert", ds.GetSslCert())
	require.Equal(t, "/tmp/key.pem", ds.GetSslKeyPath())
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

func TestValidateDataSourceTLSConfigRejectsRelativePath(t *testing.T) {
	err := validateDataSourceTLSConfig(&storepb.DataSource{UseSsl: true, SslCaPath: "ca.pem"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "ssl_ca_path must be an absolute path")
}
