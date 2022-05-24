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
	"github.com/spf13/cobra"
	"github.com/xo/dburl"
)

func newMigrateCmd() *cobra.Command {
	var (
		dsn         string
		fileList    []string
		commandList []string
		description string
		issueID     string
	)
	migrateCmd := &cobra.Command{
		Use:   "migrate",
		Short: "Migrate the database schema.",
		RunE: func(_ *cobra.Command, _ []string) error {
			u, err := dburl.Parse(dsn)
			if err != nil {
				return fmt.Errorf("failed to parse dsn, got error: %w", err)
			}

			var sqlReaders []io.Reader

			// TODO(qsliu): support file and command combined as the passed order.
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
			return migrateDatabase(context.Background(), u, description, issueID, false /*createDatabase*/, sqlReader)
		}}

	migrateCmd.Flags().StringVar(&dsn, "dsn", "", dsnUsage)
	migrateCmd.Flags().StringSliceVarP(&fileList, "file", "f", []string{}, "SQL file to execute.")
	migrateCmd.Flags().StringSliceVarP(&commandList, "command", "c", []string{}, "SQL command to execute.")
	migrateCmd.Flags().StringVar(&description, "description", "", "Description of migration.")
	migrateCmd.Flags().StringVar(&issueID, "issue-id", "", "Issue ID of migration.")
	return migrateCmd
}

func migrateDatabase(ctx context.Context, u *dburl.URL, description, issueID string, createDatabase bool, sqlReader io.Reader) error {
	driver, err := open(ctx, logger, u)
	if err != nil {
		return err
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
		Database:       getDatabase(u),
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
