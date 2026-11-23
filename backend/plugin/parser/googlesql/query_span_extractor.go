package googlesql

import (
	"context"

	"github.com/bytebase/omni/googlesql/analysis"
	"github.com/bytebase/omni/metadata"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func getQuerySpan(ctx context.Context, cfg Config, defaultDatabase string, gCtx base.GetQuerySpanContext, statement string) (*base.QuerySpan, error) {
	getMeta := base.QuerySpanProtoMetadataFunc(ctx, gCtx, true)
	span, err := analysis.ResolveQuerySpan(statement, analysis.ResolveOptions{
		Dialect:                cfg.Dialect,
		DefaultDatabase:        defaultDatabase,
		IgnoreWrittenDatabase:  cfg.IgnoreWrittenDatabase,
		SchemaScopedMetadata:   cfg.SchemaScopedMetadata,
		SystemOnlyEmptySpan:    cfg.SystemOnlyEmptySpan,
		SetStatementIsSelect:   cfg.SetStatementIsSelect,
		UppercaseStarMergeName: cfg.UppercaseStarMergeName,
		CanonicalBaseFieldName: cfg.CanonicalBaseFieldName,
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
