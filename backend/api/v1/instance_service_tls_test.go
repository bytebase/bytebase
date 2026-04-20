package v1

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"testing"

	"github.com/stretchr/testify/require"

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

const validKeyPEM = `-----BEGIN RSA PRIVATE KEY-----
MIIEvAIBADANBgkqhkiG9w0BAQEFAASCBKYwggSiAgEAAoIBAQDoZtrm0dXV0Aqi
4Bpc7f95sNRTiu/AJSD8I1onY9PnEsPg3VVxvytsVJbYdcqr4w99V3AgpH/UNzMS
gAZ/8lZBNbsSDOVesJ3euVqMRfYPvd9pYl6QPRRpSDPm+2tNdn3QFAvta9EgJ3sW
URnoU85w+W6aLI2bNSq3AaE771p3VbkGolpEjo9h+i42TBHo1rhPNKPkGupR8/QX
AOLMpInRdeaHyDwb2a3DE5I3dG7VAVzrVfJ6W6Q84YoFX+rpEE2SVM17SAjy6xQy
VjKgLvK2mk0xbtfa+h0B6VK7bmODHZqeP18NVm6HsBcXn7iclLgAC3SfWU1jucZK
x1lqzw9tAgMBAAECggEABWzxS1Y2wckblnXY57Z+sl6YdmLV+gxj2r8Qib7g4ZIk
lIlWR1OJNfw7kU4eryib4fc6nOh6O4AWZyYqAK6tqNQSS/eVG0LQTLTTEldHyVJL
dvBe+MsUQOj4nTndZW+QvFzbcm2D8lY5n2nBSxU5ypVoKZ1EqQzytFcLZpTN7d89
EPj0qDyrV4NZlWAwL1AygCwnlwhMQjXEalVF1ylXwU3QzyZ/6MgvF6d3SSUlh+sq
XefuyigXw484cQQgbzopv6niMOmGP3of+yV4JQqUSb3IDmmT68XjGd2Dkxl4iPki
6ZwXf3CCi+c+i/zVEcufgZ3SLf8D99kUGE7v7fZ6AQKBgQD1ZX3RAla9hIhxCf+O
3D+I1j2LMrdjAh0ZKKqwMR4JnHX3mjQI6LwqIctPWTU8wYFECSh9klEclSdCa64s
uI/GNpcqPXejd0cAAdqHEEeG5sHMDt0oFSurL4lyud0GtZvwlzLuwEweuDtvT9cJ
Wfvl86uyO36IW8JdvUprYDctrQKBgQDycZ697qutBieZlGkHpnYWUAeImVA878sJ
w44NuXHvMxBPz+lbJGAg8Cn8fcxNAPqHIraK+kx3po8cZGQywKHUWsxi23ozHoxo
+bGqeQb9U661TnfdDspIXia+xilZt3mm5BPzOUuRqlh4Y9SOBpSWRmEhyw76w4ZP
OPxjWYAgwQKBgA/FehSYxeJgRjSdo+MWnK66tjHgDJE8bYpUZsP0JC4R9DL5oiaA
brd2fI6Y+SbyeNBallObt8LSgzdtnEAbjIH8uDJqyOmknNePRvAvR6mP4xyuR+Bv
m+Lgp0DMWTw5J9CKpydZDItc49T/mJ5tPhdFVd+am0NAQnmr1MCZ6nHxAoGABS3Y
LkaC9FdFUUqSU8+Chkd/YbOkuyiENdkvl6t2e52jo5DVc1T7mLiIrRQi4SI8N9bN
/3oJWCT+uaSLX2ouCtNFunblzWHBrhxnZzTeqVq4SLc8aESAnbslKL4i8/+vYZlN
s8xtiNcSvL+lMsOBORSXzpj/4Ot8WwTkn1qyGgECgYBKNTypzAHeLE6yVadFp3nQ
Ckq9yzvP/ib05rvgbvrne00YeOxqJ9gtTrzgh7koqJyX1L4NwdkEza4ilDWpucn0
xiUZS4SoaJq6ZvcBYS62Yr1t8n09iG47YL8ibgtmH3L+svaotvpVxVK+d7BLevA/
ZboOWVe3icTy64BT3OQhmg==
-----END RSA PRIVATE KEY-----`

func fullTLSMask() []string {
	return []string{
		"use_ssl",
		"ssl_ca",
		"ssl_cert",
		"ssl_key",
		"ssl_ca_path",
		"ssl_cert_path",
		"ssl_key_path",
	}
}

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

