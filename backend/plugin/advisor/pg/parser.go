package pg

import (
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/parser"
	"github.com/bytebase/bytebase/backend/plugin/parser/ast"
)

func parseStatement(statement string) ([]ast.Node, []advisor.Advice) {
	nodes, err := parser.Parse(parser.Postgres, parser.ParseContext{}, statement)
	if err != nil {
		if _, ok := err.(*parser.ConvertError); ok {
			return nil, []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    advisor.Internal,
					Title:   "Parser conversion error",
					Content: err.Error(),
					Line:    calculateErrorLine(statement),
				},
			}
		}
		return nil, []advisor.Advice{
			{
				Status:  advisor.Error,
				Code:    advisor.StatementSyntaxError,
				Title:   advisor.SyntaxErrorTitle,
				Content: err.Error(),
				Line:    calculateErrorLine(statement),
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

func calculateErrorLine(statement string) int {
	statementList, err := parser.SplitMultiSQL(parser.Postgres, statement)
	if err != nil {
		//nolint:nilerr
		return 1
	}

	for _, stmt := range statementList {
		if _, err := parser.Parse(parser.Postgres, parser.ParseContext{}, stmt.Text); err != nil {
			return stmt.LastLine
		}
	}

	return 0
}
