package trino

import (
	"context"
	stderrors "errors"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/bytebase/omni/trino/analysis"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterGetQuerySpan(storepb.Engine_TRINO, GetQuerySpan)
}

// GetQuerySpan gets the query span for a Trino statement. It is the entry point
// registered with the base package.
func GetQuerySpan(ctx context.Context, gCtx base.GetQuerySpanContext, stmt base.Statement, database, schema string, ignoreCaseSensitive bool) (*base.QuerySpan, error) {
	opts := analysis.ResolveOptions{
		DefaultDatabase:     database,
		DefaultSchema:       schema,
		IgnoreCaseSensitive: ignoreCaseSensitive,
	}
	if gCtx.ListDatabaseNamesFunc != nil {
		opts.ListDatabases = func() ([]string, error) {
			return gCtx.ListDatabaseNamesFunc(ctx, gCtx.InstanceID)
		}
	}
	if gCtx.GetDatabaseMetadataFunc != nil {
		opts.GetDatabaseMetadata = func(db string) (*metadatapb.DatabaseSchemaMetadata, error) {
			_, meta, err := gCtx.GetDatabaseMetadataFunc(ctx, gCtx.InstanceID, db)
			if err != nil {
				return nil, base.ToOmniNotFoundError(err)
			}
			if meta == nil {
				return nil, nil
			}
			return meta.GetProto(), nil
		}
	}

	resolved, err := analysis.ResolveQuerySpan(stmt.Text, opts)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get query span from statement: %s", stmt.Text)
	}

	span := &base.QuerySpan{
		Type:             base.OmniQueryType(resolved.Type),
		SourceColumns:    base.OmniColumnSet(resolved.SourceColumns),
		PredicateColumns: base.OmniColumnSet(resolved.PredicateColumns),
		Results:          base.OmniQuerySpanResults(resolved.Results),
	}
	if resolved.NotFoundError != nil {
		converted := base.OmniNotFoundError(resolved.NotFoundError)
		var nf *base.ResourceNotFoundError
		if stderrors.As(converted, &nf) {
			span.NotFoundError = nf
		} else {
			span.NotFoundError = &base.ResourceNotFoundError{Err: resolved.NotFoundError}
		}
	}
	return span, nil
}
