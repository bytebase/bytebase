package mssql

import (
	"strings"

	"github.com/bytebase/omni/mssql/ast"
)

// normalizeTableRef joins non-empty parts of a TableRef into a dot-separated string.
// Case is preserved to match NormalizeTSQLTableName behavior.
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
		parts = append(parts, db)
	}
	if schema != "" {
		parts = append(parts, schema)
	}
	if obj != "" {
		parts = append(parts, obj)
	}
	return strings.Join(parts, ".")
}
