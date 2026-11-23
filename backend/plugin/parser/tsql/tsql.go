package tsql

import (
	mssqlparser "github.com/bytebase/omni/mssql/parser"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterParseStatementsFunc(storepb.Engine_MSSQL, parseTSQLStatements)
	base.RegisterGetStatementTypes(storepb.Engine_MSSQL, GetStatementTypes)
}

func parseTSQLStatements(statement string) ([]base.ParsedStatement, error) {
	return base.ParseNonEmptyMultiNodeStatements(statement, SplitSQL, func(stmt base.Statement) ([]base.AST, error) {
		parsed, err := ParseTSQL(stmt.Text)
		if err != nil {
			return nil, convertError(err, stmt)
		}
		asts := make([]base.AST, 0, len(parsed))
		for _, os := range parsed {
			if os.Empty() {
				continue
			}
			asts = append(asts, &AST{Node: os.AST, Text: stmt.Text, StartPosition: stmt.Start})
		}
		return asts, nil
	})
}

// convertError converts an omni parser error to a base.SyntaxError with proper line:column position.
func convertError(err error, stmt base.Statement) error {
	return base.ConvertSyntaxError(stmt, err, false,
		func(pe *mssqlparser.ParseError) (int, string) { return pe.Position, pe.Message },
		func(pe *mssqlparser.ParseError, pos *storepb.Position) string {
			return base.FormatLineSyntaxError(pos, pe.Message)
		},
	)
}
