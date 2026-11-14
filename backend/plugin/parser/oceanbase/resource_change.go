package oceanbase

import (
	"github.com/bytebase/omni/oceanbase/analysis"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

func init() {
	base.RegisterExtractChangedResourcesFunc(storepb.Engine_OCEANBASE, extractChangedResources)
}

func extractChangedResources(currentDatabase string, _ string, dbMetadata *model.DatabaseMetadata, asts []base.AST, _ string) (*base.ChangeSummary, error) {
	summary := &base.ChangeSummary{
		ChangedResources: model.NewChangedResources(dbMetadata),
	}
	for _, unifiedAST := range asts {
		node, ok := unifiedAST.(*AST)
		if !ok {
			return nil, errors.New("expected AST for MySQL")
		}
		addStatementChanges(summary, analysis.ExtractChangedResources(node.Node, currentDatabase, node.Text))
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
