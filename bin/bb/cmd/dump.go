// Package cmd is the command surface of Bytebase bb tool provided by bytebase.com.
package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/bytebase/bytebase/bin/bb/cmdutils"
	"go.uber.org/zap"

	// install mysql driver.
	_ "github.com/bytebase/bytebase/plugin/db/mysql"
	"github.com/spf13/cobra"
)

func newDumpCmd(ctx context.Context, logger *zap.Logger) *cobra.Command {
	var (
		file string
		// Dump options.
		schemaOnly bool
	)
	dumpCmd := &cobra.Command{
		Use:   "dump",
		Short: "Exports the schema of a database instance",
	}
	dbOption := cmdutils.NeedDatabaseDriver(dumpCmd)
	dumpCmd.Flags().StringVarP(&file, "file", "f", "", "File to store the dump. Output to stdout if unspecified")
	dumpCmd.Flags().BoolVar(&schemaOnly, "schema-only", false, "Schema only dump.")

	dumpCmd.RunE = func(cmd *cobra.Command, args []string) error {
		driver, err := dbOption.Connect(ctx, logger, schemaOnly)
		if err != nil {
			return err
		}
		defer driver.Close(ctx)

		out := cmd.OutOrStdout()
		if file != "" {
			f, err := os.Create(file)
			if err != nil {
				return fmt.Errorf("failed to create dump file %s, got error: %w", file, err)
			}
			defer f.Close()
			out = f
		}

		return driver.Dump(ctx, dbOption.Database, out, schemaOnly)
	}
	return dumpCmd
}
