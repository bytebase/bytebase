//go:build release

package server

import (
	// Register snowflake driver.
	_ "github.com/bytebase/bytebase/backend/plugin/db/snowflake"
	// Register mongodb driver.
	_ "github.com/bytebase/bytebase/backend/plugin/db/mongodb"
	// Register spanner driver.
	_ "github.com/bytebase/bytebase/backend/plugin/db/spanner"
	// Register redis driver.
	_ "github.com/bytebase/bytebase/backend/plugin/db/redis"
	// Register oracle driver.
	_ "github.com/bytebase/bytebase/backend/plugin/db/oracle"
	// Register redshift driver.
	_ "github.com/bytebase/bytebase/backend/plugin/db/redshift"
	// Register clickhouse driver.
	_ "github.com/bytebase/bytebase/backend/plugin/db/clickhouse"
	// Register dm driver.
	_ "github.com/bytebase/bytebase/backend/plugin/db/dm"
	// Register risingwave driver.
	_ "github.com/bytebase/bytebase/backend/plugin/db/risingwave"
	// Register mssql driver.
	_ "github.com/bytebase/bytebase/backend/plugin/db/mssql"

	// Register oracle advisor.
	_ "github.com/bytebase/bytebase/backend/plugin/advisor/oracle"
	// Register snowflake advisor.
	_ "github.com/bytebase/bytebase/backend/plugin/advisor/snowflake"
	// Register mssql advisor.
	_ "github.com/bytebase/bytebase/backend/plugin/advisor/mssql"

	// Register oracle differ driver.
	_ "github.com/bytebase/bytebase/backend/plugin/parser/sql/differ/plsql"
)
