// Package cmd is the command surface of Bytebase bb tool provided by bytebase.com.
package cmd

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/bytebase/bytebase/plugin/db"

	// install mysql driver.
	_ "github.com/bytebase/bytebase/plugin/db/mysql"
	"github.com/spf13/cobra"
)

func newDumpCmd() *cobra.Command {
	var (
		ds   dataSource
		dsn  string
		file string

		// Dump options.
		schemaOnly bool
	)
	dumpCmd := &cobra.Command{
		Use:   "dump",
		Short: "Exports the schema of a database instance",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(dsn) != 0 {
				datasource, err := parseDSN(dsn)
				if err != nil {
					return err
				}
				ds = *datasource
			}
			out := cmd.OutOrStdout()
			if file != "" {
				f, err := os.Create(file)
				if err != nil {
					return fmt.Errorf("failed to create dump file %s, got error: %w", file, err)
				}
				defer f.Close()
				out = f
			}
			return dumpDatabase(context.Background(), ds, out, schemaOnly)
		},
	}

	dumpCmd.Flags().StringVar(&dsn, "dsn", "", "database connection string. e.g. mysql://root@localhost:3306/bytebase")
	dumpCmd.Flags().StringVar(&ds.driver, "type", "mysql", "Database type. (mysql or pg).")
	dumpCmd.Flags().StringVar(&ds.username, "username", "", "Username to login database. (default mysql:root pg:postgres).")
	dumpCmd.Flags().StringVar(&ds.password, "password", "", "Password to login database.")
	dumpCmd.Flags().StringVar(&ds.host, "hostname", "", "Hostname of database.")
	dumpCmd.Flags().StringVar(&ds.port, "port", "", "Port of database. (default mysql:3306 pg:5432).")
	dumpCmd.Flags().StringVar(&ds.database, "database", "", "Database to connect and export.")
	dumpCmd.Flags().StringVar(&file, "file", "", "File to store the dump. Output to stdout if unspecified")

	// tls flags for SSL connection.
	dumpCmd.Flags().StringVar(&ds.sslCA, "ssl-ca", "", "CA file in PEM format.")
	dumpCmd.Flags().StringVar(&ds.sslCert, "ssl-cert", "", "X509 cert in PEM format.")
	dumpCmd.Flags().StringVar(&ds.sslKey, "ssl-key", "", "X509 key in PEM format.")

	dumpCmd.Flags().BoolVar(&schemaOnly, "schema-only", false, "Schema only dump.")

	return dumpCmd
}

// dumpDatabase exports the schema of a database instance.
// When file isn't specified, the schema will be exported to stdout.
func dumpDatabase(ctx context.Context, ds dataSource, out io.Writer, schemaOnly bool) error {
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

	if err := db.Dump(ctx, ds.database, out, schemaOnly); err != nil {
		return fmt.Errorf("failed to create dump, got error: %w", err)
	}
	return nil
}
