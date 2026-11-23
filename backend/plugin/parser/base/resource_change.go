package base

import (
	omnianalysis "github.com/bytebase/omni/analysis"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store/model"
)

// AppendStatementChanges merges omni statement changes into summary. If skipEmptyTable
// is true, non-database resources with an empty Table name are ignored.
func AppendStatementChanges(summary *ChangeSummary, changes omnianalysis.StatementChanges, skipEmptyTable bool) {
	for _, r := range changes.Resources {
		if r.DatabaseOnly {
			summary.ChangedResources.AddDatabase(r.Database)
			continue
		}
		if skipEmptyTable && r.Table == "" {
			continue
		}
		summary.ChangedResources.AddTable(r.Database, r.Schema, &storepb.ChangedResourceTable{Name: r.Table}, r.AffectsData)
	}
	summary.DMLStatements = append(summary.DMLStatements, changes.DMLStatements...)
	summary.DMLCount += changes.DMLCount
	summary.InsertCount += changes.InsertCount
}

// ExtractChangedResourcesFromASTs builds a ChangeSummary by extracting changes from each AST in asts.
func ExtractChangedResourcesFromASTs(
	dbMetadata *model.DatabaseMetadata,
	asts []AST,
	skipEmptyTable bool,
	extract func(AST) (omnianalysis.StatementChanges, bool, error),
) (*ChangeSummary, error) {
	summary := &ChangeSummary{
		ChangedResources: model.NewChangedResources(dbMetadata),
	}
	for _, a := range asts {
		changes, ok, err := extract(a)
		if err != nil {
			return nil, err
		}
		if ok {
			AppendStatementChanges(summary, changes, skipEmptyTable)
		}
	}
	return summary, nil
}
