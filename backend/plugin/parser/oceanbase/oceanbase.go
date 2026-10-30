// Package oceanbase parses OceanBase's MySQL mode. OceanBase accepts distributed DDL
// MySQL rejects, so it carries its own omni parser and AST rather than MySQL's.
package oceanbase

import (
	"errors"
	"fmt"

	omniparser "github.com/bytebase/omni/oceanbase/parser"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterParseStatementsFunc(storepb.Engine_OCEANBASE, parseStatements)
	base.RegisterGetStatementTypes(storepb.Engine_OCEANBASE, GetStatementTypes)
}

// parseStatements is the ParseStatementsFunc for OceanBase. Each returned statement
// carries both its text and its AST.
func parseStatements(statement string) ([]base.ParsedStatement, error) {
	stmts, err := SplitSQL(statement)
	if err != nil {
		return nil, err
	}

	var result []base.ParsedStatement
	for _, stmt := range stmts {
		if stmt.Empty {
			continue
		}

		list, omniErr := ParseOceanBase(stmt.Text)
		if omniErr != nil {
			return nil, convertOmniError(omniErr, stmt)
		}
		if list == nil || len(list.Items) == 0 {
			continue
		}

		for _, node := range list.Items {
			result = append(result, base.ParsedStatement{
				Statement: stmt,
				AST: &OmniAST{
					Node:          node,
					Text:          stmt.Text,
					StartPosition: stmt.Start,
				},
			})
		}
	}

	return result, nil
}

// convertOmniError converts an omni parser error to a base.SyntaxError with proper line:column position.
func convertOmniError(err error, stmt base.Statement) error {
	var parseErr *omniparser.ParseError
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
