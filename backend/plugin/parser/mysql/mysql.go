package mysql

import (
	mysqlomniparser "github.com/bytebase/omni/mysql/parser"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterParseStatementsFunc(storepb.Engine_MYSQL, parseStatements)
	base.RegisterGetStatementTypes(storepb.Engine_MYSQL, GetStatementTypes)
}

func parseStatements(statement string) ([]base.ParsedStatement, error) {
	return base.ParseNonEmptyMultiNodeStatements(statement, SplitSQL, func(stmt base.Statement) ([]base.AST, error) {
		list, err := mysqlomniparser.Parse(stmt.Text)
		if err != nil {
			return nil, convertError(err, stmt)
		}
		if list == nil || len(list.Items) == 0 {
			return nil, nil
		}
		asts := make([]base.AST, len(list.Items))
		for i, node := range list.Items {
			asts[i] = &AST{Node: node, Text: stmt.Text, StartPosition: stmt.Start}
		}
		return asts, nil
	})
}

// convertError converts an omni MySQL parser error to a base.SyntaxError with proper line:column position.
func convertError(err error, stmt base.Statement) error {
	return base.ConvertSyntaxError(stmt, err, true,
		func(pe *mysqlomniparser.ParseError) (int, string) { return pe.Position, pe.Message },
		func(pe *mysqlomniparser.ParseError, pos *storepb.Position) string {
			return base.FormatLineColSyntaxError(pos, pe.Message, pe.RelatedText)
		},
	)
}
