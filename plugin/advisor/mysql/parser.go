package mysql

import (
	"github.com/bytebase/bytebase/plugin/advisor"
	"github.com/pingcap/tidb/parser"
	"github.com/pingcap/tidb/parser/ast"
)

// Wrapper for parser.New().
func newParser() *parser.Parser {
	p := parser.New()

	// To support MySQL8 window function syntax.
	// See https://github.com/bytebase/bytebase/issues/175.
	p.EnableWindowFunc(true)

	return p
}

func parseStatement(statement string, charset string, collation string) ([]ast.StmtNode, []advisor.Advice) {
	p := newParser()

	root, _, err := p.Parse(statement, charset, collation)
	if err != nil {
		return nil, []advisor.Advice{
			{
				Status:  advisor.Error,
				Title:   "Syntax error",
				Content: err.Error(),
			},
		}
	}
	return root, nil
}
