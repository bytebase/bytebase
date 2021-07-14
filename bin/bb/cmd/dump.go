// cmd is the command surface of Bytebase bb tool provided by bytebase.com.
package cmd

import (
	"github.com/bytebase/bytebase/bin/bb/dump"
	"github.com/spf13/cobra"
)

func init() {
	dumpCmd.Flags().StringVar(&databaseType, "type", "mysql", "Database type. (mysql, or pg).")
	dumpCmd.Flags().StringVar(&username, "username", "", "Username to login database. (default mysql:root pg:postgres).")
	dumpCmd.Flags().StringVar(&password, "password", "", "Password to login database.")
	dumpCmd.Flags().StringVar(&hostname, "hostname", "", "Hostname of database.")
	dumpCmd.Flags().StringVar(&port, "port", "", "Port of database. (default mysql:3306 pg:5432).")
	dumpCmd.Flags().StringVar(&database, "database", "", "Database to connect and export.")
	dumpCmd.Flags().StringVar(&directory, "directory", "", "Directory to dump baselines; output to stdout if unspecified.")

	// tls flags for SSL connection.
	dumpCmd.Flags().StringVar(&sslCA, "ssl-ca", "", "CA file in PEM format.")
	dumpCmd.Flags().StringVar(&sslCert, "ssl-cert", "", "X509 cert in PEM format.")
	dumpCmd.Flags().StringVar(&sslKey, "ssl-key", "", "X509 key in PEM format.")

	rootCmd.AddCommand(dumpCmd)
}

var (
	databaseType string
	username     string
	password     string
	hostname     string
	port         string
	database     string
	directory    string

	// SSL flags.
	sslCA               string // server-ca.pem
	sslCert             string // client-cert.pem
	sslKey              string // client-key.pem
	sslVerifyServerName string // The server name to be verified.

	dumpCmd = &cobra.Command{
		Use:   "dump",
		Short: "Exports the schema of a database instance",
		RunE: func(cmd *cobra.Command, args []string) error {
			tlsCfg := dump.TlsConfig{
				SslCA:   sslCA,
				SslCert: sslCert,
				SslKey:  sslKey,
			}
			return dump.Dump(databaseType, username, password, hostname, port, database, directory, tlsCfg)
		},
	}
)
