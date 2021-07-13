// Bytebase cli command.
package main

import (
	"os"

	"github.com/bytebase/bytebase/bin/bb/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
