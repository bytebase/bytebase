package pg

import "github.com/bytebase/bytebase/backend/plugin/db"

// sslMode is the PGSSLMode type.
// https://www.postgresql.org/docs/current/libpq-connect.html#LIBPQ-CONNECT-SSLMODE
type sslMode string

const (
	SSLModeDisable sslMode = "disable"
	SSLModeAllow   sslMode = "allow"
	// It is the default mode of sslmode.
	// https://www.postgresql.org/docs/current/libpq-ssl.html
	SSLModePrefer     sslMode = "prefer"
	SSLModeRequire    sslMode = "require"
	SSLModeVerifyCA   sslMode = "verify-ca"
	SSLModeVerifyFull sslMode = "verify-full"
)

func getSSLMode(tlsConfig db.TLSConfig, sshConfig db.SSHConfig) sslMode {
	sslMode := SSLModePrefer
	if tlsConfig.UseSSL {
		sslMode = SSLModeVerifyFull
		if tlsConfig.SslCA != "" {
			if sshConfig.Host != "" {
				sslMode = SSLModeVerifyCA
			}
		}
	}
	return sslMode
}
