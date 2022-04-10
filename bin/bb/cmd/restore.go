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
		ds   dataSource
		dsn  string
		file string
	)
	restoreCmd := &cobra.Command{
		Use:   "restore",
		Short: "restores the schema of a database instance",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(dsn) != 0 {
				datasource, err := parseDSN(dsn)
				if err != nil {
					return err
				}
				ds = *datasource
			}
			return restoreDatabase(context.Background(), ds, file)
		},
	}
	restoreCmd.Flags().StringVar(&dsn, "dsn", "", "database connection string. e.g. mysql://root@localhost:3306/bytebase")
	restoreCmd.Flags().StringVar(&ds.driver, "type", "mysql", "Database type. (mysql, or pg).")
	restoreCmd.Flags().StringVar(&ds.username, "username", "", "Username to login database. (default mysql:root pg:postgres).")
	restoreCmd.Flags().StringVar(&ds.password, "password", "", "Password to login database.")
	restoreCmd.Flags().StringVar(&ds.host, "host", "", "Hostname of database.")
	restoreCmd.Flags().StringVar(&ds.port, "port", "", "Port of database. (default mysql:3306 pg:5432).")
	restoreCmd.Flags().StringVar(&ds.database, "database", "", "Database to connect and export.")
	restoreCmd.Flags().StringVar(&file, "file", "", "File to store the dump.")
	if err := restoreCmd.MarkFlagRequired("database"); err != nil {
		panic(err)
	}
	if err := restoreCmd.MarkFlagRequired("file"); err != nil {
		panic(err)
	}

	// tls flags for SSL connection.
	restoreCmd.Flags().StringVar(&ds.sslCA, "ssl-ca", "", "CA file in PEM format.")
	restoreCmd.Flags().StringVar(&ds.sslCert, "ssl-cert", "", "X509 cert in PEM format.")
	restoreCmd.Flags().StringVar(&ds.sslKey, "ssl-key", "", "X509 key in PEM format.")

	return restoreCmd
}

// restoreDatabase restores the schema of a database instance.
func restoreDatabase(ctx context.Context, ds dataSource, file string) error {
	f, err := os.OpenFile(file, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return fmt.Errorf("os.OpenFile(%q) error: %v", file, err)
	}
	defer f.Close()
	sc := bufio.NewScanner(f)

	var dbType db.Type
	switch ds.driver {
	case "mysql":
		dbType = db.MySQL
		if ds.username == "" {
			ds.username = "root"
		}
	case "pg":
		dbType = db.Postgres
	default:
		return fmt.Errorf("database type %q not supported; supported types: mysql, pg", ds.driver)
	}
	db, err := db.Open(
		ctx,
		dbType,
		db.DriverConfig{Logger: logger},
		db.ConnectionConfig{
			Host:     ds.host,
			Port:     ds.port,
			Username: ds.username,
			Password: ds.password,
			Database: ds.database,
			TLSConfig: db.TLSConfig{
				SslCA:   ds.sslCA,
				SslCert: ds.sslCert,
				SslKey:  ds.sslKey,
			},
		},
		db.ConnectionContext{},
	)
	if err != nil {
		return err
	}
	defer db.Close(ctx)

	if err := db.Restore(ctx, sc); err != nil {
		return fmt.Errorf("failed to restore from database dump %s got error: %w", file, err)
	}
	return nil
}
