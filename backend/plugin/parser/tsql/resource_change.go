package tsql

import (
	"github.com/bytebase/omni/mssql/analysis"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

func init() {
	base.RegisterExtractChangedResourcesFunc(storepb.Engine_MSSQL, extractChangedResources)
}

func extractChangedResources(currentDatabase string, currentSchema string, dbMetadata *model.DatabaseMetadata, asts []base.AST, _ string) (*base.ChangeSummary, error) {
	return base.ExtractTypedChangedResources(dbMetadata, asts, "expected AST for MSSQL", false, func(node *AST) (analysis.StatementChanges, bool) {
		if node.Node == nil {
			return analysis.StatementChanges{}, false
		}
		return analysis.ExtractChangedResources(node.Node, currentDatabase, currentSchema, node.Text), true
	})
}
