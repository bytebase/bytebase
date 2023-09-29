// Package parser provides the interfaces and libraries for SQL parser.
package parser

import (
	"strings"
)

// EngineType is the type of a parser engine.
type EngineType string

const (
	// Standard is the engine type for standard SQL.
	Standard EngineType = "STANDARD"
	// MySQL is the engine type for MYSQL.
	MySQL EngineType = "MYSQL"
	// TiDB is the engine type for TiDB.
	TiDB EngineType = "TIDB"
	// MariaDB is the engine type for MariaDB.
	MariaDB EngineType = "MARIADB"
	// Postgres is the engine type for POSTGRES.
	Postgres EngineType = "POSTGRES"
	// Oracle is the engine type for Oracle.
	Oracle EngineType = "ORACLE"
	// MSSQL is the engine type for MSSQL.
	MSSQL EngineType = "MSSQL"
	// Redshift is the engine type for redshift.
	Redshift EngineType = "REDSHIFT"
	// OceanBase is the engine type for OceanBase.
	OceanBase EngineType = "OCEANBASE"
	// Snowflake is the engine type for Snowflake.
	Snowflake EngineType = "SNOWFLAKE"
	//  SQLServer is the engine type for SQLServer.
	SQLServer EngineType = "SQLSERVER"
	// RisingWave is the engine type for RisingWave.
	RisingWave EngineType = "RISINGWAVE"
	// DeparseIndentString is the string for each indent level.
	DeparseIndentString = "    "
)

// ParseContext is the context for parsing.
type ParseContext struct {
}

// DeparseContext is the contxt for restoring.
type DeparseContext struct {
	// IndentLevel is indent level for current line.
	// The parser deparses statements with the indent level for pretty format.
	IndentLevel int
}

// WriteIndent is the helper function to write indent string.
func (ctx DeparseContext) WriteIndent(buf *strings.Builder, indent string) error {
	for i := 0; i < ctx.IndentLevel; i++ {
		if _, err := buf.WriteString(indent); err != nil {
			return err
		}
	}
	return nil
}
