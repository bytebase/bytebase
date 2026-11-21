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
	changedResources := model.NewChangedResources(dbMetadata)

	var dmlCount, insertCount int
	var dmlStatements []string

	for _, unifiedAST := range asts {
		parsed, ok := unifiedAST.(*astNode)
		if !ok {
			return nil, errors.New("expected astNode for Doris")
		}
		if parsed.node == nil {
			continue
		}

		changes := analysis.ExtractChangedResources(parsed.node, currentDatabase, parsed.text)
		for _, r := range changes.Resources {
			if r.DatabaseOnly {
				changedResources.AddDatabase(r.Database)
				continue
			}
			if r.Table == "" {
				continue
			}
			changedResources.AddTable(r.Database, "", &storepb.ChangedResourceTable{Name: r.Table}, r.AffectsData)
		}
		dmlStatements = append(dmlStatements, changes.DMLStatements...)
		dmlCount += changes.DMLCount
		insertCount += changes.InsertCount
	}

	return &base.ChangeSummary{
		ChangedResources: changedResources,
		DMLStatements:    dmlStatements,
		DMLCount:         dmlCount,
		InsertCount:      insertCount,
	}, nil
}
