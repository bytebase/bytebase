package trino

import (
	"context"
	stderrors "errors"

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
	resolved, err := analysis.ResolveQuerySpan(stmt.Text, analysis.ResolveOptions{
		DefaultDatabase:     database,
		DefaultSchema:       schema,
		IgnoreCaseSensitive: ignoreCaseSensitive,
		ListDatabases:       base.QuerySpanListDatabasesFunc(ctx, gCtx, false),
		GetDatabaseMetadata: base.QuerySpanProtoMetadataFunc(ctx, gCtx, true),
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get query span from statement: %s", stmt.Text)
	}

	span := base.OmniQuerySpan(resolved.Type, resolved.SourceColumns, resolved.PredicateColumns, resolved.Results, nil)
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
