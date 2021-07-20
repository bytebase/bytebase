// cmd is the command surface of Bytebase bb tool provided by bytebase.com.
package cmd

import (
	"bufio"
	"fmt"
	"os"

	"github.com/bytebase/bytebase/bin/bb/connect"
	"github.com/bytebase/bytebase/bin/bb/restore/mysqlrestore"
	"github.com/bytebase/bytebase/bin/bb/restore/pgrestore"
	"github.com/spf13/cobra"
)

func init() {
	restoreCmd.Flags().StringVar(&databaseType, "type", "mysql", "Database type. (mysql, or pg).")
	restoreCmd.Flags().StringVar(&username, "username", "", "Username to login database. (default mysql:root pg:postgres).")
	restoreCmd.Flags().StringVar(&password, "password", "", "Password to login database.")
	restoreCmd.Flags().StringVar(&hostname, "hostname", "", "Hostname of database.")
	restoreCmd.Flags().StringVar(&port, "port", "", "Port of database. (default mysql:3306 pg:5432).")
	// TODO(spinningbot): database and file are required flags.
	restoreCmd.Flags().StringVar(&database, "database", "", "Database to connect and export.")
	restoreCmd.Flags().StringVar(&file, "file", "", "Directory to dump baselines; output to stdout if unspecified.")

	// tls flags for SSL connection.
	restoreCmd.Flags().StringVar(&sslCA, "ssl-ca", "", "CA file in PEM format.")
	restoreCmd.Flags().StringVar(&sslCert, "ssl-cert", "", "X509 cert in PEM format.")
	restoreCmd.Flags().StringVar(&sslKey, "ssl-key", "", "X509 key in PEM format.")

	rootCmd.AddCommand(restoreCmd)
}

var (
	restoreCmd = &cobra.Command{
		Use:   "restore",
		Short: "restores the schema of a database instance",
		RunE: func(cmd *cobra.Command, args []string) error {
			tlsCfg := connect.TlsConfig{
				SslCA:   sslCA,
				SslCert: sslCert,
				SslKey:  sslKey,
			}
			return restoreDatabase(databaseType, username, password, hostname, port, database, file, tlsCfg)
		},
	}
)

// restoreDatabase restores the schema of a database instance.
func restoreDatabase(databaseType, username, password, hostname, port, database, file string, tlsCfg connect.TlsConfig) error {
	f, err := os.OpenFile(file, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return fmt.Errorf("os.OpenFile(%q) error: %v", file, err)
	}
	defer f.Close()
	sc := bufio.NewScanner(f)

	switch databaseType {
	case "mysql":
		if username == "" {
			username = "root"
		}
		if port == "" {
			port = "3306"
		}
		tlsConfig, err := tlsCfg.GetSslConfig()
		if err != nil {
			return fmt.Errorf("TlsConfig.GetSslConfig() got error: %v", err)
		}
		conn, err := connect.NewMysql(username, password, hostname, port, database, tlsConfig)
		if err != nil {
			return fmt.Errorf("connect.NewMysql(%q, %q, %q, %q) got error: %v", username, password, hostname, port, err)
		}
		defer conn.Close()

		if err := mysqlrestore.Restore(conn, sc); err != nil {
			return fmt.Errorf("mysqlrestore.Restore() got error: %v", err)
		}
		return nil
	case "pg":
		conn, err := connect.NewPostgres(username, password, hostname, port, database, tlsCfg.SslCA, tlsCfg.SslCert, tlsCfg.SslKey)
		if err != nil {
			return fmt.Errorf("connect.NewPostgres(%q, %q, %q, %q) got error: %v", username, password, hostname, port, err)
		}
		defer conn.Close()

		if err := conn.SwitchDatabase(database); err != nil {
			return fmt.Errorf("conn.SwitchDatabase(%q) got error: %v", database, err)
		}
		if err := pgrestore.Restore(conn, sc); err != nil {
			return fmt.Errorf("mysqlrestore.Restore() got error: %v", err)
		}
		return nil
	default:
		return fmt.Errorf("database type %q not supported; supported types: mysql, pg.", databaseType)
	}
}
