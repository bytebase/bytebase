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

func newVersionCmd() *cobra.Command {
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print the version of bb",
		Run: func(cmd *cobra.Command, _ []string) {
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "bb version: %s\n", version)
			fmt.Fprintf(out, "Golang version: %s\n", goversion)
			fmt.Fprintf(out, "Git commit hash: %s\n", gitcommit)
			fmt.Fprintf(out, "Built on: %s\n", buildtime)
			fmt.Fprintf(out, "Built by: %s\n", builduser)
		},
	}
	return versionCmd
}
