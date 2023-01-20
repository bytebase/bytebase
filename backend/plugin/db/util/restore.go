package util

import "fmt"

const (
	// MaxDatabaseNameLength is the allowed max database name length in MySQL.
	MaxDatabaseNameLength = 64
)

// TODO(dragonly): Check for PostgreSQL.

// GetPITRDatabaseName composes a pitr database name that we use as the target database for full backup recovery and binlog recovery.
// For example, GetPITRDatabaseName("dbfoo", 1653018005) -> "dbfoo_pitr_1653018005".
func GetPITRDatabaseName(database string, suffixTs int64) string {
	suffix := fmt.Sprintf("pitr_%d", suffixTs)
	return GetSafeName(database, suffix)
}

// GetPITROldDatabaseName composes a database name that we use as the target database for swapping out the original database.
// For example, GetPITROldDatabaseName("dbfoo", 1653018005) -> "dbfoo_pitr_1653018005_del".
func GetPITROldDatabaseName(database string, suffixTs int64) string {
	suffix := fmt.Sprintf("pitr_%d_del", suffixTs)
	return GetSafeName(database, suffix)
}

// GetSafeName trims the name according to max allowed database name length.
func GetSafeName(baseName, suffix string) string {
	name := fmt.Sprintf("%s_%s", baseName, suffix)
	if len(name) <= MaxDatabaseNameLength {
		return name
	}
	extraCharacters := len(name) - MaxDatabaseNameLength
	return fmt.Sprintf("%s_%s", baseName[0:len(baseName)-extraCharacters], suffix)
}
