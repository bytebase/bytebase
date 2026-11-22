package doris

import (
	"github.com/bytebase/omni/doris/analysis"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

func init() {
	base.RegisterExtractChangedResourcesFunc(storepb.Engine_DORIS, extractChangedResources)
}

func extractChangedResources(currentDatabase string, _ string, dbMetadata *model.DatabaseMetadata, asts []base.AST, _ string) (*base.ChangeSummary, error) {
	return base.ExtractChangedResourcesFromASTs(dbMetadata, asts, true, func(a base.AST) (base.StatementChanges, bool, error) {
		parsed, ok := a.(*astNode)
		if !ok {
			return base.StatementChanges{}, false, errors.New("expected astNode for Doris")
		}
		if parsed.node == nil {
			return base.StatementChanges{}, false, nil
		}
		changes := analysis.ExtractChangedResources(parsed.node, currentDatabase, parsed.text)
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
