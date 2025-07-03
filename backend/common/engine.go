package common

import (
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func EngineSupportSQLReview(engine storepb.Engine) bool {
	//exhaustive:enforce
	switch engine {
	case
		storepb.Engine_POSTGRES,
		storepb.Engine_MYSQL,
		storepb.Engine_TIDB,
		storepb.Engine_MARIADB,
		storepb.Engine_ORACLE,
		storepb.Engine_OCEANBASE_ORACLE,
		storepb.Engine_OCEANBASE,
		storepb.Engine_SNOWFLAKE,
		storepb.Engine_DM,
		storepb.Engine_MSSQL:
		return true
	case
		storepb.Engine_ENGINE_UNSPECIFIED,
		storepb.Engine_CASSANDRA,
		storepb.Engine_SQLITE,
		storepb.Engine_MONGODB,
		storepb.Engine_REDIS,
		storepb.Engine_CLICKHOUSE,
		storepb.Engine_SPANNER,
		storepb.Engine_BIGQUERY,
		storepb.Engine_REDSHIFT,
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
		return false
	default:
		return false
	}
}

func EngineSupportQueryNewACL(engine storepb.Engine) bool {
	//exhaustive:enforce
	switch engine {
	case
		storepb.Engine_MYSQL,
		storepb.Engine_POSTGRES,
		storepb.Engine_ORACLE,
		storepb.Engine_MSSQL,
		storepb.Engine_TIDB,
		storepb.Engine_SNOWFLAKE,
		storepb.Engine_SPANNER,
		storepb.Engine_BIGQUERY:
		return true
	case
		storepb.Engine_ENGINE_UNSPECIFIED,
		storepb.Engine_CASSANDRA,
		storepb.Engine_SQLITE,
		storepb.Engine_MONGODB,
		storepb.Engine_REDIS,
		storepb.Engine_CLICKHOUSE,
		storepb.Engine_OCEANBASE,
		storepb.Engine_OCEANBASE_ORACLE,
		storepb.Engine_MARIADB,
		storepb.Engine_DM,
		storepb.Engine_REDSHIFT,
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
		return false
	default:
		return false
	}
}

func EngineSupportMasking(e storepb.Engine) bool {
	//exhaustive:enforce
	switch e {
	case
		storepb.Engine_MYSQL,
		storepb.Engine_POSTGRES,
		storepb.Engine_ORACLE,
		storepb.Engine_MSSQL,
		storepb.Engine_MARIADB,
		storepb.Engine_OCEANBASE,
		storepb.Engine_TIDB,
		storepb.Engine_BIGQUERY,
		storepb.Engine_SPANNER,
		storepb.Engine_TRINO:
		return true
	case
		storepb.Engine_ENGINE_UNSPECIFIED,
		storepb.Engine_CASSANDRA,
		storepb.Engine_SQLITE,
		storepb.Engine_MONGODB,
		storepb.Engine_REDIS,
		storepb.Engine_CLICKHOUSE,
		storepb.Engine_OCEANBASE_ORACLE,
		storepb.Engine_DM,
		storepb.Engine_SNOWFLAKE,
		storepb.Engine_REDSHIFT,
		storepb.Engine_STARROCKS,
		storepb.Engine_RISINGWAVE,
		storepb.Engine_HIVE,
		storepb.Engine_COCKROACHDB,
		storepb.Engine_DORIS,
		storepb.Engine_DYNAMODB,
		storepb.Engine_ELASTICSEARCH,
		storepb.Engine_DATABRICKS,
		storepb.Engine_COSMOSDB:
		return false
	default:
		return false
	}
}

func EngineSupportAutoComplete(e storepb.Engine) bool {
	//exhaustive:enforce
	switch e {
	case
		storepb.Engine_MYSQL,
		storepb.Engine_TIDB,
		storepb.Engine_MARIADB,
		storepb.Engine_OCEANBASE,
		storepb.Engine_CLICKHOUSE,
		storepb.Engine_STARROCKS,
		storepb.Engine_DORIS,
		storepb.Engine_POSTGRES,
		storepb.Engine_REDSHIFT,
		storepb.Engine_RISINGWAVE,
		storepb.Engine_COCKROACHDB,
		storepb.Engine_MSSQL,
		storepb.Engine_ORACLE,
		storepb.Engine_DM,
		storepb.Engine_OCEANBASE_ORACLE,
		storepb.Engine_SNOWFLAKE,
		storepb.Engine_DYNAMODB,
		storepb.Engine_TRINO:
		return true
	case
		storepb.Engine_ENGINE_UNSPECIFIED,
		storepb.Engine_CASSANDRA,
		storepb.Engine_SQLITE,
		storepb.Engine_MONGODB,
		storepb.Engine_REDIS,
		storepb.Engine_SPANNER,
		storepb.Engine_BIGQUERY,
		storepb.Engine_HIVE,
		storepb.Engine_ELASTICSEARCH,
		storepb.Engine_DATABRICKS,
		storepb.Engine_COSMOSDB:
		return false
	default:
		return false
	}
}

func EngineSupportStatementAdvise(e storepb.Engine) bool {
	//exhaustive:enforce
	switch e {
	case
		storepb.Engine_MYSQL,
		storepb.Engine_TIDB,
		storepb.Engine_POSTGRES,
		storepb.Engine_ORACLE,
		storepb.Engine_OCEANBASE_ORACLE,
		storepb.Engine_OCEANBASE,
		storepb.Engine_SNOWFLAKE,
		storepb.Engine_MSSQL,
		storepb.Engine_DYNAMODB,
		storepb.Engine_COCKROACHDB,
		storepb.Engine_REDSHIFT:
		return true
	case
		storepb.Engine_ENGINE_UNSPECIFIED,
		storepb.Engine_DM,
		storepb.Engine_CASSANDRA,
		storepb.Engine_SQLITE,
		storepb.Engine_MONGODB,
		storepb.Engine_REDIS,
		storepb.Engine_CLICKHOUSE,
		storepb.Engine_SPANNER,
		storepb.Engine_BIGQUERY,
		storepb.Engine_MARIADB,
		storepb.Engine_STARROCKS,
		storepb.Engine_RISINGWAVE,
		storepb.Engine_HIVE,
		storepb.Engine_DORIS,
		storepb.Engine_ELASTICSEARCH,
		storepb.Engine_DATABRICKS,
		storepb.Engine_COSMOSDB,
		storepb.Engine_TRINO:
		return false
	default:
		return false
	}
}

func EngineSupportStatementReport(e storepb.Engine) bool {
	//exhaustive:enforce
	switch e {
	case
		storepb.Engine_POSTGRES,
		storepb.Engine_MYSQL,
		storepb.Engine_TIDB,
		storepb.Engine_OCEANBASE,
		storepb.Engine_ORACLE,
		storepb.Engine_OCEANBASE_ORACLE,
		storepb.Engine_MSSQL,
		storepb.Engine_MARIADB,
		storepb.Engine_REDSHIFT:
		return true
	case
		storepb.Engine_ENGINE_UNSPECIFIED,
		storepb.Engine_DM,
		storepb.Engine_SNOWFLAKE,
		storepb.Engine_CASSANDRA,
		storepb.Engine_SQLITE,
		storepb.Engine_MONGODB,
		storepb.Engine_REDIS,
		storepb.Engine_CLICKHOUSE,
		storepb.Engine_SPANNER,
		storepb.Engine_BIGQUERY,
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
		return false
	default:
		return false
	}
}

func EngineSupportPriorBackup(e storepb.Engine) bool {
	//exhaustive:enforce
	switch e {
	case
		storepb.Engine_MYSQL,
		storepb.Engine_TIDB,
		storepb.Engine_MSSQL,
		storepb.Engine_ORACLE,
		storepb.Engine_POSTGRES:
		return true
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
		return false
	default:
		return false
	}
}

func EngineSupportCreateDatabase(e storepb.Engine) bool {
	//exhaustive:enforce
	switch e {
	case
		storepb.Engine_SQLITE,
		storepb.Engine_MYSQL,
		storepb.Engine_POSTGRES,
		storepb.Engine_MSSQL,
		storepb.Engine_SNOWFLAKE,
		storepb.Engine_CLICKHOUSE,
		storepb.Engine_MONGODB,
		storepb.Engine_TIDB,
		storepb.Engine_OCEANBASE,
		storepb.Engine_REDSHIFT,
		storepb.Engine_MARIADB,
		storepb.Engine_STARROCKS,
		storepb.Engine_RISINGWAVE,
		storepb.Engine_HIVE,
		storepb.Engine_COCKROACHDB,
		storepb.Engine_DORIS:
		return true
	case
		storepb.Engine_ENGINE_UNSPECIFIED,
		storepb.Engine_CASSANDRA,
		storepb.Engine_SPANNER,
		storepb.Engine_BIGQUERY,
		storepb.Engine_DYNAMODB,
		storepb.Engine_REDIS,
		storepb.Engine_ORACLE,
		storepb.Engine_DM,
		storepb.Engine_OCEANBASE_ORACLE,
		storepb.Engine_ELASTICSEARCH,
		storepb.Engine_DATABRICKS,
		storepb.Engine_COSMOSDB,
		storepb.Engine_TRINO:
		return false
	default:
		return false
	}
}

func EngineSupportQuerySpanPlainField(e storepb.Engine) bool {
	//exhaustive:enforce
	switch e {
	case
		storepb.Engine_MYSQL,
		storepb.Engine_OCEANBASE,
		storepb.Engine_MARIADB:
		return true
	case
		storepb.Engine_ENGINE_UNSPECIFIED,
		storepb.Engine_CASSANDRA,
		storepb.Engine_SPANNER,
		storepb.Engine_BIGQUERY,
		storepb.Engine_DYNAMODB,
		storepb.Engine_REDIS,
		storepb.Engine_ORACLE,
		storepb.Engine_DM,
		storepb.Engine_OCEANBASE_ORACLE,
		storepb.Engine_ELASTICSEARCH,
		storepb.Engine_DATABRICKS,
		storepb.Engine_COSMOSDB,
		storepb.Engine_TRINO,
		storepb.Engine_SQLITE,
		storepb.Engine_POSTGRES,
		storepb.Engine_MSSQL,
		storepb.Engine_SNOWFLAKE,
		storepb.Engine_CLICKHOUSE,
		storepb.Engine_MONGODB,
		storepb.Engine_TIDB,
		storepb.Engine_REDSHIFT,
		storepb.Engine_STARROCKS,
		storepb.Engine_RISINGWAVE,
		storepb.Engine_HIVE,
		storepb.Engine_COCKROACHDB,
		storepb.Engine_DORIS:
		return false
	default:
		return false
	}
}

func EngineSupportSyntaxCheck(e storepb.Engine) bool {
	//exhaustive:enforce
	switch e {
	case
		storepb.Engine_TIDB,
		storepb.Engine_MYSQL,
		storepb.Engine_MARIADB,
		storepb.Engine_OCEANBASE,
		storepb.Engine_POSTGRES,
		storepb.Engine_REDSHIFT,
		storepb.Engine_ORACLE,
		storepb.Engine_OCEANBASE_ORACLE,
		storepb.Engine_SNOWFLAKE,
		storepb.Engine_MSSQL,
		storepb.Engine_DYNAMODB,
		storepb.Engine_COCKROACHDB:
		return true
	case
		storepb.Engine_ENGINE_UNSPECIFIED,
		storepb.Engine_CASSANDRA,
		storepb.Engine_SQLITE,
		storepb.Engine_MONGODB,
		storepb.Engine_REDIS,
		storepb.Engine_CLICKHOUSE,
		storepb.Engine_SPANNER,
		storepb.Engine_BIGQUERY,
		storepb.Engine_STARROCKS,
		storepb.Engine_RISINGWAVE,
		storepb.Engine_HIVE,
		storepb.Engine_DORIS,
		storepb.Engine_ELASTICSEARCH,
		storepb.Engine_DATABRICKS,
		storepb.Engine_COSMOSDB,
		storepb.Engine_TRINO,
		storepb.Engine_DM:
		return false
	default:
		return false
	}
}

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

// EngineDBSchemaReadyToMigrate returns true if the engine needs column default migration.
// This is used by both the migrator and the sync process to determine if a database
// needs the column default migration.
// When an engine's sync.go is updated to write to the Default field with proper
// normalization (like schema qualification), we move it to the false case.
func EngineDBSchemaReadyToMigrate(e storepb.Engine) bool {
	//exhaustive:enforce
	switch e {
	case
		storepb.Engine_POSTGRES, storepb.Engine_MYSQL, storepb.Engine_MSSQL:
		return true
	case

		storepb.Engine_TIDB,
		storepb.Engine_MARIADB,
		storepb.Engine_ORACLE,
		storepb.Engine_OCEANBASE_ORACLE,
		storepb.Engine_OCEANBASE,
		storepb.Engine_SNOWFLAKE,
		storepb.Engine_DM,
		storepb.Engine_CLICKHOUSE,
		storepb.Engine_COCKROACHDB,
		storepb.Engine_SPANNER,
		storepb.Engine_BIGQUERY,
		storepb.Engine_REDSHIFT,
		storepb.Engine_RISINGWAVE,
		storepb.Engine_STARROCKS,
		storepb.Engine_DORIS:
		// These engines still need migration as their sync.go hasn't been updated yet.
		return false
	case
		storepb.Engine_ENGINE_UNSPECIFIED,
		storepb.Engine_CASSANDRA,
		storepb.Engine_SQLITE,
		storepb.Engine_MONGODB,
		storepb.Engine_REDIS,
		storepb.Engine_HIVE,
		storepb.Engine_DYNAMODB,
		storepb.Engine_ELASTICSEARCH,
		storepb.Engine_DATABRICKS,
		storepb.Engine_COSMOSDB,
		storepb.Engine_TRINO:
		// These engines don't have traditional column defaults or are NoSQL databases.
		return true
	default:
		return true
	}
}
