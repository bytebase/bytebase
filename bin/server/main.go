package main

import (
	"math/rand"
	"time"

	"github.com/bytebase/bytebase/bin/server/cmd"

	// Register clickhouse driver
	_ "github.com/bytebase/bytebase/plugin/db/clickhouse"
	// Register mysql driver
	_ "github.com/bytebase/bytebase/plugin/db/mysql"
	// Register postgres driver
	_ "github.com/bytebase/bytebase/plugin/db/pg"
	// Register snowflake driver
	_ "github.com/bytebase/bytebase/plugin/db/snowflake"

	// Register fake advisor
	_ "github.com/bytebase/bytebase/plugin/advisor/fake"
	// Register mysql advisor
	_ "github.com/bytebase/bytebase/plugin/advisor/mysql"
)

func main() {
	rand.Seed(time.Now().UnixNano())
	cmd.Execute()
}
