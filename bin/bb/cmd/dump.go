// Package cmd is the command surface of Bytebase bb tool provided by bytebase.com.
package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/bytebase/bytebase/plugin/db"
	"github.com/spf13/cobra"
)

func init() {
	dumpCmd.Flags().StringVar(&databaseType, "type", "mysql", "Database type. (mysql or pg).")
	dumpCmd.Flags().StringVar(&username, "username", username, "Username to login database. (default mysql:root pg:postgres).")
	dumpCmd.Flags().StringVar(&password, "password", password, "Password to login database.")
	dumpCmd.Flags().StringVar(&hostname, "hostname", "", "Hostname of database.")
	dumpCmd.Flags().StringVar(&port, "port", "", "Port of database. (default mysql:3306 pg:5432).")
	dumpCmd.Flags().StringVar(&database, "database", "", "Database to connect and export.")
	dumpCmd.Flags().StringVar(&file, "file", "", "File to store the dump. Output to stdout if unspecified")

	// tls flags for SSL connection.
	dumpCmd.Flags().StringVar(&sslCA, "ssl-ca", "", "CA file in PEM format.")
	dumpCmd.Flags().StringVar(&sslCert, "ssl-cert", "", "X509 cert in PEM format.")
	dumpCmd.Flags().StringVar(&sslKey, "ssl-key", "", "X509 key in PEM format.")

	dumpCmd.Flags().BoolVar(&schemaOnly, "schema-only", false, "Schema only dump.")

	rootCmd.AddCommand(dumpCmd)
}

var (
	dumpCmd = &cobra.Command{
		Use:   "dump",
		Short: "Exports the schema of a database instance",
		RunE: func(cmd *cobra.Command, args []string) error {
			tlsCfg := db.TLSConfig{
				SslCA:   sslCA,
				SslCert: sslCert,
				SslKey:  sslKey,
			}
			return dumpDatabase(context.Background(), databaseType, username, password, hostname, port, database, file, tlsCfg, schemaOnly)
		},
	}
)

// dumpDatabase exports the schema of a database instance.
// When file isn't specified, the schema will be exported to stdout.
func dumpDatabase(ctx context.Context, databaseType, username, password, hostname, port, database, file string, tlsCfg db.TLSConfig, schemaOnly bool) error {
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
	defer db.Close(ctx)

	out := os.Stdout
	if file != "" {
		out, err = os.Create(file)
		if err != nil {
			return fmt.Errorf("failed to create dump file %s, got error: %w", file, err)
		}
	}
	defer out.Close()

	if err := db.Dump(ctx, database, out, schemaOnly); err != nil {
		return fmt.Errorf("failed to create dump %s, got error: %w", file, err)
	}
	return nil
}
