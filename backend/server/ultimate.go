package server

import (
	// Drivers.
	_ "github.com/bytebase/bytebase/backend/plugin/db/drivers"

	// Parsers.
	_ "github.com/bytebase/bytebase/backend/plugin/parser/bigquery"
	_ "github.com/bytebase/bytebase/backend/plugin/parser/cassandra"
	_ "github.com/bytebase/bytebase/backend/plugin/parser/clickhouse"
	_ "github.com/bytebase/bytebase/backend/plugin/parser/cosmosdb"
	_ "github.com/bytebase/bytebase/backend/plugin/parser/databricks"
	_ "github.com/bytebase/bytebase/backend/plugin/parser/doris"
	_ "github.com/bytebase/bytebase/backend/plugin/parser/dynamodb"
	_ "github.com/bytebase/bytebase/backend/plugin/parser/elasticsearch"
	_ "github.com/bytebase/bytebase/backend/plugin/parser/hive"
	_ "github.com/bytebase/bytebase/backend/plugin/parser/mariadb"
	_ "github.com/bytebase/bytebase/backend/plugin/parser/mongodb"
	_ "github.com/bytebase/bytebase/backend/plugin/parser/mssql"
	_ "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	_ "github.com/bytebase/bytebase/backend/plugin/parser/oceanbase"
	_ "github.com/bytebase/bytebase/backend/plugin/parser/oracle"
	_ "github.com/bytebase/bytebase/backend/plugin/parser/pg"
	_ "github.com/bytebase/bytebase/backend/plugin/parser/redis"
	_ "github.com/bytebase/bytebase/backend/plugin/parser/redshift"
	_ "github.com/bytebase/bytebase/backend/plugin/parser/snowflake"
	_ "github.com/bytebase/bytebase/backend/plugin/parser/spanner"
	_ "github.com/bytebase/bytebase/backend/plugin/parser/starrocks"
	_ "github.com/bytebase/bytebase/backend/plugin/parser/tidb"
	_ "github.com/bytebase/bytebase/backend/plugin/parser/trino"

	// Advisors.
	_ "github.com/bytebase/bytebase/backend/plugin/advisor/bigquery"
	_ "github.com/bytebase/bytebase/backend/plugin/advisor/mariadb"
	_ "github.com/bytebase/bytebase/backend/plugin/advisor/mssql"
	_ "github.com/bytebase/bytebase/backend/plugin/advisor/mysql"
	_ "github.com/bytebase/bytebase/backend/plugin/advisor/oceanbase"
	_ "github.com/bytebase/bytebase/backend/plugin/advisor/oracle"
	_ "github.com/bytebase/bytebase/backend/plugin/advisor/pg"
	_ "github.com/bytebase/bytebase/backend/plugin/advisor/redshift"
	_ "github.com/bytebase/bytebase/backend/plugin/advisor/snowflake"
	_ "github.com/bytebase/bytebase/backend/plugin/advisor/tidb"

	// Schema designer.
	_ "github.com/bytebase/bytebase/backend/plugin/schema/clickhouse"
	_ "github.com/bytebase/bytebase/backend/plugin/schema/cockroachdb"
	_ "github.com/bytebase/bytebase/backend/plugin/schema/mariadb"
	_ "github.com/bytebase/bytebase/backend/plugin/schema/mssql"
	_ "github.com/bytebase/bytebase/backend/plugin/schema/mysql"
	_ "github.com/bytebase/bytebase/backend/plugin/schema/oceanbase"
	_ "github.com/bytebase/bytebase/backend/plugin/schema/oracle"
	_ "github.com/bytebase/bytebase/backend/plugin/schema/pg"
	_ "github.com/bytebase/bytebase/backend/plugin/schema/redshift"
	_ "github.com/bytebase/bytebase/backend/plugin/schema/tidb"
	_ "github.com/bytebase/bytebase/backend/plugin/schema/trino"

	// IM webhooks.
	_ "github.com/bytebase/bytebase/backend/plugin/webhook/dingtalk"
	_ "github.com/bytebase/bytebase/backend/plugin/webhook/discord"
	_ "github.com/bytebase/bytebase/backend/plugin/webhook/feishu"
	_ "github.com/bytebase/bytebase/backend/plugin/webhook/googlechat"
	_ "github.com/bytebase/bytebase/backend/plugin/webhook/lark"
	_ "github.com/bytebase/bytebase/backend/plugin/webhook/slack"
	_ "github.com/bytebase/bytebase/backend/plugin/webhook/teams"
	_ "github.com/bytebase/bytebase/backend/plugin/webhook/wecom"
)
