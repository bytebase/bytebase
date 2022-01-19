package main

import (
	"math/rand"
	"time"

	"github.com/bytebase/bytebase/bin/server/cmd"

	// Import sqlite3 driver.
	_ "github.com/mattn/go-sqlite3"

	// Register clickhouse driver.
	_ "github.com/bytebase/bytebase/plugin/db/clickhouse"
	// Register mysql driver.
	_ "github.com/bytebase/bytebase/plugin/db/mysql"
	// Register postgres driver.
	_ "github.com/bytebase/bytebase/plugin/db/pg"
	_ "github.com/lib/pq"

	// Register snowflake driver.
	_ "github.com/bytebase/bytebase/plugin/db/snowflake"
	// Register sqlite driver.
	_ "github.com/bytebase/bytebase/plugin/db/sqlite"

	// Register pingcap parser driver.
	_ "github.com/pingcap/tidb/types/parser_driver"
	// Register fake advisor.
	_ "github.com/bytebase/bytebase/plugin/advisor/fake"
	// Register mysql advisor.
	_ "github.com/bytebase/bytebase/plugin/advisor/mysql"
)

func main() {
	rand.Seed(time.Now().UnixNano())
	cmd.Execute()
}
