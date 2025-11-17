package pg

import (
	"context"

	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterGetQuerySpan(storepb.Engine_POSTGRES, GetQuerySpan)
	base.RegisterGetQuerySpan(storepb.Engine_REDSHIFT, GetQuerySpan)
	base.RegisterGetQuerySpan(storepb.Engine_COCKROACHDB, GetQuerySpan)
}

// GetQuerySpan returns the query span for the given statement.
func GetQuerySpan(ctx context.Context, gCtx base.GetQuerySpanContext, statement, database, schema string, _ bool) (*base.QuerySpan, error) {
	parseResults, err := ParsePostgreSQL(statement)
	if err != nil {
		return nil, err
	}

	if len(parseResults) == 0 {
		return &base.QuerySpan{
			Results:          []base.QuerySpanResult{},
			SourceColumns:    base.SourceColumnSet{},
			PredicateColumns: base.SourceColumnSet{},
		}, nil
	}

	if len(parseResults) != 1 {
		return nil, errors.Errorf("expecting only one statement to get query span, but got %d", len(parseResults))
	}

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

	// Use the new ANTLR-based implementation
	querySpan, err := extractor.getQuerySpan(ctx, parseResults[0])
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get query span from statement: %s", statement)
	}
	return querySpan, nil
}
