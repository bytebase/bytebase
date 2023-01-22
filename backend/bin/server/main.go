// Package main is the main binary for Bytebase service.
package main

import (
	"os"

	"github.com/bytebase/bytebase/backend/bin/server/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
