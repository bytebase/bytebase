package doris

import (
	"github.com/bytebase/omni/doris/analysis"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

func init() {
	base.RegisterExtractChangedResourcesFunc(storepb.Engine_DORIS, extractChangedResources)
}

func extractChangedResources(currentDatabase string, _ string, dbMetadata *model.DatabaseMetadata, asts []base.AST, _ string) (*base.ChangeSummary, error) {
	return base.ExtractTypedChangedResources(dbMetadata, asts, "expected astNode for Doris", true, func(parsed *astNode) (analysis.StatementChanges, bool) {
		if parsed.node == nil {
			return analysis.StatementChanges{}, false
		}
		return analysis.ExtractChangedResources(parsed.node, currentDatabase, parsed.text), true
	})
}
