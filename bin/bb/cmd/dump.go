// cmd is the command surface of Bytebase bb tool provided by bytebase.com.
package cmd

import (
	"fmt"
	"os"
	"path"

	"github.com/bytebase/bytebase/bin/bb/connect"
	"github.com/bytebase/bytebase/bin/bb/dump/mysqldump"
	"github.com/bytebase/bytebase/bin/bb/dump/pgdump"
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

	dumpCmd.Flags().BoolVar(&schemaOnly, "schema-only", false, "Schema only dump.")

	rootCmd.AddCommand(dumpCmd)
}

var (
	dumpCmd = &cobra.Command{
		Use:   "dump",
		Short: "Exports the schema of a database instance",
		RunE: func(cmd *cobra.Command, args []string) error {
			tlsCfg := connect.TlsConfig{
				SslCA:   sslCA,
				SslCert: sslCert,
				SslKey:  sslKey,
			}
			return dumpDatabase(databaseType, username, password, hostname, port, database, directory, tlsCfg, schemaOnly)
		},
	}
)

// dumpDatabase exports the schema of a database instance.
// All non-system databases will be exported to the input directory in the format of database_name.sql for each database.
// When directory isn't specified, the schema will be exported to stdout.
func dumpDatabase(databaseType, username, password, hostname, port, database, directory string, tlsCfg connect.TlsConfig, schemaOnly bool) error {
	if directory != "" {
		dirInfo, err := os.Stat(directory)
		if os.IsNotExist(err) {
			return fmt.Errorf("directory %q does not exist", directory)
		}
		if !dirInfo.IsDir() {
			return fmt.Errorf("path %q isn't a directory", directory)
		}
	}

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
		conn, err := connect.NewMysql(username, password, hostname, port, "" /* database */, tlsConfig)
		if err != nil {
			return fmt.Errorf("connect.NewMysql(%q, %q, %q, %q) got error: %v", username, password, hostname, port, err)
		}
		defer conn.Close()

		dp := mysqldump.New(conn)
		databases, err := dp.GetDumpableDatabases(database)
		if err != nil {
			return err
		}
		for _, dbName := range databases {
			out, err := getOutFile(dbName, directory)
			if err != nil {
				return fmt.Errorf("getOutFile(%s, %s) got error: %s", dbName, directory, err)
			}
			defer out.Close()

			if err := dp.Dump(dbName, out, schemaOnly); err != nil {
				return err
			}
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
			out, err := getOutFile(dbName, directory)
			if err != nil {
				return fmt.Errorf("getOutFile(%s, %s) got error: %s", dbName, directory, err)
			}
			defer out.Close()

			if err := dp.Dump(dbName, out, schemaOnly); err != nil {
				return err
			}
		}
		return nil
	default:
		return fmt.Errorf("database type %q not supported; supported types: mysql, pg.", databaseType)
	}
}

// getOutFile gets the file descriptor to export the dump.
func getOutFile(dbName, directory string) (*os.File, error) {
	if directory == "" {
		return os.Stdout, nil
	} else {
		path := path.Join(directory, fmt.Sprintf("%s.sql", dbName))
		f, err := os.Create(path)
		if err != nil {
			return nil, err
		}
		return f, nil
	}
}
