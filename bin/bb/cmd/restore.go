// cmd is the command surface of Bytebase bb tool provided by bytebase.com.
package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"

	"github.com/bytebase/bytebase/plugin/db"
	"github.com/spf13/cobra"
)

func init() {
	restoreCmd.Flags().StringVar(&databaseType, "type", "mysql", "Database type. (mysql, or pg).")
	restoreCmd.Flags().StringVar(&username, "username", "", "Username to login database. (default mysql:root pg:postgres).")
	restoreCmd.Flags().StringVar(&password, "password", "", "Password to login database.")
	restoreCmd.Flags().StringVar(&hostname, "hostname", "", "Hostname of database.")
	restoreCmd.Flags().StringVar(&port, "port", "", "Port of database. (default mysql:3306 pg:5432).")
	restoreCmd.Flags().StringVar(&database, "database", "", "Database to connect and export.")
	restoreCmd.Flags().StringVar(&file, "file", "", "File to store the dump.")
	restoreCmd.MarkFlagRequired("database")
	restoreCmd.MarkFlagRequired("file")

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
			tlsCfg := db.TlsConfig{
				SslCA:   sslCA,
				SslCert: sslCert,
				SslKey:  sslKey,
			}
			return restoreDatabase(databaseType, username, password, hostname, port, database, file, tlsCfg)
		},
	}
)

// restoreDatabase restores the schema of a database instance.
func restoreDatabase(databaseType, username, password, hostname, port, database, file string, tlsCfg db.TlsConfig) error {
	f, err := os.OpenFile(file, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return fmt.Errorf("os.OpenFile(%q) error: %v", file, err)
	}
	defer f.Close()
	sc := bufio.NewScanner(f)

	var dbType db.Type
	switch databaseType {
	case "mysql":
		dbType = db.MySQL
		if username == "" {
			username = "root"
		}
	case "pg":
		dbType = db.Postgres
	default:
		return fmt.Errorf("database type %q not supported; supported types: mysql, pg", databaseType)
	}
	db, err := db.Open(
		dbType,
		db.DriverConfig{Logger: logger},
		db.ConnectionConfig{
			Host:      hostname,
			Port:      port,
			Username:  username,
			Password:  password,
			Database:  database,
			TlsConfig: tlsCfg,
		},
		db.ConnectionContext{},
	)
	if err != nil {
		return err
	}
	defer db.Close(context.Background())

	if err := db.Restore(context.Background(), sc); err != nil {
		return fmt.Errorf("failed to restore from database dump %s got error: %w", file, err)
	}
	return nil
}
