package base

import (
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store/model"
)

// ChangedResource describes a single database or table change reported by an engine analyzer.
type ChangedResource struct {
	Database     string
	Schema       string
	Table        string
	AffectsData  bool
	DatabaseOnly bool
}

// StatementChanges captures resource and DML changes extracted from one or more statements.
type StatementChanges struct {
	Resources     []ChangedResource
	DMLStatements []string
	DMLCount      int
	InsertCount   int
}

// AppendStatementChanges merges changes into summary. If skipEmptyTable is true,
// non-database resources with an empty Table name are ignored.
func AppendStatementChanges(summary *ChangeSummary, changes StatementChanges, skipEmptyTable bool) {
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
	extract func(AST) (StatementChanges, bool, error),
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

// MapChangedResources converts engine-specific changed resource entries to []ChangedResource.
func MapChangedResources[T any](items []T, mapFn func(T) ChangedResource) []ChangedResource {
	out := make([]ChangedResource, 0, len(items))
	for _, item := range items {
		out = append(out, mapFn(item))
	}
	return out
}
