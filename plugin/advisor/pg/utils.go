package pg

import (
	"fmt"

	"github.com/bytebase/bytebase/plugin/parser/ast"
)

const (
	// PostgreSQLPublicSchema is the string for PostgreSQL public schema
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

type columnMap map[columnName]bool

func getTableNameWithSchema(table *ast.TableDef) string {
	schema := table.Schema
	if schema == "" {
		schema = "public"
	}
	return fmt.Sprintf("%s.%s", schema, table.Name)
}
