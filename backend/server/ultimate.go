//go:build !minimal

package server

import (
	// Drivers.
	_ "github.com/bytebase/bytebase/backend/plugin/db/clickhouse"
	_ "github.com/bytebase/bytebase/backend/plugin/db/dm"
	_ "github.com/bytebase/bytebase/backend/plugin/db/mongodb"
	_ "github.com/bytebase/bytebase/backend/plugin/db/mssql"
	_ "github.com/bytebase/bytebase/backend/plugin/db/oracle"
	_ "github.com/bytebase/bytebase/backend/plugin/db/redis"
	_ "github.com/bytebase/bytebase/backend/plugin/db/redshift"
	_ "github.com/bytebase/bytebase/backend/plugin/db/risingwave"
	_ "github.com/bytebase/bytebase/backend/plugin/db/snowflake"
	_ "github.com/bytebase/bytebase/backend/plugin/db/spanner"

	// Parsers.
	_ "github.com/bytebase/bytebase/backend/plugin/parser/plsql"
	_ "github.com/bytebase/bytebase/backend/plugin/parser/snowflake"
	_ "github.com/bytebase/bytebase/backend/plugin/parser/standard"
	_ "github.com/bytebase/bytebase/backend/plugin/parser/tsql"

	// Advisors.
	_ "github.com/bytebase/bytebase/backend/plugin/advisor/mssql"
	_ "github.com/bytebase/bytebase/backend/plugin/advisor/oracle"
	_ "github.com/bytebase/bytebase/backend/plugin/advisor/snowflake"
)
