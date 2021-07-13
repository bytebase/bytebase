// cmd is the command surface of Bytebase bb tool provided by bytebase.com.
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version of bb",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Bytebase bb tool v0.0 (bytebase.com)")
	},
}
