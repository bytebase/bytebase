// Package db provides the database utility for SQL advisor.
package db

import (
	"strings"

	"github.com/pkg/errors"
)

// Type is the type of a database.
// nolint
type Type string

// TODO(d): use a centric database type.
const (
	// MySQL is the database type for MYSQL.
	MySQL Type = "MYSQL"
	// Postgres is the database type for POSTGRES.
	Postgres Type = "POSTGRES"
	// TiDB is the database type for TiDB.
	TiDB Type = "TIDB"
	// MariaDB is the database type for MariaDB.
	MariaDB Type = "MARIADB"
	// Oracle is the database type for Oracle.
	Oracle Type = "ORACLE"
	// OceanBase is the database type for OceanBase.
	OceanBase Type = "OCEANBASE"
	// Snowflake is the database type for Snowflake.
	Snowflake Type = "SNOWFLAKE"
	// MSSQL is the database type for SQL Server.
	MSSQL Type = "MSSQL"
	// DM is the database type for DM.
	DM Type = "DM"
)

// ConvertToAdvisorDBType will convert db type into advisor db type.
func ConvertToAdvisorDBType(dbType string) (Type, error) {
	switch strings.ToUpper(dbType) {
	case string(MySQL):
		return MySQL, nil
	case string(Postgres):
		return Postgres, nil
	case string(TiDB):
		return TiDB, nil
	case string(Oracle):
		return Oracle, nil
	case string(OceanBase):
		return OceanBase, nil
	case string(Snowflake):
		return Snowflake, nil
	case string(MSSQL):
		return MSSQL, nil
	case string(DM):
		return DM, nil
	}

	return "", errors.Errorf("unsupported db type %s for advisor", dbType)
}
