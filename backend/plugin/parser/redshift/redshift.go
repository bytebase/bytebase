package redshift

import (
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterParseStatementsFunc(storepb.Engine_REDSHIFT, parseRedshiftStatements)
	base.RegisterGetStatementTypes(storepb.Engine_REDSHIFT, GetStatementTypes)
}

// parseRedshiftStatements is the ParseStatementsFunc for Redshift.
// Returns []ParsedStatement with both text and AST populated.
func parseRedshiftStatements(statement string) ([]base.ParsedStatement, error) {
	return base.ParseMultiNodeStatements(statement, SplitSQL, func(stmt base.Statement) ([]base.AST, error) {
		if stmt.Empty {
			return nil, nil
		}
		parsed, err := ParseRedshift(stmt.Text)
		if err != nil {
			return nil, convertError(err, stmt)
		}
		asts := make([]base.AST, 0, len(parsed))
		for _, parsedStmt := range parsed {
			if parsedStmt.Empty() {
				continue
			}
			asts = append(asts, &AST{
				Node:          parsedStmt.AST,
				Text:          stmt.Text,
				StartPosition: stmt.Start,
			})
		}
		return asts, nil
	})
}
