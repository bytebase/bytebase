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
	
	// Extract table information from FROM clause
	keyspace, table := e.extractTableFromFromSpec(ctx.FromSpec())
	if keyspace == "" {
		keyspace = e.defaultKeyspace
	}
	
	// Extract column information from SELECT clause
	results := e.extractSelectElements(ctx.SelectElements(), keyspace, table)
	e.querySpan.Results = results
	
	// Extract columns from WHERE clause for predicate tracking
	if whereSpec := ctx.WhereSpec(); whereSpec != nil {
		e.extractWhereColumns(whereSpec, keyspace, table)
	}
}

// extractTableFromFromSpec extracts keyspace and table name from FROM clause
func (e *querySpanExtractor) extractTableFromFromSpec(fromSpec cql.IFromSpecContext) (keyspace, table string) {
	if fromSpec == nil {
		return "", ""
	}
	
	if from, ok := fromSpec.(*cql.FromSpecContext); ok {
		if fromElem := from.FromSpecElement(); fromElem != nil {
			// FromSpecElement can be either "table" or "keyspace.table"
			text := fromElem.GetText()
			parts := strings.Split(text, ".")
			if len(parts) == 2 {
				return parts[0], parts[1]
			}
			return "", parts[0]
		}
	}
	return "", ""
}

// extractSelectElements extracts column information from SELECT clause
func (e *querySpanExtractor) extractSelectElements(selectElements cql.ISelectElementsContext, keyspace, table string) []base.QuerySpanResult {
	if selectElements == nil {
		return nil
	}
	
	if sel, ok := selectElements.(*cql.SelectElementsContext); ok {
		// Check for SELECT *
		if sel.GetStar() != nil {
			return e.expandSelectAsterisk(keyspace, table)
		}
		
		// Handle specific column selections
		var results []base.QuerySpanResult
		for _, elem := range sel.AllSelectElement() {
			if selElem, ok := elem.(*cql.SelectElementContext); ok {
				columnName := e.extractColumnName(selElem)
				if columnName != "" {
					// Create source column reference
					sourceColumn := base.SourceColumnSet{}
					sourceColumn[base.ColumnResource{
						Database: keyspace,
						Table:    table,
						Column:   columnName,
					}] = true
					
					results = append(results, base.QuerySpanResult{
						Name:          columnName,
						SourceColumns: sourceColumn,
						IsPlainField:  true,
					})
				}
			}
		}
		return results
	}
	
	return nil
}

// extractColumnName extracts the column name from a select element
func (e *querySpanExtractor) extractColumnName(elem *cql.SelectElementContext) string {
	if elem == nil {
		return ""
	}
	
	// Get the text of the select element
	// This could be a simple column name or table.column
	text := elem.GetText()
	
	// Handle table.column notation
	parts := strings.Split(text, ".")
	if len(parts) == 2 {
		return parts[1]
	}
	
	// Handle AS alias
	if elem.GetChildCount() > 2 {
		// Check if there's an AS keyword
		for i := 0; i < elem.GetChildCount(); i++ {
			if child := elem.GetChild(i); child != nil {
				if strings.ToUpper(child.(antlr.ParseTree).GetText()) == "AS" && i+1 < elem.GetChildCount() {
					// Return the alias
					return elem.GetChild(i + 1).(antlr.ParseTree).GetText()
				}
			}
		}
	}
	
	return parts[0]
}

// expandSelectAsterisk expands SELECT * into individual column results
func (e *querySpanExtractor) expandSelectAsterisk(keyspace, table string) []base.QuerySpanResult {
	if e.gCtx.GetDatabaseMetadataFunc == nil {
		// Test environment
		return []base.QuerySpanResult{{
			Name:           "",
			SourceColumns:  base.SourceColumnSet{},
			SelectAsterisk: true,
		}}
	}
	
	ctx := context.Background()
	_, metadata, err := e.gCtx.GetDatabaseMetadataFunc(ctx, e.gCtx.InstanceID, keyspace)
	if err != nil || metadata == nil {
		return []base.QuerySpanResult{}
	}
	
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
			break
		}
	}
	
	return results
}

// extractWhereColumns extracts column references from WHERE clause
func (e *querySpanExtractor) extractWhereColumns(whereSpec cql.IWhereSpecContext, keyspace, table string) {
	if whereSpec == nil {
		return
	}
	
	// For now, we'll mark that we found columns in WHERE clause
	// A full implementation would parse the relation elements
	// to extract all column references
	
	// This is a simplified implementation - a complete one would
	// walk through all relation elements and extract column names
}