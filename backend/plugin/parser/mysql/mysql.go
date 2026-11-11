package mysql

import (
	"errors"
	"fmt"

	mysqlomniparser "github.com/bytebase/omni/mysql/parser"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterParseStatementsFunc(storepb.Engine_MYSQL, parseMySQLStatements)
	base.RegisterGetStatementTypes(storepb.Engine_MYSQL, GetStatementTypes)
}

// parseMySQLStatements is the ParseStatementsFunc for MySQL.
// Returns []ParsedStatement with both text and AST populated.
func parseMySQLStatements(statement string) ([]base.ParsedStatement, error) {
	return parseStatements(statement)
}

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

		list, err := ParseMySQL(stmt.Text)
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

// convertError converts an omni MySQL parser error to a base.SyntaxError with proper line:column position.
func convertError(err error, stmt base.Statement) error {
	var parseErr *mysqlomniparser.ParseError
	if !errors.As(err, &parseErr) {
		return err
	}
	message, position, relatedText := parseErr.Message, parseErr.Position, parseErr.RelatedText

	pos := ByteOffsetToRunePosition(stmt.Text, position)

	// Offset by the statement's 1-based start; its column applies only to the statement's first line.
	if stmt.Start != nil {
		if pos.Line == 1 {
			pos.Column += stmt.Start.Column - 1
		}
		pos.Line += stmt.Start.Line - 1
	}

	msg := fmt.Sprintf("Syntax error at line %d:%d: %s", pos.Line, pos.Column, message)
	if relatedText != "" {
		msg += "\nrelated text: " + relatedText
	}

	return &base.SyntaxError{
		Position:   pos,
		Message:    msg,
		RawMessage: message,
	}
}
