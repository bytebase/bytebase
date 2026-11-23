package cockroachdb

import (
	"github.com/bytebase/omni/cockroachdb/analysis"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

func init() {
	base.Register(base.Engine{
		Parse:          parseCockroachDBStatements,
		StatementTypes: GetStatementTypes,
		ExtractChanged: extractChangedResources,
	}, storepb.Engine_COCKROACHDB)
}

// AST is the AST implementation for CockroachDB parser.
// It implements the base.AST interface.
type AST struct {
	// StartPosition is the 1-based position where this statement starts.
	StartPosition *storepb.Position
	Stmt          analysis.ParsedStatement
}

// ASTStartPosition implements base.AST interface.
func (a *AST) ASTStartPosition() *storepb.Position {
	return a.StartPosition
}

// parseCockroachDBStatements is the ParseStatementsFunc for CockroachDB.
// Returns []ParsedStatement with both text and AST populated.
func parseCockroachDBStatements(statement string) ([]base.ParsedStatement, error) {
	// First split to get Statement with text and positions (uses PostgreSQL splitter).
	stmts, err := base.SplitMultiSQL(storepb.Engine_COCKROACHDB, statement)
	if err != nil {
		return nil, err
	}

	parsedStmts, err := analysis.ParseScript(statement)
	if err != nil {
		return nil, err
	}

	var result []base.ParsedStatement
	astIndex := 0
	for _, stmt := range stmts {
		ps := base.ParsedStatement{
			Statement: stmt,
		}
		if !stmt.Empty && astIndex < len(parsedStmts) {
			ps.AST = &AST{
				StartPosition: &storepb.Position{Line: 1},
				Stmt:          parsedStmts[astIndex],
			}
			astIndex++
		}
		result = append(result, ps)
	}

	return result, nil
}

// GetStatementTypes returns the types of the statements that have one, in order, each followed by
// the types of the mutations in its CTEs and statement sources, each type once per statement.
func GetStatementTypes(asts []base.AST) ([]storepb.StatementType, error) {
	stmts, err := statementsFromASTs(asts)
	if err != nil {
		return nil, err
	}
	return base.OmniStatementTypes(analysis.StatementTypesFromParsed(stmts)), nil
}

// extractChangedResources resolves an unqualified table to the current database and to a schema of
// the search path: currentSchema, or else the search path synced in dbMetadata, or "public", until
// the statements SET another.
func extractChangedResources(database string, currentSchema string, dbMetadata *model.DatabaseMetadata, asts []base.AST, _ string) (*base.ChangeSummary, error) {
	stmts, err := statementsFromASTs(asts)
	if err != nil {
		return nil, err
	}

	searchPath := []string{currentSchema}
	if currentSchema == "" {
		searchPath = []string{"public"}
		if dbMetadata != nil && len(dbMetadata.GetSearchPath()) > 0 {
			searchPath = dbMetadata.GetSearchPath()
		}
	}

	changes := analysis.ExtractChangedResourcesFromParsed(stmts, database, searchPath, changeMetadata{dbMetadata: dbMetadata})
	summary := &base.ChangeSummary{
		ChangedResources: model.NewChangedResources(dbMetadata),
		DMLCount:         changes.DMLCount,
		InsertCount:      changes.InsertCount,
	}
	for _, resource := range changes.Resources {
		summary.ChangedResources.AddTable(resource.Database, resource.Schema, &storepb.ChangedResourceTable{Name: resource.Table}, resource.AffectsData)
	}
	for _, changedDatabase := range changes.Databases {
		summary.ChangedResources.AddDatabase(changedDatabase)
	}
	for _, dml := range changes.DMLStatements {
		text := dml.Text
		if len(dml.SearchPath) > 0 {
			text = base.WithSearchPath(text, dml.SearchPath)
		}
		summary.DMLStatements = append(summary.DMLStatements, text)
	}
	return summary, nil
}

func statementsFromASTs(asts []base.AST) ([]analysis.ParsedStatement, error) {
	stmts := make([]analysis.ParsedStatement, 0, len(asts))
	for _, ast := range asts {
		crdbAST, ok := ast.(*AST)
		if !ok {
			return nil, errors.New("expected CockroachDB AST")
		}
		stmts = append(stmts, crdbAST.Stmt)
	}
	return stmts, nil
}

// changeMetadata answers omni's relation and index lookups with the metadata
// model's own name matching.
type changeMetadata struct {
	dbMetadata *model.DatabaseMetadata
}

func (m changeMetadata) RelationSchema(searchPath []string, name string) string {
	if m.dbMetadata == nil {
		return ""
	}
	for _, schemaName := range searchPath {
		schema := m.dbMetadata.GetSchemaMetadata(schemaName)
		if schema == nil {
			continue
		}
		if schema.GetTable(name) != nil || schema.GetView(name) != nil || schema.GetMaterializedView(name) != nil {
			return schemaName
		}
	}
	return ""
}

func (m changeMetadata) IndexTable(searchPath []string, name string) (string, string, bool) {
	if m.dbMetadata == nil {
		return "", "", false
	}
	schema, indexMetadata := m.dbMetadata.SearchIndex(searchPath, name)
	if indexMetadata == nil {
		return "", "", false
	}
	return schema, indexMetadata.GetTableProto().GetName(), true
}
