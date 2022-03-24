package cmd

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/user"
	"strings"
	"time"

	"github.com/bytebase/bytebase/bin/bb/cmdutils"
	"github.com/bytebase/bytebase/plugin/db"
	"go.uber.org/zap"

	// install mysql driver.
	_ "github.com/bytebase/bytebase/plugin/db/mysql"
	"github.com/spf13/cobra"
)

func newMigrateCmd(ctx context.Context, logger *zap.Logger) *cobra.Command {
	var (
		file        string
		command     string
		description string
		issueID     string
	)
	migrateCmd := &cobra.Command{
		Use:   "migrate",
		Short: "Migrate the database schema",
	}
	dbFlags := cmdutils.AddDatabaseFlags(migrateCmd)
	migrateCmd.Flags().StringVarP(&file, "file", "f", "", "SQL file to execute.")
	migrateCmd.Flags().StringVarP(&command, "command", "c", "", "SQL command to execute.")
	migrateCmd.Flags().StringVar(&description, "description", "", "Description of migration.")
	migrateCmd.Flags().StringVar(&issueID, "issue-id", "", "Issue ID of migration.")

	migrateCmd.RunE = func(cmd *cobra.Command, args []string) error {
		driver, err := dbFlags.Connect(ctx, logger, false)
		if err != nil {
			return err
		}
		defer driver.Close(ctx)

		if file != "" {
			f, err := os.Open(file)
			if err != nil {
				return fmt.Errorf("failed to open file %s, got error: %w", file, err)
			}
			defer f.Close()
			return migrateDatabase(ctx, driver, dbFlags.Database, description, issueID, false, f)
		}

		if command != "" {
			return migrateDatabase(ctx, driver, dbFlags.Database, description, issueID, false, strings.NewReader(command))
		}

		return fmt.Errorf("no file or command specified")
	}

	return migrateCmd
}

func migrateDatabase(ctx context.Context, driver db.Driver, database, description, issueID string, createDatabase bool, sqlReader io.Reader) error {
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
		Description:    description,
		Creator:        migrationCreator,
		IssueID:        issueID,
		CreateDatabase: createDatabase,
	}, buf.String()); err != nil {
		return fmt.Errorf("failed to migrate database, got error: %w", err)
	}
	return nil
}

func defaultMigrationVersion() string {
	return time.Now().Format("20060102150405")
}
