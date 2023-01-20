// Package db provides the database utility for SQL advisor.
package db

import (
	"strings"

	"github.com/pkg/errors"
)

// Type is the type of a database.
// nolint
type Type string

const (
	// MySQL is the database type for MYSQL.
	MySQL Type = "MYSQL"
	// Postgres is the database type for POSTGRES.
	Postgres Type = "POSTGRES"
	// TiDB is the database type for TiDB.
	TiDB Type = "TIDB"
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
	}

	return "", errors.Errorf("unsupported db type %s for advisor", dbType)
}
