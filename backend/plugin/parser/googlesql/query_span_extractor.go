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
	getMeta := base.QuerySpanProtoMetadataFunc(ctx, q.gCtx, true)
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
			if getMeta == nil {
				return nil, &analysis.ResourceNotFoundError{Database: &database}
			}
			return getMeta(database)
		},
	})
	if err != nil {
		if mixErr := base.ConvertOmniMixTablesError(err); mixErr != nil {
			return nil, mixErr
		}
		return nil, err
	}
	return base.OmniQuerySpan(span.Type, span.SourceColumns, nil, span.Results, span.NotFoundError), nil
}
