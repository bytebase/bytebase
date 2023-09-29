package parser

import (
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
	standardparser "github.com/bytebase/bytebase/backend/plugin/parser/standard"
)

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
