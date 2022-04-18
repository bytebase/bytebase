package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// These should be set via go build -ldflags -X 'xxxx'
var version = "development"
var goversion = "unknown"
var gitcommit = "unknown"
var buildtime = "unknown"
var builduser = "unknown"

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version of Bytebase",
	Run: func(_ *cobra.Command, _ []string) {
		fmt.Printf("Bytebase version: %s\n", version)
		fmt.Printf("Golang version: %s\n", goversion)
		fmt.Printf("Git commit hash: %s\n", gitcommit)
		fmt.Printf("Built on: %s\n", buildtime)
		fmt.Printf("Built by: %s\n", builduser)
	},
}
