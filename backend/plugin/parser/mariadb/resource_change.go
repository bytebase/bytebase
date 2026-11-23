package mariadb

import (
	"github.com/bytebase/omni/mariadb/analysis"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

func init() {
	base.RegisterExtractChangedResourcesFunc(storepb.Engine_MARIADB, extractChangedResources)
}

func extractChangedResources(currentDatabase string, _ string, dbMetadata *model.DatabaseMetadata, asts []base.AST, _ string) (*base.ChangeSummary, error) {
	return base.ExtractTypedChangedResources(dbMetadata, asts, "expected AST for MariaDB", false, func(node *AST) (analysis.StatementChanges, bool) {
		return analysis.ExtractChangedResources(node.Node, currentDatabase, node.Text), true
	})
}
