package pg

import (
	"errors"
	"fmt"
	"strings"

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
	stmts, err := SplitSQL(statement)
	if err != nil {
		return nil, err
	}

	var result []base.ParsedStatement
	for _, stmt := range stmts {
		if stmt.Empty {
			continue
		}

		omniStmts, omniErr := ParsePg(stmt.Text)
		if omniErr != nil {
			return nil, convertOmniError(omniErr, statement, stmt)
		}

		for _, os := range omniStmts {
			result = append(result, base.ParsedStatement{
				Statement: stmt,
				AST: &OmniAST{
					Node:          os.AST,
					Text:          stmt.Text,
					StartPosition: stmt.Start,
				},
			})
		}
	}
	return result, nil
}

// convertOmniError converts an omni parser error to a base.SyntaxError with proper line:column position.
func convertOmniError(err error, _ string, stmt base.Statement) error {
	var parseErr *omniparser.ParseError
	if !errors.As(err, &parseErr) {
		return err
	}

	// Convert byte offset within stmt.Text to line:column.
	pos := ByteOffsetToRunePosition(stmt.Text, parseErr.Position)

	// Adjust line by the statement's base line (stmt.Start.Line is 1-based).
	if stmt.Start != nil {
		pos.Line += stmt.Start.Line - 1
	}

	return &base.SyntaxError{
		Position:   pos,
		Message:    fmt.Sprintf("Syntax error at line %d: %s", pos.Line, parseErr.Message),
		RawMessage: parseErr.Message,
	}
}

// normalizePostgreSQLQuotedIdentifier removes surrounding double quotes and unescapes internal ones.
func normalizePostgreSQLQuotedIdentifier(s string) string {
	if len(s) < 2 {
		return s
	}
	return strings.ReplaceAll(s[1:len(s)-1], `""`, `"`)
}
