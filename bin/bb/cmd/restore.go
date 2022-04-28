// Package cmd is the command surface of Bytebase bb tool provided by bytebase.com.
package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"

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
				return fmt.Errorf("failed to parse dsn, got error: %w", err)
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
	f, err := os.OpenFile(file, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return fmt.Errorf("os.OpenFile(%q) error: %v", file, err)
	}
	defer f.Close()
	sc := bufio.NewScanner(f)

	db, err := open(ctx, logger, u)
	if err != nil {
		return err
	}
	defer db.Close(ctx)

	if err := db.Restore(ctx, sc); err != nil {
		return fmt.Errorf("failed to restore from database dump %s got error: %w", file, err)
	}
	return nil
}