func TestValidateDataSourceTLSWriteMatchesRuntimeCAPoolParsing(t *testing.T) {
	caBundle := validKeyPEM + "\n" + validCAPEM
	err := validateDataSourceTLSWrite(
		&storepb.DataSource{UseSsl: true, SslCa: caBundle},
		&storepb.DataSource{UseSsl: true, SslCa: caBundle},
		[]string{"ssl_ca"},
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
		&storepb.DataSource{UseSsl: true, SslCertPath: "/tmp/cert.pem", SslKey: validKeyPEM},
		&storepb.DataSource{UseSsl: true, SslCertPath: "/tmp/cert.pem", SslKey: validKeyPEM},
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

func TestValidateDataSourceTLSWriteMatchesRuntimeCertPEMScanning(t *testing.T) {
	certBundle := validKeyPEM + "\n" + validCAPEM
	err := validateDataSourceTLSWrite(
		&storepb.DataSource{UseSsl: true, SslCert: certBundle, SslKeyPath: "/tmp/key.pem"},
		&storepb.DataSource{UseSsl: true, SslCert: certBundle, SslKeyPath: "/tmp/key.pem"},
		[]string{"ssl_cert", "ssl_key_path"},
	)
	require.NoError(t, err)
}

func TestValidateDataSourceTLSWriteRejectsMalformedInlineCertWithKeyPath(t *testing.T) {
	err := validateDataSourceTLSWrite(
		&storepb.DataSource{UseSsl: true, SslCert: "not-a-cert", SslKeyPath: "/tmp/key.pem"},
		&storepb.DataSource{UseSsl: true, SslCert: "not-a-cert", SslKeyPath: "/tmp/key.pem"},
		[]string{"ssl_cert", "ssl_key_path"},
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid ssl_cert PEM")
}

func TestValidateDataSourceTLSWriteRejectsMalformedInlineKeyWithCertPath(t *testing.T) {
	err := validateDataSourceTLSWrite(
		&storepb.DataSource{UseSsl: true, SslCertPath: "/tmp/cert.pem", SslKey: "not-a-key"},
		&storepb.DataSource{UseSsl: true, SslCertPath: "/tmp/cert.pem", SslKey: "not-a-key"},
		[]string{"ssl_cert_path", "ssl_key"},
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid ssl_key PEM")
}

func TestValidateDataSourceTLSWriteMatchesRuntimeKeyPEMScanning(t *testing.T) {
	keyBundle := validCAPEM + "\n" + validKeyPEM
	err := validateDataSourceTLSWrite(
		&storepb.DataSource{UseSsl: true, SslCertPath: "/tmp/cert.pem", SslKey: keyBundle},
		&storepb.DataSource{UseSsl: true, SslCertPath: "/tmp/cert.pem", SslKey: keyBundle},
		[]string{"ssl_cert_path", "ssl_key"},
	)
	require.NoError(t, err)
}

func TestValidateDataSourceTLSWriteAllowsEd25519PrivateKey(t *testing.T) {
	keyPEM := generateEd25519KeyPEM(t)
	err := validateDataSourceTLSWrite(
		&storepb.DataSource{UseSsl: true, SslCertPath: "/tmp/cert.pem", SslKey: keyPEM},
		&storepb.DataSource{UseSsl: true, SslCertPath: "/tmp/cert.pem", SslKey: keyPEM},
		[]string{"ssl_cert_path", "ssl_key"},
	)
	require.NoError(t, err)
}

func generateEd25519KeyPEM(t *testing.T) string {
	t.Helper()

	_, key, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	der, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)
	return string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}))
}

func TestValidateDataSourceTLSWriteRejectsRelativePathOnAdd(t *testing.T) {
	ds := &storepb.DataSource{UseSsl: true, SslCaPath: "ca.pem"}

	err := validateDataSourceTLSWrite(ds, ds, fullTLSMask())
	require.Error(t, err)
	require.Contains(t, err.Error(), "ssl_ca_path must be an absolute path")
}

func TestValidateDataSourceTLSWriteRejectsIncompleteCertPathOnAdd(t *testing.T) {
	ds := &storepb.DataSource{UseSsl: true, SslCertPath: "/tmp/cert.pem"}

	err := validateDataSourceTLSWrite(ds, ds, fullTLSMask())
	require.Error(t, err)
	require.Contains(t, err.Error(), "ssl_cert and ssl_key must be both set or unset")
}

func TestValidateDataSourceTLSWriteAllowsValidPathOnlyCertOnAdd(t *testing.T) {
	ds := &storepb.DataSource{UseSsl: true, SslCertPath: "/tmp/cert.pem", SslKeyPath: "/tmp/key.pem"}

	err := validateDataSourceTLSWrite(ds, ds, fullTLSMask())
	require.NoError(t, err)
}

func TestValidateAndNormalizeDataSourceTLSForReplaceRejectsRelativePath(t *testing.T) {
	dataSources := []*storepb.DataSource{{UseSsl: true, SslCaPath: "ca.pem"}}

	err := validateAndNormalizeDataSourceTLSForReplace(dataSources)
	require.Error(t, err)
	require.Contains(t, err.Error(), "ssl_ca_path must be an absolute path")
}

func TestValidateAndNormalizeDataSourceTLSForReplaceRejectsIncompleteCertPath(t *testing.T) {
	dataSources := []*storepb.DataSource{{UseSsl: true, SslCertPath: "/tmp/cert.pem"}}

	err := validateAndNormalizeDataSourceTLSForReplace(dataSources)
	require.Error(t, err)
	require.Contains(t, err.Error(), "ssl_cert and ssl_key must be both set or unset")
}

func TestValidateAndNormalizeDataSourceTLSForReplaceRejectsSameSlotConflict(t *testing.T) {
	dataSources := []*storepb.DataSource{{UseSsl: true, SslCa: "inline-ca", SslCaPath: "/tmp/ca.pem"}}

	err := validateAndNormalizeDataSourceTLSForReplace(dataSources)
	require.Error(t, err)
	require.Contains(t, err.Error(), "cannot set both ssl_ca and ssl_ca_path")
}

func TestValidateAndNormalizeDataSourceTLSForReplaceAllowsValidPathOnlyCert(t *testing.T) {
	dataSources := []*storepb.DataSource{{UseSsl: true, SslCertPath: "/tmp/cert.pem", SslKeyPath: "/tmp/key.pem"}}

	err := validateAndNormalizeDataSourceTLSForReplace(dataSources)
	require.NoError(t, err)
	require.Equal(t, "/tmp/cert.pem", dataSources[0].GetSslCertPath())
	require.Equal(t, "/tmp/key.pem", dataSources[0].GetSslKeyPath())
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

func TestValidateDataSourceTLSConfigRejectsRelativePath(t *testing.T) {
	err := validateDataSourceTLSConfig(&storepb.DataSource{UseSsl: true, SslCaPath: "ca.pem"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "ssl_ca_path must be an absolute path")
}
