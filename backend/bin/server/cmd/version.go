package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/bytebase/bytebase/backend/args"
)

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version of Bytebase",
	Run: func(_ *cobra.Command, _ []string) {
		fmt.Printf("Bytebase version: %s\n", args.Version)
		fmt.Printf("Git commit hash: %s\n", args.GitCommit)
	},
}
