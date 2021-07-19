// dump is a library for dumping database schemas provided by bytebase.com.
package dump

import (
	"fmt"
	"os"
	"path"

	"github.com/bytebase/bytebase/bin/bb/connect"
	"github.com/bytebase/bytebase/bin/bb/dump/mysqldump"
	"github.com/bytebase/bytebase/bin/bb/dump/pgdump"
)

// Dump exports the schema of a database instance.
// All non-system databases will be exported to the input directory in the format of database_name.sql for each database.
// When directory isn't specified, the schema will be exported to stdout.
func Dump(databaseType, username, password, hostname, port, database, directory string, tlsCfg connect.TlsConfig, schemaOnly bool) error {
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
		conn, err := connect.NewMysql(username, password, hostname, port, tlsConfig)
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
