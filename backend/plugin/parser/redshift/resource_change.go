package redshift

import (
	"github.com/bytebase/omni/redshift/analysis"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

func init() {
	base.RegisterExtractChangedResourcesFunc(storepb.Engine_REDSHIFT, extractChangedResources)
}

func extractChangedResources(database string, schema string, dbMetadata *model.DatabaseMetadata, _ []base.AST, statement string) (*base.ChangeSummary, error) {
	changes, err := analysis.ExtractChangedResources(statement, database, schema)
	if err != nil {
		return nil, err
	}

	changedResources := model.NewChangedResources(dbMetadata)
	for _, resource := range changes.Resources {
		changedResources.AddTable(
			resource.Database,
			resource.Schema,
			&storepb.ChangedResourceTable{Name: resource.Table},
			resource.AffectsData,
		)
	}
	return &base.ChangeSummary{
		ChangedResources: changedResources,
		DMLStatements:    changes.DMLStatements,
		DMLCount:         changes.DMLCount,
		InsertCount:      changes.InsertCount,
	}, nil
}
