package googlesql

import (
	"context"

	"github.com/bytebase/omni/googlesql/analysis"
	"github.com/bytebase/omni/metadata"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// QuerySpanExtractor analyses a single GoogleSQL statement and produces a
// base.QuerySpan: the set of base tables it accesses, the per-output column
// lineage (expanded against catalog metadata when available), and its statement
// classification.
type QuerySpanExtractor struct {
	cfg             Config
	defaultDatabase string
	gCtx            base.GetQuerySpanContext
}

// NewQuerySpanExtractor creates a query span extractor for one statement
// evaluation.
func NewQuerySpanExtractor(cfg Config, defaultDatabase string, gCtx base.GetQuerySpanContext) *QuerySpanExtractor {
	return &QuerySpanExtractor{
		cfg:             cfg,
		defaultDatabase: defaultDatabase,
		gCtx:            gCtx,
	}
}

// GetQuerySpan extracts the query span for a single GoogleSQL statement.
func (q *QuerySpanExtractor) GetQuerySpan(ctx context.Context, statement string) (*base.QuerySpan, error) {
	span, err := analysis.ResolveQuerySpan(statement, analysis.ResolveOptions{
		Dialect:                q.cfg.Dialect,
		DefaultDatabase:        q.defaultDatabase,
		IgnoreWrittenDatabase:  q.cfg.IgnoreWrittenDatabase,
		SchemaScopedMetadata:   q.cfg.SchemaScopedMetadata,
		SystemOnlyEmptySpan:    q.cfg.SystemOnlyEmptySpan,
		SetStatementIsSelect:   q.cfg.SetStatementIsSelect,
		UppercaseStarMergeName: q.cfg.UppercaseStarMergeName,
		CanonicalBaseFieldName: q.cfg.CanonicalBaseFieldName,
		GetDatabaseMetadata: func(database string) (*metadata.DatabaseSchemaMetadata, error) {
			if q.gCtx.GetDatabaseMetadataFunc == nil {
				return nil, &analysis.ResourceNotFoundError{Database: &database}
			}
			_, meta, err := q.gCtx.GetDatabaseMetadataFunc(ctx, q.gCtx.InstanceID, database)
			if err != nil {
				return nil, base.ToOmniNotFoundError(err)
			}
			if meta == nil {
				return nil, nil
			}
			return meta.GetProto(), nil
		},
	})
	if err != nil {
		if mixErr := base.ConvertOmniMixTablesError(err); mixErr != nil {
			return nil, mixErr
		}
		return nil, err
	}
	return &base.QuerySpan{
		Type:          MapQueryType(span.Type),
		SourceColumns: base.OmniColumnSet(span.SourceColumns),
		Results:       base.OmniQuerySpanResults(span.Results),
		NotFoundError: base.OmniNotFoundError(span.NotFoundError),
	}, nil
}
