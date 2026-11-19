package oceanbase

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
)

// TestDriverConfig maps a data source onto the driver contract: the login, the
// database, the application name the /*app=*/ comment carries, and the extra
// parameters the MySQL family passes through.
func TestDriverConfig(t *testing.T) {
	cfg, err := driverConfig(context.Background(), db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Username:                  "root@test",
			Host:                      "127.0.0.1",
			Port:                      "2881",
			ExtraConnectionParameters: map[string]string{"charset": "utf8mb4", "loc": "UTC"},
		},
		Password: "secret",
		ConnectionContext: db.ConnectionContext{
			DatabaseName:  "app",
			EngineVersion: "4.3.5",
		},
	}, &Driver{})
	require.NoError(t, err)
	require.Equal(t, "127.0.0.1", cfg.Host)
	require.Equal(t, "2881", cfg.Port)
	require.Equal(t, "app", cfg.Database)
	require.Equal(t, "root@test", cfg.User)
	require.Equal(t, "secret", cfg.Password)
	require.Equal(t, "bytebase", cfg.ApplicationName)
	require.Equal(t, "4.3.5", cfg.EngineVersion)
	require.Equal(t, map[string]string{"charset": "utf8mb4", "loc": "UTC"}, cfg.Params)
	require.Nil(t, cfg.Dial)
	require.Nil(t, cfg.TLS)
}

// TestDriverConfigRefusesAllowAllFiles keeps the one data source parameter the
// MySQL family refuses, in any spelling.
func TestDriverConfigRefusesAllowAllFiles(t *testing.T) {
	for _, key := range []string{"allowAllFiles", "allowallfiles", " AllowAllFiles "} {
		_, err := driverConfig(context.Background(), db.ConnectionConfig{
			DataSource: &storepb.DataSource{
				Username:                  "root@test",
				Host:                      "127.0.0.1",
				Port:                      "2881",
				ExtraConnectionParameters: map[string]string{key: "true"},
			},
			ConnectionContext: db.ConnectionContext{DatabaseName: "app"},
		}, &Driver{})
		require.ErrorContains(t, err, "is not allowed for security reasons", key)
	}
}

// TestDriverConfigTLS maps the data source's TLS material onto Config.TLS,
// which the driver requires once it is set.
func TestDriverConfigTLS(t *testing.T) {
	cfg, err := driverConfig(context.Background(), db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Username: "root@test",
			Host:     "127.0.0.1",
			Port:     "2881",
			UseSsl:   true,
		},
		ConnectionContext: db.ConnectionContext{DatabaseName: "app"},
	}, &Driver{})
	require.NoError(t, err)
	require.NotNil(t, cfg.TLS)
	tlsConfig, err := cfg.TLS(cfg.Host)
	require.NoError(t, err)
	require.True(t, tlsConfig.InsecureSkipVerify, "verification is off unless the data source asks for it")

	// Without use_ssl the driver negotiates the server's default instead.
	cfg, err = driverConfig(context.Background(), db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Username: "root@test",
			Host:     "127.0.0.1",
			Port:     "2881",
		},
		ConnectionContext: db.ConnectionContext{DatabaseName: "app"},
	}, &Driver{})
	require.NoError(t, err)
	require.Nil(t, cfg.TLS)
}

// TestRDSTLSConfig builds the TLS config an RDS IAM connection always needs:
// an operator-supplied CA wins over the downloaded AWS bundle, and verification
// is the chain-and-hostname callback.
func TestRDSTLSConfig(t *testing.T) {
	dataSource := &storepb.DataSource{
		Username:             "root@test",
		Host:                 "rds.example.com",
		Port:                 "3306",
		VerifyTlsCertificate: true,
		SslCa:                testCACertPEM,
	}
	tlsConfig, err := rdsTLSConfig(context.Background(), dataSource)
	require.NoError(t, err)
	require.NotNil(t, tlsConfig.RootCAs)
	require.NotNil(t, tlsConfig.VerifyPeerCertificate)

	// Verification off still encrypts, and consults no root pool.
	dataSource.VerifyTlsCertificate = false
	tlsConfig, err = rdsTLSConfig(context.Background(), dataSource)
	require.NoError(t, err)
	require.True(t, tlsConfig.InsecureSkipVerify)
	require.Nil(t, tlsConfig.RootCAs)
	require.Nil(t, tlsConfig.VerifyPeerCertificate)

	// A CA the operator supplied but the server cannot parse is an error rather
	// than a silent fallback to the downloaded bundle.
	dataSource.VerifyTlsCertificate = true
	dataSource.SslCa = "not a certificate"
	_, err = rdsRootCAs(context.Background(), dataSource)
	require.ErrorContains(t, err, "failed to parse ssl_ca certificates")
}

// A self-signed certificate's PEM is enough to prove the CA is preferred over
// the downloaded bundle: nothing here verifies a server against it.
const testCACertPEM = `-----BEGIN CERTIFICATE-----
MIIBhTCCASugAwIBAgIQIRi6zePL6mKjOipn+dNuaTAKBggqhkjOPQQDAjASMRAw
DgYDVQQKEwdBY21lIENvMB4XDTE3MTAyMDE5NDMwNloXDTE4MTAyMDE5NDMwNlow
EjEQMA4GA1UEChMHQWNtZSBDbzBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABD0d
7VNhbWvZLWPuj/RtHFjvtJBEwOkhbN/BnnE8rnZR8+sbwnc/KhCk3FhnpHZnQz7B
5aETbbIgmuvewdjvSBSjYzBhMA4GA1UdDwEB/wQEAwICpDATBgNVHSUEDDAKBggr
BgEFBQcDATAPBgNVHRMBAf8EBTADAQH/MCkGA1UdEQQiMCCCDmxvY2FsaG9zdDo1
NDUzgg4xMjcuMC4wLjE6NTQ1MzAKBggqhkjOPQQDAgNIADBFAiEA2zpJEPQyz6/l
Wf86aX6PepsntZv2GYlA5UpabfT2EZICICpJ5h/iI+i341gBmLiAFQOyTDT+/wQc
6MF9+Yw1Yy0t
-----END CERTIFICATE-----`

// TestExplainFormat maps the API's plan format onto the driver's.
func TestExplainFormat(t *testing.T) {
	require.Equal(t, "JSON", explainFormat(v1pb.QueryOption_JSON))
	require.Equal(t, "", explainFormat(v1pb.QueryOption_TEXT))
	require.Equal(t, "", explainFormat(v1pb.QueryOption_EXPLAIN_FORMAT_UNSPECIFIED))
}
