package pg

import (
	"github.com/auxten/postgresql-parser/pkg/sql/parser"
	"github.com/bytebase/bytebase/plugin/advisor"
)

func parseStatement(statement string) (parser.Statements, []advisor.Advice) {
	stmts, err := parser.Parse(statement)
	if err != nil {
		return nil, []advisor.Advice{
			{
				Status:  advisor.Error,
				Code:    advisor.DbStatementSyntaxError,
				Title:   advisor.SyntaxErrorTitle,
				Content: err.Error(),
			},
		}
	}
	return stmts, nil
}
