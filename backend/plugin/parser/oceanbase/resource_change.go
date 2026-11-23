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
	return base.ExtractChangedResourcesFromASTs(dbMetadata, asts, false, func(a base.AST) (analysis.StatementChanges, bool, error) {
		node, ok := a.(*AST)
		if !ok {
			return analysis.StatementChanges{}, false, errors.New("expected AST for MySQL")
		}
		return analysis.ExtractChangedResources(node.Node, currentDatabase, node.Text), true, nil
	})
}
