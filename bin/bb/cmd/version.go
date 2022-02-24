package cmd

import (
	"fmt"
	"io"

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
	Short: "Print the version of bb",
	Run: func(cmd *cobra.Command, args []string) {
		out := cmd.OutOrStdout()
		io.WriteString(out, fmt.Sprintf("bb version: %s\n", version))
		io.WriteString(out, fmt.Sprintf("bb version: %s\n", version))
		io.WriteString(out, fmt.Sprintf("Golang version: %s\n", goversion))
		io.WriteString(out, fmt.Sprintf("Git commit hash: %s\n", gitcommit))
		io.WriteString(out, fmt.Sprintf("Built on: %s\n", buildtime))
		io.WriteString(out, fmt.Sprintf("Built by: %s\n", builduser))
	},
}
