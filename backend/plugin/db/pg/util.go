package pg

import (
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

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

func getSSLMode(ds *storepb.DataSource) sslMode {
	sslMode := SSLModePrefer
	if ds.GetUseSsl() {
		sslMode = SSLModeVerifyFull
		if ds.GetSslCa() != "" {
			if ds.GetSshHost() != "" {
				sslMode = SSLModeVerifyCA
			}
		}
	}
	return sslMode
}
