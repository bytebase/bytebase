package mysql

import (
	"errors"
	"fmt"

	mariadbomniparser "github.com/bytebase/omni/mariadb/parser"
	mysqlomniparser "github.com/bytebase/omni/mysql/parser"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/plugin/parser/mariadb"
)

func init() {
	base.RegisterParseStatementsFunc(storepb.Engine_MYSQL, parseMySQLStatements)
	base.RegisterParseStatementsFunc(storepb.Engine_MARIADB, parseMariaDBStatements)
	base.RegisterParseStatementsFunc(storepb.Engine_OCEANBASE, parseMySQLStatements)
	base.RegisterGetStatementTypes(storepb.Engine_MYSQL, GetStatementTypes)
	base.RegisterGetStatementTypes(storepb.Engine_MARIADB, GetStatementTypes)
	base.RegisterGetStatementTypes(storepb.Engine_OCEANBASE, GetStatementTypes)
}

// parseMySQLStatements is the ParseStatementsFunc for MySQL and OceanBase.
// Returns []ParsedStatement with both text and AST populated.
func parseMySQLStatements(statement string) ([]base.ParsedStatement, error) {
	return parseStatements(statement, false /* mariaDBFallback */)
}

// parseMariaDBStatements is the ParseStatementsFunc for MariaDB. The MySQL code
// paths read omni's MySQL AST, so each statement goes to omni's MySQL parser
// first. A statement only omni's MariaDB parser accepts, such as CREATE OR
// REPLACE TABLE, gets a *mariadb.OmniAST instead, which those paths do not read.
func parseMariaDBStatements(statement string) ([]base.ParsedStatement, error) {
	return parseStatements(statement, true /* mariaDBFallback */)
}

func parseStatements(statement string, mariaDBFallback bool) ([]base.ParsedStatement, error) {
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

		list, omniErr := ParseMySQL(stmt.Text)
		if omniErr != nil {
			if !mariaDBFallback {
				return nil, convertOmniError(omniErr, stmt)
			}
			mariaDBList, mariaDBErr := mariadb.ParseMariaDB(stmt.Text)
			if mariaDBErr != nil {
				return nil, convertOmniError(mariaDBErr, stmt)
			}
			if mariaDBList == nil {
				continue
			}
			for _, node := range mariaDBList.Items {
				result = append(result, base.ParsedStatement{
					Statement: stmt,
					AST: &mariadb.OmniAST{
						Node:          node,
						Text:          stmt.Text,
						StartPosition: stmt.Start,
					},
				})
			}
			continue
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

// convertOmniError converts an omni MySQL or MariaDB parser error to a base.SyntaxError with proper line:column position.
func convertOmniError(err error, stmt base.Statement) error {
	var message, relatedText string
	var position int
	var mysqlErr *mysqlomniparser.ParseError
	var mariaDBErr *mariadbomniparser.ParseError
	switch {
	case errors.As(err, &mysqlErr):
		message, position, relatedText = mysqlErr.Message, mysqlErr.Position, mysqlErr.RelatedText
	case errors.As(err, &mariaDBErr):
		message, position, relatedText = mariaDBErr.Message, mariaDBErr.Position, mariaDBErr.RelatedText
	default:
		return err
	}

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
