package cassandra

import (
	"context"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/bytebase/parser/cql"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// querySpanExtractor walks the CQL parse tree to extract query span information
type querySpanExtractor struct {
	*cql.BaseCqlParserListener

	// Context
	defaultKeyspace string
	gCtx            base.GetQuerySpanContext

	// Results we're building
	querySpan *base.QuerySpan

	// Error handling
	err error
}

// unquoteIdentifier removes double quotes from Cassandra identifiers if present.
func unquoteIdentifier(identifier string) string {
	if len(identifier) >= 2 && identifier[0] == '"' && identifier[len(identifier)-1] == '"' {
		return identifier[1 : len(identifier)-1]
	}
	return identifier
}

func newQuerySpanExtractor(defaultKeyspace string, gCtx base.GetQuerySpanContext) *querySpanExtractor {
	return &querySpanExtractor{
		BaseCqlParserListener: &cql.BaseCqlParserListener{},
		defaultKeyspace:       defaultKeyspace,
		gCtx:                  gCtx,
		querySpan: &base.QuerySpan{
			Type:             base.QueryTypeUnknown,
			Results:          []base.QuerySpanResult{},
			SourceColumns:    base.SourceColumnSet{},
			PredicateColumns: base.SourceColumnSet{},
		},
	}
}

// EnterSelect_ is called when we enter a SELECT statement
func (e *querySpanExtractor) EnterSelect_(ctx *cql.Select_Context) {
	if e.err != nil {
		return
	}

	// Set query type to SELECT
	e.querySpan.Type = base.Select

	keyspace, table := e.extractTableFromFromSpec(ctx.FromSpec())
	if keyspace == "" {
		keyspace = e.defaultKeyspace
	}

	// Add accessed table to SourceColumns (table-level resource)
	if table != "" {
		e.querySpan.SourceColumns[base.ColumnResource{
			Database: keyspace,
			Table:    table,
			Column:   "", // Empty column means table-level access
		}] = true
	}

	results := e.extractSelectElements(ctx.SelectElements(), keyspace, table)
	e.querySpan.Results = results

	if whereSpec := ctx.WhereSpec(); whereSpec != nil {
		e.extractWhereColumns(whereSpec, keyspace, table)
	}
}

// extractTableFromFromSpec extracts keyspace and table name from FROM clause
func (*querySpanExtractor) extractTableFromFromSpec(fromSpec cql.IFromSpecContext) (keyspace, table string) {
	if fromSpec == nil {
		return "", ""
	}

	if fromElem := fromSpec.FromSpecElement(); fromElem != nil {
		text := fromElem.GetText()
		// CQL supports keyspace.table in FROM clause
		if idx := strings.LastIndex(text, "."); idx > 0 {
			keyspacePart := text[:idx]
			tablePart := text[idx+1:]
			return unquoteIdentifier(keyspacePart), unquoteIdentifier(tablePart)
		}
		return "", unquoteIdentifier(text)
	}
	return "", ""
}

// extractSelectElements extracts column information from SELECT clause
func (e *querySpanExtractor) extractSelectElements(selectElements cql.ISelectElementsContext, keyspace, table string) []base.QuerySpanResult {
	if selectElements == nil {
		return nil
	}

	if selectElements.GetStar() != nil {
		return e.expandSelectAsterisk(keyspace, table)
	}

	var results []base.QuerySpanResult
	for _, elem := range selectElements.AllSelectElement() {
		aliasName, sourceName := e.extractColumnNameAndAlias(elem)
		if aliasName != "" || sourceName != "" {
			resultName := aliasName
			if resultName == "" {
				resultName = sourceName
			}

			sourceColumn := base.SourceColumnSet{}
			sourceColumn[base.ColumnResource{
				Database: keyspace,
				Table:    table,
				Column:   sourceName,
			}] = true

			results = append(results, base.QuerySpanResult{
				Name:          resultName,
				SourceColumns: sourceColumn,
				IsPlainField:  true,
			})
		}
	}
	return results
}

// extractColumnNameAndAlias extracts both the alias (if present) and the source column name
func (*querySpanExtractor) extractColumnNameAndAlias(elem cql.ISelectElementContext) (alias string, sourceColumn string) {
	if elem == nil {
		return "", ""
	}

	// Extract the source column name from the first child
	if elem.GetChildCount() > 0 {
		columnRef := elem.GetChild(0).(antlr.ParseTree).GetText()
		sourceColumn = unquoteIdentifier(columnRef)
	}

	// Check for AS alias
	for i := 0; i < elem.GetChildCount(); i++ {
		if child := elem.GetChild(i); child != nil {
			childText := child.(antlr.ParseTree).GetText()
			if strings.ToUpper(childText) == "AS" && i+1 < elem.GetChildCount() {
				aliasText := elem.GetChild(i + 1).(antlr.ParseTree).GetText()
				alias = unquoteIdentifier(aliasText)
				break
			}
		}
	}

	return alias, sourceColumn
}

// expandSelectAsterisk expands SELECT * into individual column results
func (e *querySpanExtractor) expandSelectAsterisk(keyspace, table string) []base.QuerySpanResult {
	if e.gCtx.GetDatabaseMetadataFunc == nil {
		// Test environment - fallback to SelectAsterisk flag
		return []base.QuerySpanResult{{
			Name:           "",
			SourceColumns:  base.SourceColumnSet{},
			SelectAsterisk: true,
		}}
	}

	if table == "" {
		// Cannot expand SELECT * without a table name
		return []base.QuerySpanResult{{
			Name:           "",
			SourceColumns:  base.SourceColumnSet{},
			SelectAsterisk: true,
		}}
	}

	ctx := context.Background()
	_, metadata, err := e.gCtx.GetDatabaseMetadataFunc(ctx, e.gCtx.InstanceID, keyspace)
	if err != nil || metadata == nil {
		// If we can't get metadata, fall back to SelectAsterisk flag
		// This matches behavior of other engines like TSQL
		return []base.QuerySpanResult{{
			Name:           "",
			SourceColumns:  base.SourceColumnSet{},
			SelectAsterisk: true,
		}}
	}

	// Find table and expand columns
	var results []base.QuerySpanResult
	schemaNames := metadata.ListSchemaNames()
	for _, schemaName := range schemaNames {
		schema := metadata.GetSchemaMetadata(schemaName)
		if schema == nil {
			continue
		}

		tbl := schema.GetTable(table)
		if tbl != nil {
			for _, col := range tbl.GetProto().GetColumns() {
				sourceColumn := base.SourceColumnSet{}
				sourceColumn[base.ColumnResource{
					Database: keyspace,
					Table:    table,
					Column:   col.GetName(),
				}] = true

				results = append(results, base.QuerySpanResult{
					Name:          col.GetName(),
					SourceColumns: sourceColumn,
					IsPlainField:  true,
				})
			}
			return results
		}
	}

	// Table not found - fall back to SelectAsterisk flag
	return []base.QuerySpanResult{{
		Name:           "",
		SourceColumns:  base.SourceColumnSet{},
		SelectAsterisk: true,
	}}
}

// extractWhereColumns extracts column references from WHERE clause
func (e *querySpanExtractor) extractWhereColumns(whereSpec cql.IWhereSpecContext, keyspace, table string) {
	if whereSpec == nil {
		return
	}

	// Extract relation elements from WHERE clause
	if relationElements := whereSpec.RelationElements(); relationElements != nil {
		// Process all relation elements (conditions connected by AND)
		for _, relationElement := range relationElements.AllRelationElement() {
			e.extractColumnsFromRelation(relationElement, keyspace, table)
		}
	}
}

// extractColumnsFromRelation extracts column references from a single relation element
func (e *querySpanExtractor) extractColumnsFromRelation(relation cql.IRelationElementContext, keyspace, table string) {
	if relation == nil {
		return
	}

	// Extract column names from OBJECT_NAME tokens
	// In CQL, columns are referenced as OBJECT_NAME in WHERE conditions
	for _, objName := range relation.AllOBJECT_NAME() {
		columnName := unquoteIdentifier(objName.GetText())

		colResource := base.ColumnResource{
			Database: keyspace,
			Table:    table,
			Column:   columnName,
		}

		// Add to PredicateColumns only (SourceColumns tracks tables, not columns)
		e.querySpan.PredicateColumns[colResource] = true
	}

	// TODO: Handle more complex cases:
	// - Functions containing columns (e.g., token(column))
	// - Collection operations (e.g., collection CONTAINS value)
	// - Nested expressions
}

// EnterInsert is called when we enter an INSERT statement
func (e *querySpanExtractor) EnterInsert(ctx *cql.InsertContext) {
	if e.err != nil {
		return
	}
	// Set query type to DML for INSERT
	e.querySpan.Type = base.DML

	// Extract table from INSERT INTO clause
	keyspace := e.defaultKeyspace
	table := ""

	if ctx.Keyspace() != nil {
		keyspace = unquoteIdentifier(ctx.Keyspace().GetText())
	}
	if ctx.Table() != nil {
		table = unquoteIdentifier(ctx.Table().GetText())
	}

	if table != "" {
		e.querySpan.SourceColumns[base.ColumnResource{
			Database: keyspace,
			Table:    table,
			Column:   "",
		}] = true
	}
}

// EnterUpdate is called when we enter an UPDATE statement
func (e *querySpanExtractor) EnterUpdate(ctx *cql.UpdateContext) {
	if e.err != nil {
		return
	}
	// Set query type to DML for UPDATE
	e.querySpan.Type = base.DML

	// Extract table from UPDATE clause
	keyspace := e.defaultKeyspace
	table := ""

	if ctx.Keyspace() != nil {
		keyspace = unquoteIdentifier(ctx.Keyspace().GetText())
	}
	if ctx.Table() != nil {
		table = unquoteIdentifier(ctx.Table().GetText())
	}

	if table != "" {
		e.querySpan.SourceColumns[base.ColumnResource{
			Database: keyspace,
			Table:    table,
			Column:   "",
		}] = true
	}

	// Extract WHERE clause columns for UPDATE
	if ctx.WhereSpec() != nil {
		e.extractWhereColumns(ctx.WhereSpec(), keyspace, table)
	}
}

// EnterDelete_ is called when we enter a DELETE statement
func (e *querySpanExtractor) EnterDelete_(ctx *cql.Delete_Context) {
	if e.err != nil {
		return
	}
	// Set query type to DML for DELETE
	e.querySpan.Type = base.DML

	// Extract table from DELETE FROM clause
	keyspace := e.defaultKeyspace
	table := ""
	if ctx.FromSpec() != nil {
		ks, t := e.extractTableFromFromSpec(ctx.FromSpec())
		if ks != "" {
			keyspace = ks
		}
		table = t
	}

	if table != "" {
		e.querySpan.SourceColumns[base.ColumnResource{
			Database: keyspace,
			Table:    table,
			Column:   "",
		}] = true
	}

	// Extract WHERE clause columns for DELETE
	if ctx.WhereSpec() != nil {
		e.extractWhereColumns(ctx.WhereSpec(), keyspace, table)
	}
}

// DDL Statement Handlers

// EnterCreateTable is called when we enter a CREATE TABLE statement
func (e *querySpanExtractor) EnterCreateTable(_ *cql.CreateTableContext) {
	if e.err != nil {
		return
	}
	e.querySpan.Type = base.DDL
}

// EnterAlterTable is called when we enter an ALTER TABLE statement
func (e *querySpanExtractor) EnterAlterTable(_ *cql.AlterTableContext) {
	if e.err != nil {
		return
	}
	e.querySpan.Type = base.DDL
}

// EnterDropTable is called when we enter a DROP TABLE statement
func (e *querySpanExtractor) EnterDropTable(_ *cql.DropTableContext) {
	if e.err != nil {
		return
	}
	e.querySpan.Type = base.DDL
}

// EnterCreateKeyspace is called when we enter a CREATE KEYSPACE statement
func (e *querySpanExtractor) EnterCreateKeyspace(_ *cql.CreateKeyspaceContext) {
	if e.err != nil {
		return
	}
	e.querySpan.Type = base.DDL
}

// EnterAlterKeyspace is called when we enter an ALTER KEYSPACE statement
func (e *querySpanExtractor) EnterAlterKeyspace(_ *cql.AlterKeyspaceContext) {
	if e.err != nil {
		return
	}
	e.querySpan.Type = base.DDL
}

// EnterDropKeyspace is called when we enter a DROP KEYSPACE statement
func (e *querySpanExtractor) EnterDropKeyspace(_ *cql.DropKeyspaceContext) {
	if e.err != nil {
		return
	}
	e.querySpan.Type = base.DDL
}

// EnterCreateIndex is called when we enter a CREATE INDEX statement
func (e *querySpanExtractor) EnterCreateIndex(_ *cql.CreateIndexContext) {
	if e.err != nil {
		return
	}
	e.querySpan.Type = base.DDL
}

// EnterDropIndex is called when we enter a DROP INDEX statement
func (e *querySpanExtractor) EnterDropIndex(_ *cql.DropIndexContext) {
	if e.err != nil {
		return
	}
	e.querySpan.Type = base.DDL
}

// EnterCreateMaterializedView is called when we enter a CREATE MATERIALIZED VIEW statement
func (e *querySpanExtractor) EnterCreateMaterializedView(_ *cql.CreateMaterializedViewContext) {
	if e.err != nil {
		return
	}
	e.querySpan.Type = base.DDL
}

// EnterAlterMaterializedView is called when we enter an ALTER MATERIALIZED VIEW statement
func (e *querySpanExtractor) EnterAlterMaterializedView(_ *cql.AlterMaterializedViewContext) {
	if e.err != nil {
		return
	}
	e.querySpan.Type = base.DDL
}

// EnterDropMaterializedView is called when we enter a DROP MATERIALIZED VIEW statement
func (e *querySpanExtractor) EnterDropMaterializedView(_ *cql.DropMaterializedViewContext) {
	if e.err != nil {
		return
	}
	e.querySpan.Type = base.DDL
}

// EnterCreateType is called when we enter a CREATE TYPE statement
func (e *querySpanExtractor) EnterCreateType(_ *cql.CreateTypeContext) {
	if e.err != nil {
		return
	}
	e.querySpan.Type = base.DDL
}

// EnterAlterType is called when we enter an ALTER TYPE statement
func (e *querySpanExtractor) EnterAlterType(_ *cql.AlterTypeContext) {
	if e.err != nil {
		return
	}
	e.querySpan.Type = base.DDL
}

// EnterDropType is called when we enter a DROP TYPE statement
func (e *querySpanExtractor) EnterDropType(_ *cql.DropTypeContext) {
	if e.err != nil {
		return
	}
	e.querySpan.Type = base.DDL
}

// EnterCreateFunction is called when we enter a CREATE FUNCTION statement
func (e *querySpanExtractor) EnterCreateFunction(_ *cql.CreateFunctionContext) {
	if e.err != nil {
		return
	}
	e.querySpan.Type = base.DDL
}

// EnterDropFunction is called when we enter a DROP FUNCTION statement
func (e *querySpanExtractor) EnterDropFunction(_ *cql.DropFunctionContext) {
	if e.err != nil {
		return
	}
	e.querySpan.Type = base.DDL
}

// EnterCreateTrigger is called when we enter a CREATE TRIGGER statement
func (e *querySpanExtractor) EnterCreateTrigger(_ *cql.CreateTriggerContext) {
	if e.err != nil {
		return
	}
	e.querySpan.Type = base.DDL
}

// EnterDropTrigger is called when we enter a DROP TRIGGER statement
func (e *querySpanExtractor) EnterDropTrigger(_ *cql.DropTriggerContext) {
	if e.err != nil {
		return
	}
	e.querySpan.Type = base.DDL
}
