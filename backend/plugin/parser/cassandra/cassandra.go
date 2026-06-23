package cassandra

import (
	"errors"
	"fmt"

	omniparser "github.com/bytebase/omni/cassandra/parser"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterParseStatementsFunc(storepb.Engine_CASSANDRA, parseCassandraStatements)
}

func parseCassandraStatements(statement string) ([]base.ParsedStatement, error) {
	stmts, err := SplitSQL(statement)
	if err != nil {
		return nil, err
	}

	var result []base.ParsedStatement
	for _, stmt := range stmts {
		if stmt.Empty {
			continue
		}

		omniStmts, omniErr := ParseCQL(stmt.Text)
		if omniErr != nil {
			return nil, convertOmniError(omniErr, stmt)
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

func convertOmniError(err error, stmt base.Statement) error {
	var parseErr *omniparser.ParseError
	if !errors.As(err, &parseErr) {
		return err
	}

	pos := byteOffsetToPosition(stmt.Text, parseErr.Loc.Start)
	if stmt.Start != nil {
		if pos.Line == 1 {
			pos.Column += stmt.Start.Column - 1
		}
		pos.Line += stmt.Start.Line - 1
	}

	return &base.SyntaxError{
		Position:   pos,
		Message:    fmt.Sprintf("Syntax error at line %d: %s", pos.Line, parseErr.Message),
		RawMessage: parseErr.Message,
	}
}
