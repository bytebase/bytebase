package main

import (
	"os"

	"github.com/bytebase/bytebase/bin/sql-service/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
