// Package cmd is the command surface of Bytebase bb tool provided by bytebase.com.
package cmd

import (
	"context"
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/xo/dburl"
)

func newRestoreCmd() *cobra.Command {
	var (
		dsn  string
		file string
	)
	restoreCmd := &cobra.Command{
		Use:   "restore",
		Short: "Restores schema and data of a database.",
		RunE: func(_ *cobra.Command, _ []string) error {
			u, err := dburl.Parse(dsn)
			if err != nil {
				return errors.Wrap(err, "failed to parse dsn")
			}
			return restoreDatabase(context.Background(), u, file)
		},
	}
	restoreCmd.Flags().StringVar(&dsn, "dsn", "", dsnUsage)
	restoreCmd.Flags().StringVar(&file, "file", "", "File to store the dump.")
	if err := restoreCmd.MarkFlagRequired("file"); err != nil {
		panic(err)
	}

	return restoreCmd
}

// restoreDatabase restores the schema of a database instance.
func restoreDatabase(ctx context.Context, u *dburl.URL, file string) error {
	f, err := os.Open(file)
	if err != nil {
		return errors.Wrapf(err, "failed to open file %q", file)
	}
	defer f.Close()

	db, err := open(ctx, u)
	if err != nil {
		return err
	}
	defer db.Close(ctx)

	if err := db.Restore(ctx, f); err != nil {
		return errors.Wrapf(err, "failed to restore from backup file %q", file)
	}
	return nil
}
