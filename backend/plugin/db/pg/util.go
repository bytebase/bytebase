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
	if tlsConfig.SslCA != "" || tlsConfig.SslCert != "" || tlsConfig.SslKey != "" {
		sslMode = SSLModeVerifyFull
		// If users use TLS/SSL with SSH tunneling together, the TLS/SSL handshake will be established between the localhost and the remote server.
		// Then, we connect to the db server via the SSH tunnel(localhost), so we do not verify the SAN/CN of the certificate to avoid the hostname mismatch error.
		if sshConfig.Host != "" {
			sslMode = SSLModeVerifyCA
		}
	}
	return sslMode
}
