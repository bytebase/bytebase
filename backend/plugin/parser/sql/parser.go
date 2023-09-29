// Package parser provides the interfaces and libraries for SQL parser.
package parser

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
