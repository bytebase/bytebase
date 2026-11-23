package cassandra

import (
	"context"

	"github.com/bytebase/omni/cassandra/analysis"
	"github.com/bytebase/omni/metadata"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterGetQuerySpan(storepb.Engine_CASSANDRA, GetQuerySpan)
}

// GetQuerySpan returns the query span for the given CQL statement.
func GetQuerySpan(ctx context.Context, gCtx base.GetQuerySpanContext, stmt base.Statement, database, _ string, _ bool) (*base.QuerySpan, error) {
	opts := analysis.ResolveOptions{
		DefaultKeyspace: database,
	}
	if gCtx.GetDatabaseMetadataFunc != nil {
		opts.GetDatabaseMetadata = func(keyspace string) (*metadata.DatabaseSchemaMetadata, error) {
			_, meta, err := gCtx.GetDatabaseMetadataFunc(ctx, gCtx.InstanceID, keyspace)
			if err != nil {
				return nil, err
			}
			if meta == nil {
				return nil, nil
			}
			return meta.GetProto(), nil
		}
	}

	span, err := analysis.ResolveQuerySpan(stmt.Text, opts)
	if err != nil {
		return nil, convertError(err, base.Statement{Text: stmt.Text})
	}
	return base.OmniResolvedSpan(span), nil
}
