package base

import storepb "github.com/bytebase/bytebase/proto/generated-go/store"

func BackupDatabaseNameOfEngine(e storepb.Engine) string {
	//exhaustive:enforce
	switch e {
	case
		storepb.Engine_MYSQL,
		storepb.Engine_TIDB,
		storepb.Engine_MSSQL,
		storepb.Engine_POSTGRES:
		return "bbdataarchive"
	case
		storepb.Engine_ORACLE:
		return "BBDATAARCHIVE"
	case
		storepb.Engine_ENGINE_UNSPECIFIED,
		storepb.Engine_SNOWFLAKE,
		storepb.Engine_CASSANDRA,
		storepb.Engine_SQLITE,
		storepb.Engine_MONGODB,
		storepb.Engine_REDIS,
		storepb.Engine_CLICKHOUSE,
		storepb.Engine_SPANNER,
		storepb.Engine_BIGQUERY,
		storepb.Engine_REDSHIFT,
		storepb.Engine_MARIADB,
		storepb.Engine_OCEANBASE,
		storepb.Engine_OCEANBASE_ORACLE,
		storepb.Engine_DM,
		storepb.Engine_STARROCKS,
		storepb.Engine_RISINGWAVE,
		storepb.Engine_HIVE,
		storepb.Engine_COCKROACHDB,
		storepb.Engine_DORIS,
		storepb.Engine_DYNAMODB,
		storepb.Engine_ELASTICSEARCH,
		storepb.Engine_DATABRICKS,
		storepb.Engine_COSMOSDB,
		storepb.Engine_TRINO:
		// Fallback to the default name for other engines.
		return "bbdataarchive"
	default:
		// Fallback to the default name for other engines.
		return "bbdataarchive"
	}
}
