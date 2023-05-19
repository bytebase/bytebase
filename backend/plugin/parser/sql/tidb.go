package parser

import (
	tidbparser "github.com/pingcap/tidb/parser"
	"github.com/pingcap/tidb/parser/ast"
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
