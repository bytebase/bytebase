package parser

import (
	"io"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
	plsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/plsql"
	snowparser "github.com/bytebase/bytebase/backend/plugin/parser/snowflake"
	standardparser "github.com/bytebase/bytebase/backend/plugin/parser/standard"
	tidbparser "github.com/bytebase/bytebase/backend/plugin/parser/tidb"
	tsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/tsql"
)

// ExtractChangedResources extracts the changed resources from the SQL.
func ExtractChangedResources(engineType EngineType, currentDatabase string, currentSchema string, sql string) ([]base.SchemaResource, error) {
	switch engineType {
	case MySQL, MariaDB, OceanBase:
		return mysqlparser.ExtractMySQLChangedResources(currentDatabase, sql)
	case Oracle:
		return plsqlparser.ExtractOracleChangedResources(currentDatabase, currentSchema, sql)
	default:
		if currentDatabase == "" {
			return nil, errors.Errorf("database must be specified for engine type: %s", engineType)
		}
		return nil, errors.Errorf("engine type %q is not supported", engineType)
	}
}

// ExtractResourceList extracts the resource list from the SQL.
func ExtractResourceList(engineType EngineType, currentDatabase string, currentSchema string, sql string) ([]base.SchemaResource, error) {
	switch engineType {
	case TiDB:
		return tidbparser.ExtractTiDBResourceList(currentDatabase, sql)
	case MySQL, MariaDB, OceanBase:
		// The resource list for MySQL may contains table, view and temporary table.
		return mysqlparser.ExtractMySQLResourceList(currentDatabase, sql)
	case Oracle:
		// The resource list for Oracle may contains table, view and temporary table.
		return plsqlparser.ExtractOracleResourceList(currentDatabase, currentSchema, sql)
	case Postgres, RisingWave:
		// The resource list for Postgres may contains table, view and temporary table.
		return pgparser.ExtractPostgresResourceList(currentDatabase, currentSchema, sql)
	case Snowflake:
		return snowparser.ExtractSnowflakeNormalizeResourceListFromSelectStatement(currentDatabase, "PUBLIC", sql)
	case MSSQL:
		return tsqlparser.ExtractMSSQLNormalizedResourceListFromSelectStatement(currentDatabase, "dbo", sql)
	default:
		if currentDatabase == "" {
			return nil, errors.Errorf("database must be specified for engine type: %s", engineType)
		}
		return []base.SchemaResource{{Database: currentDatabase}}, nil
	}
}

// SplitMultiSQL splits statement into a slice of the single SQL.
func SplitMultiSQL(engineType EngineType, statement string) ([]base.SingleSQL, error) {
	switch engineType {
	case Oracle:
		return plsqlparser.SplitPLSQL(statement)
	case MSSQL:
		return tsqlparser.SplitSQL(statement)
	case Postgres, Redshift, RisingWave:
		return pgparser.SplitSQL(statement)
	case MySQL, MariaDB, OceanBase:
		return mysqlparser.SplitMySQL(statement)
	case TiDB:
		return tidbparser.SplitSQL(statement)
	default:
		return standardparser.SplitSQL(statement)
	}
}

// SplitMultiSQLStream splits statement stream into a slice of the single SQL.
func SplitMultiSQLStream(engineType EngineType, src io.Reader, f func(string) error) ([]base.SingleSQL, error) {
	switch engineType {
	case Oracle:
		return plsqlparser.SplitMultiSQLStream(src, f)
	case MSSQL:
		return tsqlparser.SplitMultiSQLStream(src, f)
	case Postgres, Redshift, RisingWave:
		return pgparser.SplitMultiSQLStream(src, f)
	case MySQL, MariaDB, OceanBase:
		return mysqlparser.SplitMultiSQLStream(src, f)
	case TiDB:
		return tidbparser.SplitMultiSQLStream(src, f)
	default:
		return nil, errors.Errorf("engine type is not supported: %s", engineType)
	}
}

// ExtractDatabaseList extracts all databases from statement.
func ExtractDatabaseList(engineType EngineType, statement string, fallbackNormalizedDatabaseName string) ([]string, error) {
	switch engineType {
	case MySQL, TiDB, MariaDB, OceanBase:
		return tidbparser.ExtractDatabaseList(statement)
	case Snowflake:
		return snowparser.ExtractSnowSQLNormalizedDatabaseList(statement, fallbackNormalizedDatabaseName)
	case MSSQL:
		return tsqlparser.ExtractMSSQLNormalizedDatabaseList(statement, fallbackNormalizedDatabaseName)
	default:
		return nil, errors.Errorf("engine type is not supported: %s", engineType)
	}
}
