package tidb

import (
	"context"

	"github.com/bytebase/omni/tidb/analysis"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterGetQuerySpan(storepb.Engine_TIDB, GetQuerySpan)
}

// GetQuerySpan returns the query span for the given statement.
func GetQuerySpan(ctx context.Context, gCtx base.GetQuerySpanContext, stmt base.Statement, database, _ string, _ bool) (*base.QuerySpan, error) {
	querySpan, err := resolveQuerySpan(ctx, gCtx, stmt.Text, database)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get query span from statement: %s", stmt.Text)
	}
	return querySpan, nil
}

func resolveQuerySpan(ctx context.Context, gCtx base.GetQuerySpanContext, statement, database string) (*base.QuerySpan, error) {
	span, err := analysis.ResolveQuerySpan(statement, analysis.ResolveOptions{
		DefaultDatabase: database,
		Catalog:         newCatalog(ctx, gCtx),
	})
	if err != nil {
		if mixErr := base.ConvertOmniMixTablesError(err); mixErr != nil {
			return nil, mixErr
		}
		return nil, err
	}
	return base.OmniResolvedSpan(span), nil
}

// newCatalog reads the instance's synced metadata for omni. A lookup error
// carrying a ResourceNotFoundError becomes omni's NotFoundError, which omni
// reports on the span rather than as an error.
func newCatalog(ctx context.Context, gCtx base.GetQuerySpanContext) analysis.Catalog {
	var catalog analysis.Catalog
	if gCtx.GetDatabaseMetadataFunc != nil {
		catalog.GetDatabase = func(name string) (analysis.Database, error) {
			_, database, err := gCtx.GetDatabaseMetadataFunc(ctx, gCtx.InstanceID, name)
			if err != nil {
				return nil, base.ToOmniNotFoundError(err)
			}
			if database == nil {
				return nil, nil
			}
			return &base.UnnamedSchemaDatabase{Database: database}, nil
		}
	}
	return catalog
}
