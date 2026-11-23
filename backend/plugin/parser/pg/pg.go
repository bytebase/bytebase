package pg

import (
	omniparser "github.com/bytebase/omni/pg/parser"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterParseStatementsFunc(storepb.Engine_POSTGRES, parsePgStatements)
	base.RegisterGetStatementTypes(storepb.Engine_POSTGRES, GetStatementTypesForRegistry)
}

// parsePgStatements is the ParseStatementsFunc for PostgreSQL.
// Returns []ParsedStatement with both text and AST populated.
func parsePgStatements(statement string) ([]base.ParsedStatement, error) {
	return base.ParseNonEmptyMultiNodeStatements(statement, SplitSQL, func(stmt base.Statement) ([]base.AST, error) {
		parsed, err := ParsePg(stmt.Text)
		if err != nil {
			return nil, convertError(err, statement, stmt)
		}
		asts := make([]base.AST, len(parsed))
		for i, os := range parsed {
			asts[i] = &AST{Node: os.AST, Text: stmt.Text, StartPosition: stmt.Start}
		}
		return asts, nil
	})
}

// convertError converts an omni parser error to a base.SyntaxError with proper line:column position.
func convertError(err error, _ string, stmt base.Statement) error {
	return base.ConvertSyntaxError(stmt, err, false,
		func(pe *omniparser.ParseError) (int, string) { return pe.Position, pe.Message },
		func(pe *omniparser.ParseError, pos *storepb.Position) string {
			return base.FormatLineSyntaxError(pos, pe.Message)
		},
	)
}
