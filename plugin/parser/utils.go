package parser

import (
	"io"
	"strings"

	tidbast "github.com/pingcap/tidb/parser/ast"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/plugin/parser/ast"
)

// SingleSQL is a separate SQL split from multi-SQL.
type SingleSQL struct {
	Text     string
	LastLine int
}

// SplitMultiSQL splits statement into a slice of the single SQL.
func SplitMultiSQL(engineType EngineType, statement string) ([]SingleSQL, error) {
	switch engineType {
	case Postgres:
		t := newTokenizer(statement)
		return t.splitPostgreSQLMultiSQL()
	case MySQL, TiDB:
		t := newTokenizer(statement)
		return t.splitMySQLMultiSQL()
	default:
		return nil, errors.Errorf("engine type is not supported: %s", engineType)
	}
}

// SplitMultiSQLStream splits statement stream into a slice of the single SQL.
func SplitMultiSQLStream(engineType EngineType, src io.Reader, f func(string) error) ([]SingleSQL, error) {
	switch engineType {
	case Postgres:
		t := newStreamTokenizer(src, f)
		return t.splitPostgreSQLMultiSQL()
	case MySQL, TiDB:
		t := newStreamTokenizer(src, f)
		return t.splitMySQLMultiSQL()
	default:
		return nil, errors.Errorf("engine type is not supported: %s", engineType)
	}
}

// SetLineForCreateTableStmt sets the line for columns and table constraints in CREATE TABLE statements.
func SetLineForCreateTableStmt(engineType EngineType, node *ast.CreateTableStmt) error {
	switch engineType {
	case Postgres:
		t := newTokenizer(node.Text())
		firstLine := node.LastLine() - strings.Count(node.Text(), "\n")
		return t.setLineForPGCreateTableStmt(node, firstLine)
	default:
		return errors.Errorf("engine type is not supported: %s", engineType)
	}
}

// SetLineForMySQLCreateTableStmt sets the line for columns and table constraints in MySQL CREATE TABLE statments.
// This is a temporary function. Because we do not convert tidb AST to our AST. So we have to implement this.
// TODO(rebelice): remove it.
func SetLineForMySQLCreateTableStmt(node *tidbast.CreateTableStmt) error {
	// exclude CREATE TABLE ... AS and CREATE TABLE ... LIKE statement.
	if len(node.Cols) == 0 {
		return nil
	}
	firstLine := node.OriginTextPosition() - strings.Count(node.Text(), "\n")
	return newTokenizer(node.Text()).setLineForMySQLCreateTableStmt(node, firstLine)
}
