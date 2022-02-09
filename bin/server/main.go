package main

import (
	"os"

	"github.com/bytebase/bytebase/bin/server/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
