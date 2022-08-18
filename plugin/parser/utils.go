package parser

import (
	"github.com/bytebase/bytebase/plugin/parser/ast"
	"github.com/pkg/errors"
)

// SingleSQL is a separate SQL split from multi-SQL.
type SingleSQL struct {
	Text string
	Line int
}

// SplitMultiSQL splits statement into a slice of the single SQL.
func SplitMultiSQL(engineType EngineType, statement string) ([]SingleSQL, error) {
	switch engineType {
	case Postgres:
		t := newTokenizer(statement)
		return t.splitPostgreSQLMultiSQL()
	default:
		return nil, errors.Errorf("engine type is not supported: %s", engineType)
	}
}

// SetLineForCreateTableStmt sets the line for columns and table constraints in CREATE TABLE statements.
func SetLineForCreateTableStmt(engineType EngineType, node *ast.CreateTableStmt) error {
	switch engineType {
	case Postgres:
		t := newTokenizer(node.Text())
		return t.setLineForCreateTableStmt(node)
	default:
		return errors.Errorf("engine type is not supported: %s", engineType)
	}
}
