// cmd is the command surface of Bytebase bb tool provided by bytebase.com.
package cmd

import (
	"context"
	"fmt"
	"os"
	"path"

	"github.com/bytebase/bytebase/bin/bb/connect"
	"github.com/bytebase/bytebase/bin/bb/dump/pgdump"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/spf13/cobra"
)

func init() {
	dumpCmd.Flags().StringVar(&databaseType, "type", "mysql", "Database type. (mysql or pg).")
	dumpCmd.Flags().StringVar(&username, "username", "", "Username to login database. (default mysql:root pg:postgres).")
	dumpCmd.Flags().StringVar(&password, "password", "", "Password to login database.")
	dumpCmd.Flags().StringVar(&hostname, "hostname", "", "Hostname of database.")
	dumpCmd.Flags().StringVar(&port, "port", "", "Port of database. (default mysql:3306 pg:5432).")
	dumpCmd.Flags().StringVar(&database, "database", "", "Database to connect and export.")
	dumpCmd.Flags().StringVar(&fileOrDirectory, "file", "",
		`Result file or directory to store the dump.
For MySQL, it behaves like mysqldump --result-file; Output to stdout if unspecified.
For PostgreSQL, it behaves like pgdump --file; Output to stdout if unspecified. If specified and when --database is not specified, it points to the directory containing files for each database dump.`,
	)

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
			tlsCfg := db.TlsConfig{
				SslCA:   sslCA,
				SslCert: sslCert,
				SslKey:  sslKey,
			}
			return dumpDatabase(databaseType, username, password, hostname, port, database, fileOrDirectory, tlsCfg, schemaOnly)
		},
	}
)

// dumpDatabase exports the schema of a database instance.
// When fileOrDirectory isn't specified, the schema will be exported to stdout.
func dumpDatabase(databaseType, username, password, hostname, port, database, fileOrDirectory string, tlsCfg db.TlsConfig, schemaOnly bool) error {
	// For PostgreSQL, if we export all databases, then we treat fileOrDirectory as a directory
	// and check its existence before proceeding.
	if databaseType == "pg" && database == "" {
		dirInfo, err := os.Stat(fileOrDirectory)
		if os.IsNotExist(err) {
			return fmt.Errorf("directory %q does not exist", fileOrDirectory)
		}
		if !dirInfo.IsDir() {
			return fmt.Errorf("path %q isn't a directory", fileOrDirectory)
		}
	}

	dumpAll := database == ""
	switch databaseType {
	case "mysql":
		if username == "" {
			username = "root"
		}

		db, err := db.Open(
			db.Mysql,
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

		out := os.Stdout
		if fileOrDirectory != "" {
			out, err = os.Create(fileOrDirectory)
			if err != nil {
				return fmt.Errorf("failed to create dump file %s, got error: %w", fileOrDirectory, err)
			}
		}
		defer out.Close()

		if err := db.Dump(context.Background(), database, out, schemaOnly); err != nil {
			return fmt.Errorf("failed to create dump %s, got error: %w", fileOrDirectory, err)
		}
		return nil
	case "pg":
		conn, err := connect.NewPostgres(username, password, hostname, port, database, tlsCfg.SslCA, tlsCfg.SslCert, tlsCfg.SslKey)
		if err != nil {
			return fmt.Errorf("connect.NewPostgres(%q, %q, %q, %q) got error: %v", username, password, hostname, port, err)
		}
		defer conn.Close()

		dp := pgdump.New(conn)
		databases, err := dp.GetDumpableDatabases(database)
		if err != nil {
			return err
		}
		for _, dbName := range databases {
			out := os.Stdout
			if fileOrDirectory != "" {
				outputFileOrDirectory := fileOrDirectory
				if database == "" {
					outputFileOrDirectory = path.Join(fileOrDirectory, fmt.Sprintf("%s.sql", dbName))
				}

				out, err = os.Create(outputFileOrDirectory)
				if err != nil {
					return fmt.Errorf("failed to create database dump file %s, got error: %s", outputFileOrDirectory, err)
				}
			}
			defer out.Close()

			if err := dp.Dump(dbName, out, schemaOnly, dumpAll); err != nil {
				return err
			}
		}
		return nil
	default:
		return fmt.Errorf("database type %q not supported; supported types: mysql, pg", databaseType)
	}
}
