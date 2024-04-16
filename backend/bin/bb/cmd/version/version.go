package version

import (
	"fmt"

	"github.com/spf13/cobra"
)

// These should be set via go build -ldflags -X 'xxxx'.
var version = "development"
var goversion = "unknown"
var gitcommit = "unknown"
var buildtime = "unknown"
var builduser = "unknown"

func Cmd() *cobra.Command {
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Show bb version",
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Printf("bb version %s (Golang version: %s build on: %s by: %s commit: %s)\n", version, goversion, buildtime, builduser, gitcommit)
		},
	}
	return versionCmd
}
