package tsql

import "strings"

var (
	// https://learn.microsoft.com/en-us/sql/relational-databases/databases/system-databases?view=sql-server-ver16
	systemDatabases = map[string]string{
		"MASTER":   "master",
		"MSDB":     "msdb",
		"MODEL":    "model",
		"RESOURCE": "Resource",
		"TEMPDB":   "tempdb",
	}

	// SELECT * FROM INFORMATION_SCHEMA.SCHEMATA;.
	systemSchemas = map[string]string{
		"INFORMATION_SCHEMA": "INFORMATION_SCHEMA",
		"SYS":                "sys",
		"DB_OWNER":           "db_owner",
		"DB_ACCESSADMIN":     "db_accessadmin",
		"DB_SECURITYADMIN":   "db_securityadmin",
		"DB_DDLADMIN":        "db_ddladmin",
		"DB_BACKUPOPERATOR":  "db_backupoperator",
		"DB_DATAREADER":      "db_datareader",
		"DB_DATAWRITER":      "db_datawriter",
		"DB_DENYDATAREADER":  "db_denydatareader",
		"DB_DENYDATAWRITER":  "db_denydatawriter",
	}
)

func IsSystemDatabase(databaseName string, caseSensitive bool) bool {
	if caseSensitive {
		origin, ok := systemDatabases[strings.ToUpper(databaseName)]
		if !ok {
			return false
		}
		return origin == databaseName
	}
	_, in := systemDatabases[strings.ToUpper(databaseName)]
	return in
}

func IsSystemSchema(schemaName string, caseSensitive bool) bool {
	if caseSensitive {
		origin, ok := systemSchemas[strings.ToUpper(schemaName)]
		if !ok {
			return false
		}
		return origin == schemaName
	}
	_, in := systemSchemas[strings.ToUpper(schemaName)]
	return in
}
