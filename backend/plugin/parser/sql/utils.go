package parser

import (
	"io"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
	plsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/plsql"
	snowparser "github.com/bytebase/bytebase/backend/plugin/parser/snowflake"
	standardparser "github.com/bytebase/bytebase/backend/plugin/parser/standard"
	tidbparser "github.com/bytebase/bytebase/backend/plugin/parser/tidb"
	tsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/tsql"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// ExtractChangedResources extracts the changed resources from the SQL.
func ExtractChangedResources(engineType EngineType, currentDatabase string, currentSchema string, sql string) ([]base.SchemaResource, error) {
	switch engineType {
	case MySQL, MariaDB, OceanBase:
		// currentSchema is empty.
		return mysqlparser.ExtractChangedResources(currentDatabase, currentSchema, sql)
	case Oracle:
		return plsqlparser.ExtractChangedResources(currentDatabase, currentSchema, sql)
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

func ExtractSensitiveField(dbType storepb.Engine, statement string, currentDatabase string, schemaInfo *db.SensitiveSchemaInfo) ([]db.SensitiveField, error) {
	if schemaInfo == nil {
		return nil, nil
	}
	switch dbType {
	case storepb.Engine_MYSQL, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
		return mysqlparser.GetMaskedFields(statement, currentDatabase, schemaInfo)
	case storepb.Engine_TIDB:
		return tidbparser.GetMaskedFields(statement, currentDatabase, schemaInfo)
	case storepb.Engine_POSTGRES, storepb.Engine_REDSHIFT, storepb.Engine_RISINGWAVE:
		return pgparser.GetMaskedFields(statement, "", schemaInfo)
	case storepb.Engine_ORACLE, storepb.Engine_DM:
		return plsqlparser.GetMaskedFields(statement, currentDatabase, schemaInfo)
	case storepb.Engine_SNOWFLAKE:
		return snowparser.GetMaskedFields(statement, currentDatabase, schemaInfo)
	case storepb.Engine_MSSQL:
		return tsqlparser.GetMaskedFields(statement, currentDatabase, schemaInfo)
	default:
		return nil, nil
	}
}

// ValidateSQLForEditor validates the SQL statement for editor.
// We support the following SQLs:
// 1. EXPLAIN statement, except EXPLAIN ANALYZE
// 2. SELECT statement
//
// We also support CTE with SELECT statements, but not with DML statements.
func ValidateSQLForEditor(engine EngineType, statement string) bool {
	switch engine {
	case Postgres, Redshift, RisingWave:
		return pgparser.ValidateSQLForEditor(statement)
	case MySQL, TiDB, MariaDB, OceanBase:
		return mysqlparser.ValidateSQLForEditor(statement)
	default:
		return standardparser.ValidateSQLForEditor(statement)
	}
}
