package cmd

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/user"
	"time"

	"github.com/bytebase/bytebase/plugin/db"

	// install mysql driver.
	_ "github.com/bytebase/bytebase/plugin/db/mysql"
	"github.com/spf13/cobra"
)

func newMigrateCmd() *cobra.Command {
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
	migrateCmd := &cobra.Command{
		Use:   "migrate",
		Short: "Migrates the database schema",
		RunE: func(cmd *cobra.Command, args []string) error {
			tlsCfg := db.TLSConfig{
				SslCA:   sslCA,
				SslCert: sslCert,
				SslKey:  sslKey,
			}
			f, err := os.Open(file)
			if err != nil {
				return fmt.Errorf("failed to open sql file %s, got error: %w", file, err)
			}
			defer f.Close()
			return migrateDatabase(context.Background(), databaseType, username, password, hostname, port, database, false /*createDatabase*/, f, tlsCfg)
		}}

	migrateCmd.Flags().StringVar(&databaseType, "type", "mysql", "Database type. (mysql or pg).")
	migrateCmd.Flags().StringVar(&username, "username", "", "Username to login database. (default mysql:root pg:postgres).")
	migrateCmd.Flags().StringVar(&password, "password", "", "Password to login database.")
	migrateCmd.Flags().StringVar(&hostname, "hostname", "", "Hostname of database.")
	migrateCmd.Flags().StringVar(&port, "port", "", "Port of database. (default mysql:3306 pg:5432).")
	migrateCmd.Flags().StringVar(&database, "database", "", "Database to execute migration.")
	migrateCmd.Flags().StringVar(&file, "sql", "", "File that stores the migration script.")
	// tls flags for SSL connection.
	migrateCmd.Flags().StringVar(&sslCA, "ssl-ca", "", "CA file in PEM format.")
	migrateCmd.Flags().StringVar(&sslCert, "ssl-cert", "", "X509 cert in PEM format.")
	migrateCmd.Flags().StringVar(&sslKey, "ssl-key", "", "X509 key in PEM format.")

	return migrateCmd
}

func migrateDatabase(ctx context.Context, databaseType, username, password, hostname, port, database string, createDatabase bool, sqlReader io.Reader, tlsCfg db.TLSConfig) error {
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

	driver, err := db.Open(ctx, dbType, db.DriverConfig{Logger: logger}, db.ConnectionConfig{
		Host:     hostname,
		Port:     port,
		Username: username,
		Password: password,
		// Database:  database,
		TLSConfig: tlsCfg,
	}, db.ConnectionContext{})
	if err != nil {
		return fmt.Errorf("failed to open database, got error: %w", err)
	}
	defer driver.Close(ctx)

	if err := driver.SetupMigrationIfNeeded(ctx); err != nil {
		return fmt.Errorf("failed to setup migration, got error: %w", err)
	}

	migrationCreator := "bb-unknown-creator"
	if currentUser, err := user.Current(); err == nil {
		migrationCreator = currentUser.Username
	}

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, sqlReader); err != nil {
		return fmt.Errorf("failed to read sql file, got error: %w", err)
	}
	if _, _, err := driver.ExecuteMigration(ctx, &db.MigrationInfo{
		ReleaseVersion: version,
		Version:        defaultMigrationVersion(),
		Database:       database,
		Source:         db.LIBRARY,
		Type:           db.Migrate,
		Description:    "",
		Creator:        migrationCreator,
		IssueID:        "",
		CreateDatabase: createDatabase,
	}, buf.String()); err != nil {
		return fmt.Errorf("failed to migrate database, got error: %w", err)
	}
	return nil
}

func defaultMigrationVersion() string {
	return time.Now().Format("20060102150405")
}
