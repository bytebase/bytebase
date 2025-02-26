package cockroachdb

import (
	"strings"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func init() {
	base.RegisterSchemaDiffFunc(storepb.Engine_COCKROACHDB, SchemaDiff)
}

// SchemaDiff computes the schema differences between two SQL statements for CockroachDB.
func SchemaDiff(_ base.DiffContext, oldStmt, newStmt string) (string, error) {
	oldStmt = sanitizeCockroachSyntax(oldStmt)
	newStmt = sanitizeCockroachSyntax(newStmt)
	// Reuse the PostgreSQL schema diff implementation.
	return pgparser.SchemaDiff(base.DiffContext{}, oldStmt, newStmt)
}

// sanitizeCockroachSyntax sanitizes the CockroachDB-specific syntax before reusing the PostgreSQL schema diff implementation.
// CockroachDB has some syntax that is not supported by PostgreSQL, but we want to reuse the PostgreSQL schema diff.
// Currently, we handle this with simple strings.ReplaceAll calls.
func sanitizeCockroachSyntax(sql string) string {
	sql = removeVisible(sql)
	sql = removePrimaryKeyASC(sql)
	return sql
}

// removeVisible removes "NOT VISIBLE" and "VISIBLE" clauses.
// Postgres does not support "VISIBLE" clause for certain statements.
func removeVisible(sql string) string {
	sql = remove(sql, " NOT VISIBLE")
	sql = remove(sql, " VISIBLE")
	return sql
}

// removePrimaryKeyASC removes "ASC" from PRIMARY KEY definitions.
// PRIMARY KEY (rowid ASC) -> PRIMARY KEY (rowid).
func removePrimaryKeyASC(sql string) string {
	return remove(sql, " ASC")
}

func remove(sql, keyword string) string {
	return strings.ReplaceAll(sql, keyword, "")
}
