// Package snowflake is the advisor for snowflake database.
package snowflake

import (
	"github.com/antlr4-go/antlr/v4"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	parser "github.com/bytebase/bytebase/backend/plugin/parser/sql"
)

func parseStatement(statement string) (antlr.Tree, []advisor.Advice) {
	tree, err := parser.ParseSnowSQL(statement + ";")
	if err != nil {
		if syntaxErr, ok := err.(*parser.SyntaxError); ok {
			return nil, []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    advisor.StatementSyntaxError,
					Title:   advisor.SyntaxErrorTitle,
					Content: syntaxErr.Message,
					Line:    syntaxErr.Line,
				},
			}
		}
		return nil, []advisor.Advice{
			{
				Status:  advisor.Warn,
				Code:    advisor.Internal,
				Title:   "Parse error",
				Content: err.Error(),
				Line:    1,
			},
		}
	}

	return tree, nil
}
