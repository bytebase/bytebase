// Package oceanbase parses OceanBase's MySQL mode. OceanBase accepts distributed DDL
// MySQL rejects, so it carries its own omni parser and AST rather than MySQL's.
package oceanbase

import (
	omniparser "github.com/bytebase/omni/oceanbase/parser"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterParseStatementsFunc(storepb.Engine_OCEANBASE, parseStatements)
	base.RegisterGetStatementTypes(storepb.Engine_OCEANBASE, GetStatementTypes)
}

// parseStatements is the ParseStatementsFunc for OceanBase. Each returned statement
// carries both its text and its AST.
func parseStatements(statement string) ([]base.ParsedStatement, error) {
	return base.ParseNonEmptyMultiNodeStatements(statement, SplitSQL, func(stmt base.Statement) ([]base.AST, error) {
		list, err := omniparser.Parse(stmt.Text)
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

// convertError converts an omni parser error to a base.SyntaxError with proper line:column position.
func convertError(err error, stmt base.Statement) error {
	return base.ConvertSyntaxError(stmt, err, true,
		func(pe *omniparser.ParseError) (int, string) { return pe.Position, pe.Message },
		func(pe *omniparser.ParseError, pos *storepb.Position) string {
			return base.FormatLineColSyntaxError(pos, pe.Message, pe.RelatedText)
		},
	)
}
