package plsql

import (
	"github.com/bytebase/omni/oracle/analysis"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

func init() {
	base.RegisterExtractChangedResourcesFunc(storepb.Engine_ORACLE, extractChangedResources)
}

// extractChangedResources treats currentDatabase as the current schema, as
// Oracle does.
func extractChangedResources(currentDatabase string, _ string, dbMetadata *model.DatabaseMetadata, asts []base.AST, _ string) (*base.ChangeSummary, error) {
	var indexTable analysis.IndexTableFunc
	if dbMetadata != nil {
		indexTable = func(schema, index string) (string, bool) {
			foundIndex := dbMetadata.GetSchemaMetadata(schema).GetIndex(index)
			if foundIndex == nil {
				return "", false
			}
			return foundIndex.GetTableProto().GetName(), true
		}
	}
	return base.ExtractChangedResourcesFromASTs(dbMetadata, asts, false, func(a base.AST) (base.StatementChanges, bool, error) {
		wrapped, ok := a.(*AST)
		if !ok || wrapped == nil {
			return base.StatementChanges{}, false, nil
		}
		changes := analysis.ExtractChangedResources(wrapped.Node, currentDatabase, wrapped.Text, indexTable)
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
