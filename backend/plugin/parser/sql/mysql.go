// Package parser is the parser for SQL statement.
package parser

import (
	"strings"

	tidbparser "github.com/pingcap/tidb/parser"
	"github.com/pingcap/tidb/parser/ast"
	"github.com/pkg/errors"
)

// ParseMySQL parses the given SQL statement and returns the AST.
func ParseMySQL(statement string, charset string, collation string) ([]ast.StmtNode, error) {
	p := tidbparser.New()

	// To support MySQL8 window function syntax.
	// See https://github.com/bytebase/bytebase/issues/175.
	p.EnableWindowFunc(true)
	list, err := SplitMultiSQLAndNormalize(MySQL, statement)
	if err != nil {
		return nil, err
	}

	var result []ast.StmtNode
	for _, sql := range list {
		if sql.Empty {
			continue
		}
		if IsTiDBUnsupportDDLStmt(sql.Text) {
			// skip the TiDB parser unsupported DDL statement
			continue
		}

		nodes, _, err := p.Parse(sql.Text, charset, collation)
		if err != nil {
			return nil, err
		}
		if len(nodes) == 0 {
			continue
		}
		if len(nodes) > 1 {
			return nil, errors.Errorf("get more than one sql after split: %s", sql.Text)
		}
		node := nodes[0]
		node.SetText(nil, strings.TrimSpace(node.Text()))
		node.SetOriginTextPosition(sql.LastLine)
		if n, ok := node.(*ast.CreateTableStmt); ok {
			if err := SetLineForMySQLCreateTableStmt(n); err != nil {
				return nil, err
			}
		}
		result = append(result, node)
	}

	return result, err
}
