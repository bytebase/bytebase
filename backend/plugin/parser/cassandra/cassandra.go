package cassandra

import (
	"context"

	"github.com/bytebase/omni/cassandra/analysis"
	omniparser "github.com/bytebase/omni/cassandra/parser"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	engine := base.OmniEngine(analysis.Engine)
	engine.QuerySpan = GetQuerySpan
	base.Register(engine, storepb.Engine_CASSANDRA)
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
