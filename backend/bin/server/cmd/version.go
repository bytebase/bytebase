package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// These should be set via go build -ldflags -X 'xxxx'.
var version = "development"
var gitcommit = "unknown"

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version of Bytebase",
	Run: func(_ *cobra.Command, _ []string) {
		fmt.Printf("Bytebase version: %s\n", version)
		fmt.Printf("Git commit hash: %s\n", gitcommit)
	},
}
