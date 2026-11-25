package pg

import (
	"context"

	"github.com/bytebase/omni/metadata"
	"github.com/bytebase/omni/pg/analysis"
	omniparser "github.com/bytebase/omni/pg/parser"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

// GetQuerySpan returns the query span for the given statement.
func GetQuerySpan(ctx context.Context, gCtx base.GetQuerySpanContext, stmt base.Statement, database, schema string, _ bool) (*base.QuerySpan, error) {
	meta, err := base.RequireDatabaseMetadata(ctx, gCtx, database)
	if err != nil {
		return nil, err
	}
	searchPath := meta.GetSearchPath()
	if schema != "" {
		searchPath = analysis.SearchPathForSelectedSchema(schema, searchPath)
	}

	span, err := analysis.ResolveQuerySpan(ctx, stmt.Text, analysis.ResolveOptions{
		DefaultDatabase: database,
		SearchPath:      searchPath,
		Catalog: analysis.Catalog{
			GetDatabase: base.QuerySpanDatabaseGetter(ctx, gCtx, false, func(db *model.DatabaseMetadata) analysis.Database {
				return &catalogDatabase{database: db}
			}),
		},
	})
	if syntaxErr, ok := err.(*analysis.SyntaxError); ok {
		err = convertSyntaxError(stmt.Text, syntaxErr)
	}
	querySpan, err := base.OmniResolvedSpanResult(span, err)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get query span from statement: %s", stmt.Text)
	}
	return querySpan, nil
}

// convertSyntaxError places the error one byte before the parser's offset, so
// an error at the end of the statement points at its last character.
func convertSyntaxError(statement string, err *analysis.SyntaxError) *base.SyntaxError {
	syntaxErr := &base.SyntaxError{
		Message:  err.Error(),
		Position: &storepb.Position{Line: 1, Column: 1},
	}
	var parseErr *omniparser.ParseError
	if errors.As(err.Err, &parseErr) && parseErr.Position > 0 {
		syntaxErr.Position = base.ByteOffsetToRunePosition(statement, parseErr.Position-1)
	}
	return syntaxErr
}

// catalogDatabase answers omni's lookups with the metadata model's own name matching.
type catalogDatabase struct {
	database *model.DatabaseMetadata
}

func (d *catalogDatabase) Metadata() *metadata.DatabaseSchemaMetadata {
	return d.database.GetProto()
}

func (d *catalogDatabase) GetTable(schema, table string) *metadata.TableMetadata {
	tableMetadata := d.database.GetSchemaMetadata(schema).GetTable(table)
	if tableMetadata == nil {
		return nil
	}
	return tableMetadata.GetProto()
}

func (d *catalogDatabase) SearchRelation(searchPath []string, name string) (string, string) {
	return d.database.SearchRelation(searchPath, name)
}

func (d *catalogDatabase) SearchFunctions(searchPath []string, name string) []*metadata.FunctionMetadata {
	_, functions := d.database.SearchFunctions(searchPath, name)
	return functions
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

// changeMetadata answers omni's lookups with the metadata model's own name matching.
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
