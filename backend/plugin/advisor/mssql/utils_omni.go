package mssql

import (
	"strings"

	"github.com/bytebase/omni/mssql/ast"
)

// normalizeTableRef joins non-empty parts of a TableRef into a lowercase dot-separated string.
// SQL Server identifiers are case-insensitive by default, so we lowercase for consistent
// map keys and equality checks.
func normalizeTableRef(ref *ast.TableRef, fallbackDB, fallbackSchema string) string {
	if ref == nil {
		return ""
	}
	db, schema, obj := ref.Database, ref.Schema, ref.Object
	if db == "" {
		db = fallbackDB
	}
	if schema == "" {
		schema = fallbackSchema
	}
	var parts []string
	if db != "" {
		parts = append(parts, strings.ToLower(db))
	}
	if schema != "" {
		parts = append(parts, strings.ToLower(schema))
	}
	if obj != "" {
		parts = append(parts, strings.ToLower(obj))
	}
	return strings.Join(parts, ".")
}
