package snowflake

import (
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterParseStatementsFunc(storepb.Engine_SNOWFLAKE, parseSnowflakeStatements)
}

// parseSnowflakeStatements is the ParseStatementsFunc for Snowflake.
// Returns []ParsedStatement with both text and AST populated.
//
// Each non-empty statement is parsed with the omni parser (the legacy ANTLR
// parser is gone). A parse failure on any statement fails the whole batch
// with a *base.SyntaxError — exactly the contract the legacy path had, so
// callers (e.g. the sheet manager) keep surfacing syntax errors unchanged.
// The omni side is parsed PER STATEMENT (stmt.Text) rather than from the
// whole input, so positions inside one statement stay relative to it and are
// offset by the statement's start position afterwards.
func parseSnowflakeStatements(statement string) ([]base.ParsedStatement, error) {
	// First split to get Statement with text and positions
	stmts, err := SplitSQL(statement)
	if err != nil {
		return nil, err
	}

	var result []base.ParsedStatement
	for _, stmt := range stmts {
		ps := base.ParsedStatement{
			Statement: stmt,
		}
		if !stmt.Empty {
			node, err := parseStatementNode(stmt.Text)
			if err != nil {
				return nil, convertParseError(err, stmt)
			}
			ps.AST = &AST{
				Node: node,
				Text: stmt.Text,
				// Keep the legacy ASTStartPosition shape: 1-based line of the
				// statement start, no column component.
				StartPosition: &storepb.Position{Line: int32(stmt.BaseLine()) + 1},
			}
		}
		result = append(result, ps)
	}

	return result, nil
}
