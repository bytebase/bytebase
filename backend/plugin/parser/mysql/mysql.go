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
	base.RegisterParseStatementsFunc(storepb.Engine_MARIADB, parseMySQLStatements)
	base.RegisterParseStatementsFunc(storepb.Engine_OCEANBASE, parseMySQLStatements)
	base.RegisterGetStatementTypes(storepb.Engine_MYSQL, GetStatementTypes)
	base.RegisterGetStatementTypes(storepb.Engine_MARIADB, GetStatementTypes)
	base.RegisterGetStatementTypes(storepb.Engine_OCEANBASE, GetStatementTypes)
}

// parseMySQLStatements is the ParseStatementsFunc for MySQL, MariaDB, and OceanBase.
// Returns []ParsedStatement with both text and AST populated.
func parseMySQLStatements(statement string) ([]base.ParsedStatement, error) {
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

		list, omniErr := ParseMySQLOmni(stmt.Text)
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
	var parseErr *mysqlomniparser.ParseError
	if !errors.As(err, &parseErr) {
		return err
	}

	pos := ByteOffsetToRunePosition(stmt.Text, parseErr.Position)

	// Adjust line by the statement's base line (stmt.Start.Line is 1-based).
	if stmt.Start != nil {
		pos.Line += stmt.Start.Line - 1
	}

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
