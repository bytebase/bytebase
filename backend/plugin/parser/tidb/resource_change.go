package tidb

import (
	"github.com/bytebase/omni/tidb/analysis"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

func init() {
	base.RegisterExtractChangedResourcesFunc(storepb.Engine_TIDB, extractChangedResources)
}

func extractChangedResources(database string, _ string, dbMetadata *model.DatabaseMetadata, asts []base.AST, _ string) (*base.ChangeSummary, error) {
	return base.ExtractChangedResourcesFromASTs(dbMetadata, asts, false, func(a base.AST) (base.StatementChanges, bool, error) {
		wrapped, ok := a.(*AST)
		if !ok {
			return base.StatementChanges{}, false, errors.New("expected TiDB AST")
		}
		changes := analysis.ExtractChangedResources(wrapped.Node, database, wrapped.Text)
		return base.StatementChanges{
			Resources: base.MapChangedResources(changes.Resources, func(r analysis.ChangedResource) base.ChangedResource {
				return base.ChangedResource{Database: r.Database, Table: r.Table, AffectsData: r.AffectsData, DatabaseOnly: r.DatabaseOnly}
			}),
			DMLStatements: changes.DMLStatements,
			DMLCount:      changes.DMLCount,
			InsertCount:   changes.InsertCount,
		}, true, nil
	})
}
