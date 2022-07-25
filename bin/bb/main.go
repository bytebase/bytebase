// Bytebase cli command.
package main

import (
	"os"

	"github.com/bytebase/bytebase/bin/bb/cmd"

	// Register mysql driver.
	_ "github.com/bytebase/bytebase/plugin/db/mysql"
	// Register postgres driver.
	_ "github.com/bytebase/bytebase/plugin/db/pg"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
