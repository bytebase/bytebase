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
	summary := &base.ChangeSummary{
		ChangedResources: model.NewChangedResources(dbMetadata),
	}
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
	for _, a := range asts {
		wrapped, ok := a.(*AST)
		if !ok || wrapped == nil {
			continue
		}
		addStatementChanges(summary, analysis.ExtractChangedResources(wrapped.Node, currentDatabase, wrapped.Text, indexTable))
	}
	return summary, nil
}

func addStatementChanges(summary *base.ChangeSummary, changes analysis.StatementChanges) {
	for _, resource := range changes.Resources {
		if resource.DatabaseOnly {
			summary.ChangedResources.AddDatabase(resource.Database)
			continue
		}
		summary.ChangedResources.AddTable(resource.Database, "", &storepb.ChangedResourceTable{Name: resource.Table}, resource.AffectsData)
	}
	summary.DMLStatements = append(summary.DMLStatements, changes.DMLStatements...)
	summary.DMLCount += changes.DMLCount
	summary.InsertCount += changes.InsertCount
}
