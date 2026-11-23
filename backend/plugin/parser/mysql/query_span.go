package mysql

import (
	"context"

	"github.com/bytebase/omni/mysql/analysis"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterGetQuerySpan(storepb.Engine_MYSQL, GetQuerySpan)
}

// GetQuerySpan returns the query span for the given statement.
func GetQuerySpan(
	ctx context.Context,
	gCtx base.GetQuerySpanContext,
	stmt base.Statement,
	database, _ string,
	ignoreCaseSensitive bool,
) (*base.QuerySpan, error) {
	listDBs, getDB := base.UnnamedSchemaQuerySpanCatalog(ctx, gCtx, wrapDatabase)
	querySpan, err := base.OmniResolvedSpanResult(analysis.ResolveQuerySpan(stmt.Text, analysis.ResolveOptions{
		DefaultDatabase:     database,
		IgnoreCaseSensitive: ignoreCaseSensitive,
		Catalog:             analysis.Catalog{ListDatabaseNames: listDBs, GetDatabase: getDB},
	}))
	if err != nil {
		return nil, convertError(err, stmt)
	}
	return querySpan, nil
}
