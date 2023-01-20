// Package pg implements the SQL advisor rules for PostgreSQL.
package pg

import (
	"fmt"

	"github.com/bytebase/bytebase/backend/plugin/parser/ast"
)

const (
	// PostgreSQLPublicSchema is the string for PostgreSQL public schema.
	PostgreSQLPublicSchema = "public"
)

type columnName struct {
	schema string
	table  string
	column string
}

func (c columnName) normalizeTableName() string {
	schema := c.schema
	if schema == "" {
		schema = "public"
	}
	return fmt.Sprintf(`"%s"."%s"`, schema, c.table)
}

type columnMap map[columnName]int

func normalizeSchemaName(name string) string {
	if name != "" {
		return name
	}
	return "public"
}

func normalizeTableName(table *ast.TableDef, defaultSchema string) string {
	schema := table.Schema
	if schema == "" && defaultSchema != "" {
		schema = defaultSchema
	}

	if schema == "" {
		return fmt.Sprintf("%q", table.Name)
	}
	return fmt.Sprintf("%q.%q", schema, table.Name)
}
