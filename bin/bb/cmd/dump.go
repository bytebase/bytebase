// Package cmd is the command surface of Bytebase bb tool provided by bytebase.com.
package cmd

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/xo/dburl"

	// install mysql driver.
	_ "github.com/bytebase/bytebase/plugin/db/mysql"
	"github.com/spf13/cobra"
)

func newDumpCmd() *cobra.Command {
	var (
		dsn  string
		file string

		// Dump options.
		schemaOnly bool
	)
	dumpCmd := &cobra.Command{
		Use:   "dump",
		Short: "Exports the schema of a database instance",
		RunE: func(cmd *cobra.Command, _ []string) error {
			u, err := dburl.Parse(dsn)
			if err != nil {
				return fmt.Errorf("failed to parse dsn, got error: %w", err)
			}
			out := cmd.OutOrStdout()
			if file != "" {
				f, err := os.Create(file)
				if err != nil {
					return fmt.Errorf("failed to create dump file %s, got error: %w", file, err)
				}
				defer f.Close()
				out = f
			}
			return dumpDatabase(context.Background(), u, out, schemaOnly)
		},
	}

	dumpCmd.Flags().StringVar(&dsn, "dsn", "", "Database connection string. e.g. mysql://username:password@host:port/dbname?ssl-ca=value1&ssl-cert=value2&ssl-key=value3")
	dumpCmd.Flags().StringVar(&file, "file", "", "File to store the dump. Output to stdout if unspecified")
	dumpCmd.Flags().BoolVar(&schemaOnly, "schema-only", false, "Schema only dump.")
	return dumpCmd
}

// dumpDatabase exports the schema of a database instance.
// When file isn't specified, the schema will be exported to stdout.
func dumpDatabase(ctx context.Context, u *dburl.URL, out io.Writer, schemaOnly bool) error {
	db, err := open(ctx, logger, u)
	if err != nil {
		return err
	}
	defer db.Close(ctx)

	if err := db.Dump(ctx, getDatabase(u), out, schemaOnly); err != nil {
		return fmt.Errorf("failed to create dump, got error: %w", err)
	}
	return nil
}
