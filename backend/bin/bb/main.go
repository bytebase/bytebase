// Package main is the main package for bb CLI.
package main

import (
	"os"

	"github.com/bytebase/bytebase/backend/bin/bb/cmd"

	// Register mysql driver.
	_ "github.com/bytebase/bytebase/backend/plugin/db/mysql"
	// Register postgres driver.
	_ "github.com/bytebase/bytebase/backend/plugin/db/pg"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
