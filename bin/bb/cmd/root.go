// Package cmd is the command surface of Bytebase bb tool provided by bytebase.com.
package cmd

import (
	"github.com/bytebase/bytebase/common/log"
	"github.com/spf13/cobra"
)

// NewRootCmd creates the root command.
func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "bb",
		Short: "A database management tool provided by bytebase.com",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Usage()
		},
	}

	rootCmd.AddCommand(newDumpCmd(), newRestoreCmd(), newVersionCmd(), newMigrateCmd())

	return rootCmd
}

// Execute is the execute command for root command.
func Execute() (err error) {
	defer log.Sync()
	return NewRootCmd().Execute()
}
