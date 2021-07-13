// cmd is the command surface of Bytebase bb tool provided by bytebase.com.
package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "bb",
	Short: "A database management tool provided by bytebase.com",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Usage()
	},
}

// Execute is the execute command for root command.
func Execute() error {
	return rootCmd.Execute()
}
