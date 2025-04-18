package trino

import (
	"strings"

	parser "github.com/bytebase/trino-parser"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// StatementType is the type of the SQL statement.
type StatementType int

// Statement types
const (
	Unsupported StatementType = iota
	Select
	Explain
	Insert
	Update
	Delete
	Merge
	CreateTable
	CreateView
	AlterTable
	DropTable
	DropView
	CreateSchema
	DropSchema
	RenameTable
	CreateTableAsSelect
	Set
	Show
)

// getQueryType returns the type of the statement.
// Consistent with other database parsers for query type classification.
func getQueryType(node interface{}, allSystems bool) (base.QueryType, bool) {
	stmt, ok := node.(*parser.SingleStatementContext)
	if !ok {
		return base.QueryTypeUnknown, false
	}

	if stmt.Statement() == nil {
		return base.QueryTypeUnknown, false
	}

	// For Trino parser we need to use the parsed fields directly
	// The Statement() method returns an interface
	stmtText := stmt.GetText()

	// Check for DML statements
	if strings.HasPrefix(strings.ToUpper(stmtText), "INSERT") ||
		strings.HasPrefix(strings.ToUpper(stmtText), "UPDATE") ||
		strings.HasPrefix(strings.ToUpper(stmtText), "DELETE") ||
		strings.HasPrefix(strings.ToUpper(stmtText), "MERGE") {
		return base.DML, false
	}

	// Check for query (SELECT)
	if strings.HasPrefix(strings.ToUpper(stmtText), "SELECT") {
		if allSystems {
			return base.SelectInfoSchema, false
		}
		return base.Select, false
	}

	// Check for EXPLAIN
	if strings.HasPrefix(strings.ToUpper(stmtText), "EXPLAIN") {
		// Check if this is EXPLAIN ANALYZE
		if strings.Contains(strings.ToUpper(stmtText), "ANALYZE") {
			// If it's an EXPLAIN ANALYZE of a SELECT query
			if strings.Contains(strings.ToUpper(stmtText), "SELECT") {
				return base.Select, true
			}
			return base.QueryTypeUnknown, true
		}
		return base.Explain, false
	}

	// Check for DDL statements
	if strings.HasPrefix(strings.ToUpper(stmtText), "CREATE TABLE") ||
		strings.HasPrefix(strings.ToUpper(stmtText), "CREATE VIEW") ||
		strings.HasPrefix(strings.ToUpper(stmtText), "ALTER TABLE") ||
		strings.HasPrefix(strings.ToUpper(stmtText), "DROP TABLE") ||
		strings.HasPrefix(strings.ToUpper(stmtText), "DROP VIEW") ||
		strings.HasPrefix(strings.ToUpper(stmtText), "CREATE SCHEMA") ||
		strings.HasPrefix(strings.ToUpper(stmtText), "DROP SCHEMA") ||
		strings.HasPrefix(strings.ToUpper(stmtText), "RENAME") ||
		strings.Contains(strings.ToUpper(stmtText), "CREATE TABLE") && strings.Contains(strings.ToUpper(stmtText), "AS SELECT") {
		return base.DDL, false
	}

	// Check for other informational statements
	if strings.HasPrefix(strings.ToUpper(stmtText), "SHOW") {
		return base.SelectInfoSchema, false
	}

	// Check for session statements
	if strings.HasPrefix(strings.ToUpper(stmtText), "SET") {
		return base.Select, false // Treat SET as read-only, consistent with other parsers
	}

	return base.QueryTypeUnknown, false
}

// StatementType returns the detailed statement type (primarily for query span use).
func StatementType(tree interface{}) StatementType {
	stmt, ok := tree.(*parser.SingleStatementContext)
	if !ok {
		return Unsupported
	}

	if stmt.Statement() == nil {
		return Unsupported
	}

	stmtText := stmt.GetText()

	// Check for query (SELECT)
	if strings.HasPrefix(strings.ToUpper(stmtText), "SELECT") {
		return Select
	}

	// Check for EXPLAIN
	if strings.HasPrefix(strings.ToUpper(stmtText), "EXPLAIN") {
		return Explain
	}

	// Check for DML statements
	if strings.HasPrefix(strings.ToUpper(stmtText), "INSERT") {
		return Insert
	}
	if strings.HasPrefix(strings.ToUpper(stmtText), "UPDATE") {
		return Update
	}
	if strings.HasPrefix(strings.ToUpper(stmtText), "DELETE") {
		return Delete
	}
	if strings.HasPrefix(strings.ToUpper(stmtText), "MERGE") {
		return Merge
	}

	// Check for DDL statements
	if strings.HasPrefix(strings.ToUpper(stmtText), "CREATE TABLE") && !strings.Contains(strings.ToUpper(stmtText), "AS SELECT") {
		return CreateTable
	}
	if strings.HasPrefix(strings.ToUpper(stmtText), "CREATE VIEW") {
		return CreateView
	}
	if strings.HasPrefix(strings.ToUpper(stmtText), "ALTER TABLE") {
		return AlterTable
	}
	if strings.HasPrefix(strings.ToUpper(stmtText), "DROP TABLE") {
		return DropTable
	}
	if strings.HasPrefix(strings.ToUpper(stmtText), "DROP VIEW") {
		return DropView
	}
	if strings.HasPrefix(strings.ToUpper(stmtText), "CREATE SCHEMA") {
		return CreateSchema
	}
	if strings.HasPrefix(strings.ToUpper(stmtText), "DROP SCHEMA") {
		return DropSchema
	}
	if strings.HasPrefix(strings.ToUpper(stmtText), "ALTER") && strings.Contains(strings.ToUpper(stmtText), "RENAME") {
		return RenameTable
	}
	if strings.HasPrefix(strings.ToUpper(stmtText), "CREATE TABLE") && strings.Contains(strings.ToUpper(stmtText), "AS SELECT") {
		return CreateTableAsSelect
	}

	// Check for other statement types
	if strings.HasPrefix(strings.ToUpper(stmtText), "SET") {
		return Set
	}
	if strings.HasPrefix(strings.ToUpper(stmtText), "SHOW") {
		return Show
	}

	return Unsupported
}

// IsReadOnlyStatement returns whether the statement is read-only.
func IsReadOnlyStatement(tree interface{}) bool {
	queryType, _ := getQueryType(tree, false)
	return queryType == base.Select || 
		queryType == base.Explain || 
		queryType == base.SelectInfoSchema
}

// IsDataChangingStatement returns whether the statement changes data.
func IsDataChangingStatement(tree interface{}) bool {
	queryType, _ := getQueryType(tree, false)
	return queryType == base.DML
}

// IsSchemaChangingStatement returns whether the statement changes schema.
func IsSchemaChangingStatement(tree interface{}) bool {
	queryType, _ := getQueryType(tree, false)
	return queryType == base.DDL
}