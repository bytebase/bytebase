package googlesql

import (
	"context"
	"errors"

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
				var notFound *base.ResourceNotFoundError
				if errors.As(err, &notFound) {
					return nil, &analysis.ResourceNotFoundError{
						Err:      notFound.Err,
						Database: notFound.Database,
						Schema:   notFound.Schema,
						Table:    notFound.Table,
						Column:   notFound.Column,
					}
				}
				return nil, err
			}
			if meta == nil {
				return nil, nil
			}
			return meta.GetProto(), nil
		},
	})
	if err != nil {
		if errors.Is(err, analysis.ErrMixUserSystemTables) {
			return nil, base.MixUserSystemTablesError
		}
		return nil, err
	}
	return &base.QuerySpan{
		Type:          MapQueryType(span.Type),
		SourceColumns: convertColumnSet(span.SourceColumns),
		Results:       convertResults(span.Results),
		NotFoundError: convertNotFoundError(span.NotFoundError),
	}, nil
}

func convertColumnSet(set analysis.ColumnSet) base.SourceColumnSet {
	return base.ConvertColumnSet(set, func(c analysis.ColumnResource) base.ColumnResource {
		return base.ColumnResource{Database: c.Database, Schema: c.Schema, Table: c.Table, Column: c.Column}
	})
}

func convertResults(results []analysis.ResolvedQuerySpanResult) []base.QuerySpanResult {
	return base.ConvertQuerySpanResults(results, func(r analysis.ResolvedQuerySpanResult) base.QuerySpanResult {
		return base.QuerySpanResult{
			Name:          r.Name,
			SourceColumns: convertColumnSet(r.SourceColumns),
			IsPlainField:  r.IsPlainField,
		}
	})
}

func convertNotFoundError(err error) error {
	if err == nil {
		return nil
	}
	var notFound *analysis.ResourceNotFoundError
	if errors.As(err, &notFound) {
		return &base.ResourceNotFoundError{
			Err:      notFound.Err,
			Database: notFound.Database,
			Schema:   notFound.Schema,
			Table:    notFound.Table,
			Column:   notFound.Column,
		}
	}
	return err
}
