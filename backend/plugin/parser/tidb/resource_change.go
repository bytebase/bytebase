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
	summary := &base.ChangeSummary{
		ChangedResources: model.NewChangedResources(dbMetadata),
	}
	for _, a := range asts {
		wrapped, ok := a.(*AST)
		if !ok {
			return nil, errors.New("expected TiDB AST")
		}
		changes := analysis.ExtractChangedResources(wrapped.Node, database, wrapped.Text)
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
	return summary, nil
}
