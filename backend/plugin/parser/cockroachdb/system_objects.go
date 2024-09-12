package cockroachdb

import (
	"fmt"
	"strings"
)

var (
	systemDatabases = map[string]bool{
		"postgres": true,
		"system":   true,
	}
	systemSchemas = map[string]bool{
		// https://www.cockroachlabs.com/docs/stable/system-catalogs
		"crdb_internal":      true,
		"information_schema": true,
		"pg_catalog":         true,
		"pg_extension":       true,
	}

	// SystemSchemaWhereClause is an optimization for getting less schema objects.
	SystemSchemaWhereClause = func() string {
		var schemas []string
		for schema := range systemSchemas {
			schemas = append(schemas, fmt.Sprintf("'%s'", schema))
		}
		return strings.Join(schemas, ",")
	}()
)

func IsSystemDatabase(database string) bool {
	_, ok := systemDatabases[database]
	return ok
}

func IsSystemSchema(schema string) bool {
	_, ok := systemSchemas[schema]
	return ok
}
