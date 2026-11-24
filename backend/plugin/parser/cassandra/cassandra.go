package cassandra

import (
	"context"

	omnicassandra "github.com/bytebase/omni/cassandra"
	"github.com/bytebase/omni/cassandra/analysis"
	"github.com/bytebase/omni/cassandra/ast"
	omniparser "github.com/bytebase/omni/cassandra/parser"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.Register(base.Engine{
		Split:         SplitSQL,
		Parse:         parseCassandraStatements,
		ValidateQuery: analysis.ValidateQuery,
		QuerySpan:     GetQuerySpan,
	}, storepb.Engine_CASSANDRA)
}

// AST wraps an omni AST node and implements the base.AST interface.
type AST = base.NodeAST[ast.Node]

// GetNode extracts the omni AST node from a base.AST interface.
func GetNode(a base.AST) (ast.Node, bool) {
	return base.ExtractNode[ast.Node](a)
}

func SplitSQL(statement string) ([]base.Statement, error) {
	return base.BuildStatements(statement, omnicassandra.Split(statement), func(s omnicassandra.Segment) (int, int, bool) {
		return s.ByteStart, s.ByteEnd, s.Empty
	}, false), nil
}

func parseCassandraStatements(statement string) ([]base.ParsedStatement, error) {
	return base.ParseNonEmptyMultiNodeStatements(statement, SplitSQL, func(stmt base.Statement) ([]base.AST, error) {
		parsed, err := omnicassandra.Parse(stmt.Text)
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

// GetQuerySpan returns the query span for the given CQL statement.
func GetQuerySpan(ctx context.Context, gCtx base.GetQuerySpanContext, stmt base.Statement, database, _ string, _ bool) (*base.QuerySpan, error) {
	span, err := analysis.ResolveQuerySpan(stmt.Text, analysis.ResolveOptions{
		DefaultKeyspace:     database,
		GetDatabaseMetadata: base.QuerySpanProtoMetadataFunc(ctx, gCtx, false),
	})
	if err != nil {
		return nil, convertError(err, base.Statement{Text: stmt.Text})
	}
	return base.OmniResolvedSpan(span), nil
}

func convertError(err error, stmt base.Statement) error {
	return base.ConvertSyntaxError(stmt, err, true,
		func(pe *omniparser.ParseError) (int, string) { return pe.Position, pe.Message },
		func(pe *omniparser.ParseError, pos *storepb.Position) string {
			return base.FormatLineSyntaxError(pos, pe.Message)
		},
	)
}
