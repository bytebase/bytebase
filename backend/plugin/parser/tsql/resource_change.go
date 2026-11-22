package tsql

import (
	"github.com/bytebase/omni/mssql/analysis"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

func init() {
	base.RegisterExtractChangedResourcesFunc(storepb.Engine_MSSQL, extractChangedResources)
}

func extractChangedResources(currentDatabase string, currentSchema string, dbMetadata *model.DatabaseMetadata, asts []base.AST, _ string) (*base.ChangeSummary, error) {
	return base.ExtractChangedResourcesFromASTs(dbMetadata, asts, false, func(a base.AST) (base.StatementChanges, bool, error) {
		node, ok := a.(*AST)
		if !ok {
			return base.StatementChanges{}, false, errors.New("expected AST for MSSQL")
		}
		if node.Node == nil {
			return base.StatementChanges{}, false, nil
		}
		changes := analysis.ExtractChangedResources(node.Node, currentDatabase, currentSchema, node.Text)
		return base.StatementChanges{
			Resources: base.MapChangedResources(changes.Resources, func(r analysis.ChangedResource) base.ChangedResource {
				return base.ChangedResource{Database: r.Database, Schema: r.Schema, Table: r.Table, AffectsData: r.AffectsData, DatabaseOnly: r.DatabaseOnly}
			}),
			DMLStatements: changes.DMLStatements,
			DMLCount:      changes.DMLCount,
			InsertCount:   changes.InsertCount,
		}, true, nil
	})
}
