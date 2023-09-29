package tidb

import (
	"strings"

	tidbparser "github.com/pingcap/tidb/parser"
	"github.com/pingcap/tidb/parser/ast"

	"github.com/bytebase/bytebase/backend/plugin/parser/tokenizer"
)

// ParseTiDB parses the given SQL statement and returns the AST.
func ParseTiDB(sql string, charset string, collation string) ([]ast.StmtNode, error) {
	p := tidbparser.New()

	// To support MySQL8 window function syntax.
	// See https://github.com/bytebase/bytebase/issues/175.
	p.EnableWindowFunc(true)

	nodes, _, err := p.Parse(sql, charset, collation)
	return nodes, err
}

// SetLineForMySQLCreateTableStmt sets the line for columns and table constraints in MySQL CREATE TABLE statments.
// This is a temporary function. Because we do not convert tidb AST to our AST. So we have to implement this.
// TODO(rebelice): remove it.
func SetLineForMySQLCreateTableStmt(node *ast.CreateTableStmt) error {
	// exclude CREATE TABLE ... AS and CREATE TABLE ... LIKE statement.
	if len(node.Cols) == 0 {
		return nil
	}
	firstLine := node.OriginTextPosition() - strings.Count(node.Text(), "\n")
	return tokenizer.NewTokenizer(node.Text()).SetLineForMySQLCreateTableStmt(node, firstLine)
}
