package tidb

import (
	"context"

	"github.com/bytebase/omni/tidb/analysis"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterGetQuerySpan(storepb.Engine_TIDB, GetQuerySpan)
}

// GetQuerySpan returns the query span for the given statement.
func GetQuerySpan(ctx context.Context, gCtx base.GetQuerySpanContext, stmt base.Statement, database, _ string, _ bool) (*base.QuerySpan, error) {
	_, getDB := base.UnnamedSchemaQuerySpanCatalog(ctx, gCtx, wrapDatabase)
	querySpan, err := base.OmniResolvedSpanResult(analysis.ResolveQuerySpan(stmt.Text, analysis.ResolveOptions{
		DefaultDatabase: database,
		Catalog:         analysis.Catalog{GetDatabase: getDB},
	}))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get query span from statement: %s", stmt.Text)
	}
	return querySpan, nil
}
