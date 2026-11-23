package cassandra

import (
	omniparser "github.com/bytebase/omni/cassandra/parser"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterParseStatementsFunc(storepb.Engine_CASSANDRA, parseCassandraStatements)
}

func parseCassandraStatements(statement string) ([]base.ParsedStatement, error) {
	return base.ParseNonEmptyMultiNodeStatements(statement, SplitSQL, func(stmt base.Statement) ([]base.AST, error) {
		parsed, err := ParseCQL(stmt.Text)
		if err != nil {
			return nil, convertError(err, stmt)
		}
		asts := make([]base.AST, len(parsed))
		for i, os := range parsed {
			asts[i] = &AST{Node: os.AST, Text: stmt.Text, StartPosition: stmt.Start}
		}
		return asts, nil
	})
}

func convertError(err error, stmt base.Statement) error {
	return base.ConvertSyntaxError(stmt, err, true,
		func(pe *omniparser.ParseError) (int, string) { return pe.Position, pe.Message },
		func(pe *omniparser.ParseError, pos *storepb.Position) string {
			return base.FormatLineSyntaxError(pos, pe.Message)
		},
	)
}
