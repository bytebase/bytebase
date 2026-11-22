package pg

import (
	"github.com/bytebase/omni/pg/analysis"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

func init() {
	base.RegisterExtractChangedResourcesFunc(storepb.Engine_POSTGRES, extractChangedResources)
}

func extractChangedResources(database string, currentSchema string, dbMetadata *model.DatabaseMetadata, asts []base.AST, _ string) (*base.ChangeSummary, error) {
	changedResources := model.NewChangedResources(dbMetadata)
	searchPath := dbMetadata.GetSearchPath()
	if currentSchema != "" {
		searchPath = analysis.SearchPathForSelectedSchema(currentSchema, searchPath)
	}

	if len(asts) == 0 {
		return &base.ChangeSummary{
			ChangedResources: changedResources,
			DMLCount:         0,
			DMLStatements:    []string{},
			InsertCount:      0,
		}, nil
	}

	stmts := make([]analysis.ChangeStatement, 0, len(asts))
	for _, unifiedAST := range asts {
		wrapped, ok := unifiedAST.(*AST)
		if !ok {
			return nil, errors.New("expected AST for PostgreSQL")
		}
		stmts = append(stmts, analysis.ChangeStatement{Node: wrapped.Node, Text: wrapped.Text})
	}
	changes := analysis.ExtractChangedResources(stmts, database, searchPath, &changeMetadata{database: dbMetadata})

	for _, resource := range changes.Resources {
		changedResources.AddTable(resource.Database, resource.Schema, &storepb.ChangedResourceTable{Name: resource.Table}, resource.AffectsData)
	}
	var dmlStatements []string
	for _, statement := range changes.DMLStatements {
		text := statement.Text
		if statement.SearchPath != nil {
			text = base.WithSearchPath(text, statement.SearchPath)
		}
		dmlStatements = append(dmlStatements, text)
	}
	return &base.ChangeSummary{
		ChangedResources: changedResources,
		DMLCount:         changes.DMLCount,
		DMLStatements:    dmlStatements,
		InsertCount:      changes.InsertCount,
	}, nil
}

// changeMetadata answers omni's lookups with the metadata model's own name
// matching.
type changeMetadata struct {
	database *model.DatabaseMetadata
}

func (m *changeMetadata) HasSchema(name string) bool {
	return m.database.GetSchemaMetadata(name) != nil
}

func (m *changeMetadata) RelationSchema(searchPath []string, name string) string {
	for _, schemaName := range searchPath {
		schema := m.database.GetSchemaMetadata(schemaName)
		if schema == nil {
			continue
		}
		if schema.GetTable(name) != nil || schema.GetView(name) != nil || schema.GetMaterializedView(name) != nil || schema.GetExternalTable(name) != nil {
			return schema.GetProto().GetName()
		}
	}
	return ""
}

func (m *changeMetadata) IndexTable(searchPath []string, name string) (string, string, bool) {
	schemaName, index := m.database.SearchIndex(searchPath, name)
	if index == nil {
		return "", "", false
	}
	table := index.GetTableProto()
	if table == nil {
		return "", "", false
	}
	return schemaName, table.GetName(), true
}
