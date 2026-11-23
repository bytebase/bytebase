package cassandra

import (
	"context"

	"github.com/bytebase/omni/cassandra/analysis"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterGetQuerySpan(storepb.Engine_CASSANDRA, GetQuerySpan)
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
