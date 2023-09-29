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
func ExtractChangedResources(engineType storepb.Engine, currentDatabase string, currentSchema string, sql string) ([]base.SchemaResource, error) {
	switch engineType {
	case storepb.Engine_MYSQL, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
		// currentSchema is empty.
		return mysqlparser.ExtractChangedResources(currentDatabase, currentSchema, sql)
	case storepb.Engine_ORACLE, storepb.Engine_DM:
		return plsqlparser.ExtractChangedResources(currentDatabase, currentSchema, sql)
	default:
		return nil, errors.Errorf("engine type is not supported: %s", engineType)
	}
}

// ExtractResourceList extracts the resource list from the SQL.
func ExtractResourceList(engineType storepb.Engine, currentDatabase string, currentSchema string, sql string) ([]base.SchemaResource, error) {
	switch engineType {
	case storepb.Engine_MYSQL, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
		// The resource list for MySQL may contains table, view and temporary table.
		return mysqlparser.ExtractMySQLResourceList(currentDatabase, sql)
	case storepb.Engine_TIDB:
		return tidbparser.ExtractTiDBResourceList(currentDatabase, sql)
	case storepb.Engine_POSTGRES, storepb.Engine_REDSHIFT, storepb.Engine_RISINGWAVE:
		// The resource list for Postgres may contains table, view and temporary table.
		return pgparser.ExtractPostgresResourceList(currentDatabase, currentSchema, sql)
	case storepb.Engine_ORACLE, storepb.Engine_DM:
		// The resource list for Oracle may contains table, view and temporary table.
		return plsqlparser.ExtractOracleResourceList(currentDatabase, currentSchema, sql)
	case storepb.Engine_SNOWFLAKE:
		return snowparser.ExtractSnowflakeNormalizeResourceListFromSelectStatement(currentDatabase, "PUBLIC", sql)
	case storepb.Engine_MSSQL:
		return tsqlparser.ExtractMSSQLNormalizedResourceListFromSelectStatement(currentDatabase, "dbo", sql)
	default:
		if currentDatabase == "" {
			return nil, errors.Errorf("database must be specified for engine type: %s", engineType)
		}
		return []base.SchemaResource{{Database: currentDatabase}}, nil
	}
}

// SplitMultiSQL splits statement into a slice of the single SQL.
func SplitMultiSQL(engineType storepb.Engine, statement string) ([]base.SingleSQL, error) {
	switch engineType {
	case storepb.Engine_MYSQL, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
		return mysqlparser.SplitMySQL(statement)
	case storepb.Engine_TIDB:
		return tidbparser.SplitSQL(statement)
	case storepb.Engine_POSTGRES, storepb.Engine_REDSHIFT, storepb.Engine_RISINGWAVE:
		return pgparser.SplitSQL(statement)
	case storepb.Engine_ORACLE, storepb.Engine_DM:
		return plsqlparser.SplitPLSQL(statement)
	case storepb.Engine_MSSQL:
		return tsqlparser.SplitSQL(statement)
	default:
		return standardparser.SplitSQL(statement)
	}
}

// SplitMultiSQLStream splits statement stream into a slice of the single SQL.
func SplitMultiSQLStream(engineType storepb.Engine, src io.Reader, f func(string) error) ([]base.SingleSQL, error) {
	switch engineType {
	case storepb.Engine_MYSQL, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
		return mysqlparser.SplitMultiSQLStream(src, f)
	case storepb.Engine_TIDB:
		return tidbparser.SplitMultiSQLStream(src, f)
	case storepb.Engine_POSTGRES, storepb.Engine_REDSHIFT, storepb.Engine_RISINGWAVE:
		return pgparser.SplitMultiSQLStream(src, f)
	case storepb.Engine_ORACLE, storepb.Engine_DM:
		return plsqlparser.SplitMultiSQLStream(src, f)
	case storepb.Engine_MSSQL:
		return tsqlparser.SplitMultiSQLStream(src, f)
	default:
		return nil, errors.Errorf("engine type is not supported: %s", engineType)
	}
}

// ExtractDatabaseList extracts all databases from statement.
func ExtractDatabaseList(engineType storepb.Engine, statement string, fallbackNormalizedDatabaseName string) ([]string, error) {
	switch engineType {
	case storepb.Engine_MYSQL, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
		// TODO(d): use mysql parser.
		return tidbparser.ExtractDatabaseList(statement)
	case storepb.Engine_TIDB:
		return tidbparser.ExtractDatabaseList(statement)
	case storepb.Engine_SNOWFLAKE:
		return snowparser.ExtractSnowSQLNormalizedDatabaseList(statement, fallbackNormalizedDatabaseName)
	case storepb.Engine_MSSQL:
		return tsqlparser.ExtractMSSQLNormalizedDatabaseList(statement, fallbackNormalizedDatabaseName)
	default:
		return nil, errors.Errorf("engine type is not supported: %s", engineType)
	}
}

func ExtractSensitiveField(engine storepb.Engine, statement string, currentDatabase string, schemaInfo *db.SensitiveSchemaInfo) ([]db.SensitiveField, error) {
	if schemaInfo == nil {
		return nil, nil
	}

	f := base.FieldMaskers[engine]
	return f(statement, currentDatabase, schemaInfo)
}

// ValidateSQLForEditor validates the SQL statement for editor.
// We support the following SQLs:
// 1. EXPLAIN statement, except EXPLAIN ANALYZE
// 2. SELECT statement
// We also support CTE with SELECT statements, but not with DML statements.
func ValidateSQLForEditor(engine storepb.Engine, statement string) bool {
	f := base.QueryValidators[engine]
	return f(statement)
}
