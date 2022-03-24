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
		fileList    []string
		commandList []string
		description string
		issueID     string
	)
	migrateCmd := &cobra.Command{
		Use:   "migrate",
		Short: "Migrate the database schema",
	}
	dbFlags := cmdutils.AddDatabaseFlags(migrateCmd)
	migrateCmd.Flags().StringSliceVarP(&fileList, "file", "f", []string{}, "SQL file to execute.")
	migrateCmd.Flags().StringSliceVarP(&commandList, "command", "c", []string{}, "SQL command to execute.")
	migrateCmd.Flags().StringVar(&description, "description", "", "Description of migration.")
	migrateCmd.Flags().StringVar(&issueID, "issue-id", "", "Issue ID of migration.")

	migrateCmd.RunE = func(cmd *cobra.Command, args []string) error {
		driver, err := dbFlags.Connect(ctx, logger, false)
		if err != nil {
			return err
		}
		defer driver.Close(ctx)

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
		return migrateDatabase(context.Background(), driver, dbFlags.Database, description, issueID, false /*createDatabase*/, sqlReader)
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
