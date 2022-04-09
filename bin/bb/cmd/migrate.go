package cmd

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/user"
	"strings"

	"github.com/bytebase/bytebase/common"
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
		fileList     []string
		commandList  []string
		description  string
		issueID      string

		// SSL flags.
		sslCA   string // server-ca.pem
		sslCert string // client-cert.pem
		sslKey  string // client-key.pem
	)
	migrateCmd := &cobra.Command{
		Use:   "migrate",
		Short: "Migrate the database schema",
		RunE: func(cmd *cobra.Command, args []string) error {
			tlsCfg := db.TLSConfig{
				SslCA:   sslCA,
				SslCert: sslCert,
				SslKey:  sslKey,
			}

			var sqlReaders []io.Reader

			//TODO(qsliu): support file and command combined as the passed order.
			for _, file := range fileList {
				f, err := os.Open(file)
				if err != nil {
					return err
				}
				defer f.Close()
				sqlReaders = append(sqlReaders, f)
			}

			for _, command := range commandList {
				sqlReaders = append(sqlReaders, strings.NewReader(command))
			}

			sqlReader := io.MultiReader(sqlReaders...)
			return migrateDatabase(context.Background(), databaseType, username, password, hostname, port, database, description, issueID, false /*createDatabase*/, sqlReader, tlsCfg)
		}}

	migrateCmd.Flags().StringVar(&databaseType, "type", "mysql", "Database type. (mysql or pg).")
	migrateCmd.Flags().StringVar(&username, "username", "", "Database username. (default mysql:root pg:postgres).")
	migrateCmd.Flags().StringVar(&password, "password", "", "Database password.")
	migrateCmd.Flags().StringVar(&hostname, "host", "", "Database host.")
	migrateCmd.Flags().StringVar(&port, "port", "", "Port of database. (default mysql:3306 pg:5432).")
	migrateCmd.Flags().StringVar(&database, "database", "", "Target database to execute migration.")
	migrateCmd.Flags().StringSliceVarP(&fileList, "file", "f", []string{}, "SQL file to execute.")
	migrateCmd.Flags().StringSliceVarP(&commandList, "command", "c", []string{}, "SQL command to execute.")
	migrateCmd.Flags().StringVar(&description, "description", "", "Description of migration.")
	migrateCmd.Flags().StringVar(&issueID, "issue-id", "", "Issue ID of migration.")
	// tls flags for SSL connection.
	migrateCmd.Flags().StringVar(&sslCA, "ssl-ca", "", "CA file in PEM format.")
	migrateCmd.Flags().StringVar(&sslCert, "ssl-cert", "", "X509 cert in PEM format.")
	migrateCmd.Flags().StringVar(&sslKey, "ssl-key", "", "X509 key in PEM format.")

	return migrateCmd
}

func migrateDatabase(ctx context.Context, databaseType, username, password, hostname, port, database, description, issueID string, createDatabase bool, sqlReader io.Reader, tlsCfg db.TLSConfig) error {
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
		Host:      hostname,
		Port:      port,
		Username:  username,
		Password:  password,
		Database:  database,
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
	// TODO(d): support semantic versioning.
	if _, _, err := driver.ExecuteMigration(ctx, &db.MigrationInfo{
		ReleaseVersion: version,
		Version:        common.DefaultMigrationVersion(),
		Database:       database,
		Source:         db.LIBRARY,
		Type:           db.Migrate,
		Description:    description,
		Creator:        migrationCreator,
		IssueID:        issueID,
		CreateDatabase: createDatabase,
	}, buf.String()); err != nil {
		return fmt.Errorf("failed to migrate database, got error: %w", err)
	}
	return nil
}
