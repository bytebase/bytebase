//go:build !minidemo

package server

import (
	// Drivers.
	_ "github.com/bytebase/bytebase/backend/plugin/db/bigquery"
	_ "github.com/bytebase/bytebase/backend/plugin/db/clickhouse"
	_ "github.com/bytebase/bytebase/backend/plugin/db/cockroachdb"
	_ "github.com/bytebase/bytebase/backend/plugin/db/databricks"
	_ "github.com/bytebase/bytebase/backend/plugin/db/dm"
	_ "github.com/bytebase/bytebase/backend/plugin/db/dynamodb"
	_ "github.com/bytebase/bytebase/backend/plugin/db/elasticsearch"
	_ "github.com/bytebase/bytebase/backend/plugin/db/hive"
	_ "github.com/bytebase/bytebase/backend/plugin/db/mongodb"
	_ "github.com/bytebase/bytebase/backend/plugin/db/mssql"
	_ "github.com/bytebase/bytebase/backend/plugin/db/mysql"
	_ "github.com/bytebase/bytebase/backend/plugin/db/oracle"
	_ "github.com/bytebase/bytebase/backend/plugin/db/redis"
	_ "github.com/bytebase/bytebase/backend/plugin/db/redshift"
	_ "github.com/bytebase/bytebase/backend/plugin/db/risingwave"
	_ "github.com/bytebase/bytebase/backend/plugin/db/snowflake"
	_ "github.com/bytebase/bytebase/backend/plugin/db/spanner"
	_ "github.com/bytebase/bytebase/backend/plugin/db/sqlite"
	_ "github.com/bytebase/bytebase/backend/plugin/db/starrocks"
	_ "github.com/bytebase/bytebase/backend/plugin/db/tidb"

	// Parsers.
	_ "github.com/bytebase/bytebase/backend/plugin/parser/bigquery"
	_ "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	_ "github.com/bytebase/bytebase/backend/plugin/parser/partiql"
	_ "github.com/bytebase/bytebase/backend/plugin/parser/plsql"
	_ "github.com/bytebase/bytebase/backend/plugin/parser/redis"
	_ "github.com/bytebase/bytebase/backend/plugin/parser/snowflake"
	_ "github.com/bytebase/bytebase/backend/plugin/parser/standard"
	_ "github.com/bytebase/bytebase/backend/plugin/parser/tidb"
	_ "github.com/bytebase/bytebase/backend/plugin/parser/tsql"

	// Advisors.
	_ "github.com/bytebase/bytebase/backend/plugin/advisor/mssql"
	_ "github.com/bytebase/bytebase/backend/plugin/advisor/mysql"
	_ "github.com/bytebase/bytebase/backend/plugin/advisor/oceanbase"
	_ "github.com/bytebase/bytebase/backend/plugin/advisor/oracle"
	_ "github.com/bytebase/bytebase/backend/plugin/advisor/snowflake"
	_ "github.com/bytebase/bytebase/backend/plugin/advisor/tidb"

	// Schema designer.
	_ "github.com/bytebase/bytebase/backend/plugin/schema/clickhouse"
	_ "github.com/bytebase/bytebase/backend/plugin/schema/mysql"
	_ "github.com/bytebase/bytebase/backend/plugin/schema/oracle"
	_ "github.com/bytebase/bytebase/backend/plugin/schema/pg"
	_ "github.com/bytebase/bytebase/backend/plugin/schema/redshift"
	_ "github.com/bytebase/bytebase/backend/plugin/schema/tidb"

	// Transformers.
	_ "github.com/bytebase/bytebase/backend/plugin/parser/sql/transform/mysql"

	// IM webhooks.
	_ "github.com/bytebase/bytebase/backend/plugin/webhook/dingtalk"
	_ "github.com/bytebase/bytebase/backend/plugin/webhook/feishu"
	_ "github.com/bytebase/bytebase/backend/plugin/webhook/slack"
	_ "github.com/bytebase/bytebase/backend/plugin/webhook/wecom"
)
