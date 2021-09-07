package main

import (
	"math/rand"
	"time"

	"github.com/bytebase/bytebase/bin/server/cmd"

	// Register mysql driver
	_ "github.com/bytebase/bytebase/plugin/db/mysql"
)

func main() {
	rand.Seed(time.Now().UnixNano())
	cmd.Execute()
}
