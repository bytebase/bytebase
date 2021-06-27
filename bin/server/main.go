package main

import (
	"math/rand"
	"time"

	"github.com/bytebase/bytebase/bin/server/cmd"
)

func main() {
	rand.Seed(time.Now().UnixNano())
	cmd.Execute()
}
