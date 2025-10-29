//nolint:revive
package common

import (
	"connectrpc.com/connect"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
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
		storepb.Engine_OCEANBASE,
		storepb.Engine_SNOWFLAKE,
		storepb.Engine_REDSHIFT,
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
		storepb.Engine_STARROCKS,
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
		storepb.Engine_MARIADB,
		storepb.Engine_REDSHIFT,
		storepb.Engine_STARROCKS,
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
		storepb.Engine_REDSHIFT,
		storepb.Engine_CASSANDRA,
		storepb.Engine_TRINO:
		return true
	case
		storepb.Engine_ENGINE_UNSPECIFIED,
		storepb.Engine_SQLITE,
		storepb.Engine_MONGODB,
		storepb.Engine_REDIS,
		storepb.Engine_CLICKHOUSE,
		storepb.Engine_SNOWFLAKE,
		storepb.Engine_STARROCKS,
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
		storepb.Engine_COCKROACHDB,
		storepb.Engine_MSSQL,
		storepb.Engine_ORACLE,
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
		storepb.Engine_OCEANBASE,
		storepb.Engine_SNOWFLAKE,
		storepb.Engine_MSSQL,
		storepb.Engine_DYNAMODB,
		storepb.Engine_COCKROACHDB,
		storepb.Engine_REDSHIFT:
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
		storepb.Engine_MARIADB,
		storepb.Engine_STARROCKS,
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
		storepb.Engine_MSSQL,
		storepb.Engine_MARIADB,
		storepb.Engine_REDSHIFT:
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
		storepb.Engine_STARROCKS,
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
		storepb.Engine_STARROCKS,
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
		storepb.Engine_STARROCKS,
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

// TransactionMode represents the transaction execution mode for a migration script.
type TransactionMode string

const (
	// TransactionModeOn wraps the script in a single transaction.
	TransactionModeOn TransactionMode = "on"
	// TransactionModeOff executes the script's statements sequentially in auto-commit mode.
	TransactionModeOff TransactionMode = "off"
	// TransactionModeUnspecified means no explicit mode was specified.
	TransactionModeUnspecified TransactionMode = ""
)

// IsolationLevel represents the transaction isolation level.
type IsolationLevel string

const (
	// IsolationLevelDefault uses the database's default isolation level.
	IsolationLevelDefault IsolationLevel = ""
	// IsolationLevelReadUncommitted allows dirty reads.
	IsolationLevelReadUncommitted IsolationLevel = "READ UNCOMMITTED"
	// IsolationLevelReadCommitted prevents dirty reads.
	IsolationLevelReadCommitted IsolationLevel = "READ COMMITTED"
	// IsolationLevelRepeatableRead prevents dirty reads and non-repeatable reads.
	IsolationLevelRepeatableRead IsolationLevel = "REPEATABLE READ"
	// IsolationLevelSerializable provides the highest isolation level.
	IsolationLevelSerializable IsolationLevel = "SERIALIZABLE"
)

// TransactionConfig represents the complete transaction configuration.
type TransactionConfig struct {
	Mode      TransactionMode
	Isolation IsolationLevel
}

// GetDefaultTransactionMode returns the default transaction mode.
// All engines default to "on" (transactional) for safety and backward compatibility.
// Users can explicitly set "-- txn-mode = off" when needed for engines with limited transactional DDL support.
func GetDefaultTransactionMode() TransactionMode {
	// All engines default to "on" for safety and backward compatibility
	return TransactionModeOn
}

func ConvertToParserEngine(e storepb.Engine) (storepb.Engine, error) {
	switch e {
	case storepb.Engine_POSTGRES:
		return storepb.Engine_POSTGRES, nil
	case storepb.Engine_MYSQL, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
		return storepb.Engine_MYSQL, nil
	case storepb.Engine_TIDB:
		return storepb.Engine_TIDB, nil
	case storepb.Engine_ORACLE:
		return storepb.Engine_ORACLE, nil
	case storepb.Engine_MSSQL:
		return storepb.Engine_MSSQL, nil
	case storepb.Engine_COCKROACHDB:
		return storepb.Engine_COCKROACHDB, nil
	default:
		return storepb.Engine_ENGINE_UNSPECIFIED, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid engine type %v", e))
	}
}
