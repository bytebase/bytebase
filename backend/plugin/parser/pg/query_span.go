package pg

import (
	"context"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func init() {
	base.RegisterGetQuerySpan(storepb.Engine_POSTGRES, GetQuerySpan)
	base.RegisterGetQuerySpan(storepb.Engine_REDSHIFT, GetQuerySpan)
	base.RegisterGetQuerySpan(storepb.Engine_RISINGWAVE, GetQuerySpan)
	base.RegisterGetQuerySpan(storepb.Engine_COCKROACHDB, GetQuerySpan)
}

// GetQuerySpan returns the query span for the given statement.
func GetQuerySpan(ctx context.Context, gCtx base.GetQuerySpanContext, statement, database, schema string, _ bool) (*base.QuerySpan, error) {
	if gCtx.GetDatabaseMetadataFunc == nil {
		return nil, errors.New("GetDatabaseMetadataFunc is not set in GetQuerySpanContext")
	}
	_, meta, err := gCtx.GetDatabaseMetadataFunc(ctx, gCtx.InstanceID, database)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get database metadata for instance %q and database %q", gCtx.InstanceID, database)
	}
	searchPath := meta.GetSearchPath()
	if schema != "" {
		searchPath = []string{schema}
	}
	extractor := newQuerySpanExtractor(database, searchPath, gCtx)

	querySpan, err := extractor.getQuerySpan(ctx, statement)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get query span from statement: %s", statement)
	}
	return querySpan, nil
}
