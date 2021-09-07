// cmd is the command surface of Bytebase bb tool provided by bytebase.com.
package cmd

var (
	databaseType          string
	username              string
	password              string
	hostname              string
	port                  string
	database              string
	resultFileOrDirectory string
	file                  string

	// SSL flags.
	sslCA               string // server-ca.pem
	sslCert             string // client-cert.pem
	sslKey              string // client-key.pem
	sslVerifyServerName string // The server name to be verified.

	// Dump options.
	schemaOnly bool
)
