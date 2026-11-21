package mariadb

import (
	"errors"
	"fmt"

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
	// Split once to get Statement with text and positions
	stmts, err := SplitSQL(statement)
	if err != nil {
		return nil, err
	}

	var result []base.ParsedStatement
	for _, stmt := range stmts {
		if stmt.Empty {
			continue
		}

		list, err := ParseMariaDB(stmt.Text)
		if err != nil {
			return nil, convertError(err, stmt)
		}

		if list == nil || len(list.Items) == 0 {
			continue
		}

		for _, node := range list.Items {
			result = append(result, base.ParsedStatement{
				Statement: stmt,
				AST: &AST{
					Node:          node,
					Text:          stmt.Text,
					StartPosition: stmt.Start,
				},
			})
		}
	}

	return result, nil
}

// convertError converts an omni MariaDB parser error to a base.SyntaxError with proper line:column position.
func convertError(err error, stmt base.Statement) error {
	var parseErr *mariadbomniparser.ParseError
	if !errors.As(err, &parseErr) {
		return err
	}

	pos := base.ShiftPosition(ByteOffsetToRunePosition(stmt.Text, parseErr.Position), stmt.Start)

	msg := fmt.Sprintf("Syntax error at line %d:%d: %s", pos.Line, pos.Column, parseErr.Message)
	if parseErr.RelatedText != "" {
		msg += "\nrelated text: " + parseErr.RelatedText
	}

	return &base.SyntaxError{
		Position:   pos,
		Message:    msg,
		RawMessage: parseErr.Message,
	}
}
