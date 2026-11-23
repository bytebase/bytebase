package redshift

import (
	"context"

	"github.com/bytebase/omni/metadata"
	"github.com/bytebase/omni/redshift/analysis"
	pkgerrors "github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterGetQuerySpan(storepb.Engine_REDSHIFT, GetQuerySpan)
}

// GetQuerySpan returns the query span for the given statement.
func GetQuerySpan(ctx context.Context, gCtx base.GetQuerySpanContext, stmt base.Statement, database, schema string, _ bool) (*base.QuerySpan, error) {
	if gCtx.GetDatabaseMetadataFunc == nil {
		return nil, pkgerrors.New("GetDatabaseMetadataFunc is not set in GetQuerySpanContext")
	}
	_, meta, err := gCtx.GetDatabaseMetadataFunc(ctx, gCtx.InstanceID, database)
	if err != nil {
		return nil, pkgerrors.Wrapf(err, "failed to get database metadata for instance %q and database %q", gCtx.InstanceID, database)
	}
	if meta == nil {
		return nil, pkgerrors.Errorf("database metadata for database %q not found (database not synced)", database)
	}
	searchPath := meta.GetSearchPath()
	if schema != "" {
		searchPath = []string{schema}
	}

	span, err := analysis.ResolveQuerySpan(stmt.Text, analysis.ResolveOptions{
		DefaultDatabase: database,
		SearchPath:      searchPath,
		GetDatabaseMetadata: func(db string) (*metadata.DatabaseSchemaMetadata, error) {
			_, dbMeta, err := gCtx.GetDatabaseMetadataFunc(ctx, gCtx.InstanceID, db)
			if err != nil {
				return nil, err
			}
			if dbMeta == nil {
				return nil, nil
			}
			return dbMeta.GetProto(), nil
		},
	})
	if err != nil {
		if mixErr := base.ConvertOmniMixTablesError(err); mixErr != nil {
			return nil, mixErr
		}
		err = convertError(err, stmt)
		return nil, pkgerrors.Wrapf(err, "failed to get query span from statement: %s", stmt.Text)
	}

	return &base.QuerySpan{
		Type:                      base.OmniQueryType(span.Type),
		SourceColumns:             base.OmniColumnSet(span.SourceColumns),
		PredicateColumns:          base.OmniColumnSet(span.PredicateColumns),
		Results:                   base.OmniQuerySpanResults(span.Results),
		NotFoundError:             base.OmniNotFoundError(span.NotFoundError),
		FunctionNotSupportedError: base.OmniFunctionNotSupportedError(span.FunctionNotSupportedError),
	}, nil
}
