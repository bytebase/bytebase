// Package main is the main binary for SQL review service.
package main

import (
	"os"

	"github.com/bytebase/bytebase/backend/bin/sql-service/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
