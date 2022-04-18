// Package cmd is the command surface of Bytebase bb tool provided by bytebase.com.
package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"

	"github.com/bytebase/bytebase/plugin/db"
	"github.com/spf13/cobra"
)

func newRestoreCmd() *cobra.Command {
	var (
		databaseType string
		username     string
		password     string
		hostname     string
		port         string
		database     string
		file         string

		// SSL flags.
		sslCA   string // server-ca.pem
		sslCert string // client-cert.pem
		sslKey  string // client-key.pem
	)
	restoreCmd := &cobra.Command{
		Use:   "restore",
		Short: "restores the schema of a database instance",
		RunE: func(cmd *cobra.Command, args []string) error {
			tlsCfg := db.TLSConfig{
				SslCA:   sslCA,
				SslCert: sslCert,
				SslKey:  sslKey,
			}
			return restoreDatabase(context.Background(), databaseType, username, password, hostname, port, database, file, tlsCfg)
		},
	}
	restoreCmd.Flags().StringVar(&databaseType, "type", "mysql", "Database type. (mysql, or pg).")
	restoreCmd.Flags().StringVar(&username, "username", "", "Username to login database. (default mysql:root pg:postgres).")
	restoreCmd.Flags().StringVar(&password, "password", "", "Password to login database.")
	restoreCmd.Flags().StringVar(&hostname, "hostname", "", "Hostname of database.")
	restoreCmd.Flags().StringVar(&port, "port", "", "Port of database. (default mysql:3306 pg:5432).")
	restoreCmd.Flags().StringVar(&database, "database", "", "Database to connect and export.")
	restoreCmd.Flags().StringVar(&file, "file", "", "File to store the dump.")
	if err := restoreCmd.MarkFlagRequired("database"); err != nil {
		panic(err)
	}
	if err := restoreCmd.MarkFlagRequired("file"); err != nil {
		panic(err)
	}

	// tls flags for SSL connection.
	restoreCmd.Flags().StringVar(&sslCA, "ssl-ca", "", "CA file in PEM format.")
	restoreCmd.Flags().StringVar(&sslCert, "ssl-cert", "", "X509 cert in PEM format.")
	restoreCmd.Flags().StringVar(&sslKey, "ssl-key", "", "X509 key in PEM format.")

	return restoreCmd
}

// restoreDatabase restores the schema of a database instance.
func restoreDatabase(ctx context.Context, databaseType, username, password, hostname, port, database, file string, tlsCfg db.TLSConfig) error {
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
	driver, err := db.Open(
		ctx,
		dbType,
		db.DriverConfig{Logger: logger},
		db.ConnectionConfig{
			Host:      hostname,
			Port:      port,
			Username:  username,
			Password:  password,
			Database:  database,
			TLSConfig: tlsCfg,
		},
		db.ConnectionContext{},
	)
	if err != nil {
		return err
	}
	defer driver.Close(ctx)

	if err := driver.Restore(ctx, sc); err != nil {
		return fmt.Errorf("failed to restore from database dump %s got error: %w", file, err)
	}
	return nil
}
