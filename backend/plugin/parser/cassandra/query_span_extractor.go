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

	keyspace, table := e.extractTableFromFromSpec(ctx.FromSpec())
	if keyspace == "" {
		keyspace = e.defaultKeyspace
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
		schema := metadata.GetSchema(schemaName)
		if schema == nil {
			continue
		}

		tbl := schema.GetTable(table)
		if tbl != nil {
			for _, col := range tbl.GetColumns() {
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
func (*querySpanExtractor) extractWhereColumns(whereSpec cql.IWhereSpecContext, _, _ string) {
	if whereSpec == nil {
		return
	}

	// For now, we'll mark that we found columns in WHERE clause
	// A full implementation would parse the relation elements
	// to extract all column references

	// This is a simplified implementation - a complete one would
	// walk through all relation elements and extract column names
}
