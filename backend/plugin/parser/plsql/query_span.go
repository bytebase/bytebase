package plsql

import (
	"context"

	"github.com/bytebase/omni/oracle/analysis"
	pkgerrors "github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterGetQuerySpan(storepb.Engine_ORACLE, GetQuerySpan)
}

func GetQuerySpan(ctx context.Context, gCtx base.GetQuerySpanContext, stmt base.Statement, database, _ string, _ bool) (*base.QuerySpan, error) {
	querySpan, err := base.OmniResolvedSpanResult(analysis.ResolveQuerySpan(stmt.Text, analysis.ResolveOptions{
		Instance:        gCtx.InstanceID,
		DefaultDatabase: database,
		Catalog:         newCatalog(ctx, gCtx),
	}))
	if err != nil {
		return nil, pkgerrors.Wrapf(err, "failed to get query span from statement: %s", stmt.Text)
	}
	return querySpan, nil
}

// newCatalog reads synced metadata for omni. A nil metadata function stays
// nil, so omni fails where the extractor did.
func newCatalog(ctx context.Context, gCtx base.GetQuerySpanContext) analysis.Catalog {
	var catalog analysis.Catalog
	if gCtx.GetDatabaseMetadataFunc != nil {
		catalog.GetDatabase = func(instance, name string) (analysis.Database, error) {
			_, database, err := gCtx.GetDatabaseMetadataFunc(ctx, instance, name)
			if err != nil {
				return nil, base.ToOmniNotFoundError(err)
			}
			if database == nil {
				return nil, nil
			}
			return &base.UnnamedSchemaDatabase{Database: database}, nil
		}
	}
	if gCtx.GetLinkedDatabaseMetadataFunc != nil {
		catalog.GetLinkedDatabase = func(instance, link, schema string) (string, string, analysis.Database, error) {
			linkedInstance, name, database, err := gCtx.GetLinkedDatabaseMetadataFunc(ctx, instance, link, schema)
			if err != nil {
				return "", "", nil, base.ToOmniNotFoundError(err)
			}
			if database == nil {
				return linkedInstance, name, nil, nil
			}
			return linkedInstance, name, &base.UnnamedSchemaDatabase{Database: database}, nil
		}
	}
	return catalog
}
