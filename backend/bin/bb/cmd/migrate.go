package cmd

import (
	"bytes"
	"context"
	"io"
	"os"
	"os/user"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/xo/dburl"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/plugin/db"
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
				return errors.Wrap(err, "failed to parse dsn")
			}

			var sqlReaders []io.Reader

			// TODO(qsliu): support file and command combined as the passed order.
			for _, file := range fileList {
				f, err := os.Open(file)
				if err != nil {
					return err
				}
				//nolint:revive
				// f.Close() is intended to be deferred to the end of the function.
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
	driver, err := open(ctx, u)
	if err != nil {
		return err
	}
	defer driver.Close(ctx)

	if err := driver.SetupMigrationIfNeeded(ctx); err != nil {
		return errors.Wrap(err, "failed to setup migration")
	}

	migrationCreator := "bb-unknown-creator"
	if currentUser, err := user.Current(); err == nil {
		migrationCreator = currentUser.Username
	}

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, sqlReader); err != nil {
		return errors.Wrap(err, "failed to read sql file")
	}
	// TODO(d): support semantic versioning.
	if _, _, err := driver.ExecuteMigration(ctx, nil, &db.MigrationInfo{
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
		return errors.Wrap(err, "failed to migrate database")
	}
	return nil
}
