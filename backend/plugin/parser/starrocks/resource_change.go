package starrocks

import (
	"github.com/bytebase/omni/starrocks/analysis"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

func init() {
	base.RegisterExtractChangedResourcesFunc(storepb.Engine_STARROCKS, extractChangedResources)
}

func extractChangedResources(currentDatabase string, _ string, dbMetadata *model.DatabaseMetadata, asts []base.AST, _ string) (*base.ChangeSummary, error) {
	return base.ExtractChangedResourcesFromASTs(dbMetadata, asts, true, func(a base.AST) (analysis.StatementChanges, bool, error) {
		parsed, ok := a.(*astNode)
		if !ok {
			return analysis.StatementChanges{}, false, errors.New("expected astNode for StarRocks")
		}
		if parsed.node == nil {
			return analysis.StatementChanges{}, false, nil
		}
		return analysis.ExtractChangedResources(parsed.node, currentDatabase, parsed.text), true, nil
	})
}
