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
	return base.ExtractChangedResourcesFromASTs(dbMetadata, asts, false, func(a base.AST) (analysis.StatementChanges, bool, error) {
		wrapped, ok := a.(*AST)
		if !ok {
			return analysis.StatementChanges{}, false, errors.New("expected TiDB AST")
		}
		return analysis.ExtractChangedResources(wrapped.Node, database, wrapped.Text), true, nil
	})
}
