package tsql

import (
	"errors"
	"fmt"
	"strings"
	"unicode"

	mssqlparser "github.com/bytebase/omni/mssql/parser"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterParseStatementsFunc(storepb.Engine_MSSQL, parseTSQLStatements)
	base.RegisterGetStatementTypes(storepb.Engine_MSSQL, GetStatementTypes)
}

func parseTSQLStatements(statement string) ([]base.ParsedStatement, error) {
	// Split once to get Statement with text and positions.
	stmts, err := SplitSQL(statement)
	if err != nil {
		return nil, err
	}

	var result []base.ParsedStatement
	for _, stmt := range stmts {
		if stmt.Empty {
			continue
		}

		omniStmts, omniErr := ParseTSQLOmni(stmt.Text)
		if omniErr != nil {
			return nil, convertOmniError(omniErr, stmt)
		}

		if len(omniStmts) == 0 {
			continue
		}

		for _, os := range omniStmts {
			if os.Empty() {
				continue
			}
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
func convertOmniError(err error, stmt base.Statement) error {
	var parseErr *mssqlparser.ParseError
	if !errors.As(err, &parseErr) {
		return err
	}

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

func NormalizeTSQLIdentifierText(text string) (original string, lowercase string) {
	if text == "" {
		return "", ""
	}
	if text[0] == '[' && text[len(text)-1] == ']' {
		text = text[1 : len(text)-1]
	}

	s := ""
	for _, r := range text {
		s += string(unicode.ToLower(r))
	}
	return text, s
}

// IsTSQLReservedKeyword returns true if the given keyword is a TSQL keywords.
func IsTSQLReservedKeyword(keyword string, caseSensitive bool) bool {
	if !caseSensitive {
		keyword = strings.ToUpper(keyword)
	}
	return tsqlReservedKeywordsMap[keyword]
}
