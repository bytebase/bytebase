// Package cmd is the command surface of Bytebase bb tool provided by bytebase.com.
package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"

	"github.com/bytebase/bytebase/bin/bb/cmdutils"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func newRestoreCmd(ctx context.Context, logger *zap.Logger) *cobra.Command {
	var (
		file string
	)
	restoreCmd := &cobra.Command{
		Use:   "restore",
		Short: "restores the schema of a database instance",
	}
	dbOption := cmdutils.NeedDatabaseDriver(restoreCmd)
	restoreCmd.Flags().StringVar(&file, "file", "", "File to store the dump.")
	if err := restoreCmd.MarkFlagRequired("database"); err != nil {
		panic(err)
	}
	if err := restoreCmd.MarkFlagRequired("file"); err != nil {
		panic(err)
	}

	restoreCmd.RunE = func(cmd *cobra.Command, args []string) error {
		driver, err := dbOption.Connect(ctx, logger, false)
		if err != nil {
			return err
		}
		defer driver.Close(ctx)

		f, err := os.OpenFile(file, os.O_RDONLY, os.ModePerm)
		if err != nil {
			return fmt.Errorf("failed to open dump file %s, got error: %w", file, err)
		}
		defer f.Close()

		return driver.Restore(ctx, bufio.NewScanner(f))
	}
	return restoreCmd
}
