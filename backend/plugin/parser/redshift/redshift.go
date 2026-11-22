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
	stmts, err := SplitSQL(statement)
	if err != nil {
		return nil, err
	}

	var result []base.ParsedStatement
	for _, stmt := range stmts {
		if stmt.Empty {
			result = append(result, base.ParsedStatement{Statement: stmt})
			continue
		}

		parsed, err := ParseRedshift(stmt.Text)
		if err != nil {
			return nil, convertError(err, stmt)
		}
		for _, parsedStmt := range parsed {
			if parsedStmt.Empty() {
				continue
			}
			result = append(result, base.ParsedStatement{
				Statement: stmt,
				AST: &AST{
					Node:          parsedStmt.AST,
					Text:          stmt.Text,
					StartPosition: stmt.Start,
				},
			})
		}
	}

	return result, nil
}
