package pg

import (
	"github.com/bytebase/bytebase/plugin/advisor"
	"github.com/bytebase/bytebase/plugin/parser"
	"github.com/bytebase/bytebase/plugin/parser/ast"
)

func parseStatement(statement string) ([]ast.Node, []advisor.Advice) {
	nodes, err := parser.Parse(parser.Postgres, parser.Context{}, statement)
	if err != nil {
		if _, ok := err.(*parser.ConvertError); ok {
			return nil, []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    advisor.Internal,
					Title:   "Parser conversion error",
					Content: err.Error(),
				},
			}
		}
		return nil, []advisor.Advice{
			{
				Status:  advisor.Error,
				Code:    advisor.StatementSyntaxError,
				Title:   advisor.SyntaxErrorTitle,
				Content: err.Error(),
			},
		}
	}
	var res []ast.Node
	for _, node := range nodes {
		if node != nil {
			res = append(res, node)
		}
	}
	return res, nil
}
