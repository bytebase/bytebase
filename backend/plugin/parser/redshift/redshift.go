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

		omniStmts, err := ParseRedshiftOmni(stmt.Text)
		if err != nil {
			return nil, convertOmniError(err, stmt)
		}
		for _, omniStmt := range omniStmts {
			if omniStmt.Empty() {
				continue
			}
			result = append(result, base.ParsedStatement{
				Statement: stmt,
				AST: &OmniAST{
					Node:          omniStmt.AST,
					Text:          stmt.Text,
					StartPosition: stmt.Start,
				},
			})
		}
	}

	return result, nil
}
