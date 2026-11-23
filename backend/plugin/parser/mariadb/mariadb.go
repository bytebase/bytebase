package mariadb

import (
	mariadbomniparser "github.com/bytebase/omni/mariadb/parser"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterParseStatementsFunc(storepb.Engine_MARIADB, parseStatements)
	base.RegisterGetStatementTypes(storepb.Engine_MARIADB, GetStatementTypes)
}

// parseStatements is the ParseStatementsFunc for MariaDB.
// Returns []ParsedStatement with both text and AST populated.
func parseStatements(statement string) ([]base.ParsedStatement, error) {
	return base.ParseNonEmptyMultiNodeStatements(statement, SplitSQL, func(stmt base.Statement) ([]base.AST, error) {
		list, err := ParseMariaDB(stmt.Text)
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

// convertError converts an omni MariaDB parser error to a base.SyntaxError with proper line:column position.
func convertError(err error, stmt base.Statement) error {
	return base.ConvertSyntaxError(stmt, err, true,
		func(pe *mariadbomniparser.ParseError) (int, string) { return pe.Position, pe.Message },
		func(pe *mariadbomniparser.ParseError, pos *storepb.Position) string {
			return base.FormatLineColSyntaxError(pos, pe.Message, pe.RelatedText)
		},
	)
}
