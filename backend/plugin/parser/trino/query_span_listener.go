package trino

import (
	parser "github.com/bytebase/trino-parser"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// trinoQuerySpanListener walks the parse tree to extract query span information.
type trinoQuerySpanListener struct {
	*parser.BaseTrinoParserListener

	extractor *querySpanExtractor
	results   []base.QuerySpanResult
	err       error
}

// newTrinoQuerySpanListener creates a new listener with the given extractor.
func newTrinoQuerySpanListener(extractor *querySpanExtractor) *trinoQuerySpanListener {
	return &trinoQuerySpanListener{
		BaseTrinoParserListener: &parser.BaseTrinoParserListener{},
		extractor:               extractor,
		results:                 []base.QuerySpanResult{},
	}
}

// EnterTableName processes table references in the query.
func (l *trinoQuerySpanListener) EnterTableName(ctx *parser.TableNameContext) {
	if l.err != nil {
		return // Skip if already encountered an error
	}

	// Extract database, schema, and table name from the qualified name
	if ctx.QualifiedName() == nil {
		return
	}

	db, schema, table := ExtractDatabaseSchemaName(
		ctx.QualifiedName(),
		l.extractor.defaultDatabase,
		l.extractor.defaultSchema,
	)

	// Attempt to find the table in the schema
	tableMeta, err := l.extractor.findTableSchema(db, schema, table)
	if err != nil {
		// We don't set l.err here because table references might be CTEs or subqueries
		// that won't be found via metadata lookup
		return
	}

	// Add each column from the table as a source column
	for _, col := range tableMeta.GetColumns() {
		colResource := base.ColumnResource{
			Database: db,
			Schema:   schema,
			Table:    table,
			Column:   col.Name,
		}
		l.extractor.addSourceColumn(colResource)
	}
}

// EnterSelectItem processes SELECT items in the query.
func (l *trinoQuerySpanListener) EnterSelectItem(_ *parser.SelectItemContext) {
	if l.err != nil {
		return
	}

	// The SelectItemContext should be either SelectAll or SelectSingle
	// Let the appropriate Enter* method handle the details

	// The specific methods for child rules will be called automatically
	// EnterSelectSingle or EnterSelectAll
}

// EnterSelectAll handles all cases of SELECT * expressions.
func (l *trinoQuerySpanListener) EnterSelectAll(_ *parser.SelectAllContext) {
	if l.err != nil {
		return
	}

	// For SELECT *, we add a generic result entry
	result := base.QuerySpanResult{
		Name:          "*",
		SourceColumns: l.extractor.sourceColumns,
		IsPlainField:  false,
	}

	l.results = append(l.results, result)
}

// EnterSelectSingle processes individual SELECT items.
func (l *trinoQuerySpanListener) EnterSelectSingle(ctx *parser.SelectSingleContext) {
	// Get column name and alias
	var resultName string
	var sourceColumns base.SourceColumnSet
	isPlainField := false

	if ctx.Expression() != nil {
		// For now, just use the expression text regardless of expression type
		resultName = ctx.Expression().GetText()

		// In a more complete implementation, we would handle different expression types differently
		// and extract column references from the expression
	}

	// Override with alias if provided
	if ctx.Identifier() != nil {
		resultName = NormalizeTrinoIdentifier(ctx.Identifier().GetText())
	}

	// If still no name, use a placeholder
	if resultName == "" {
		resultName = "_col" + string(rune('0'+len(l.results)))
	}

	// Create a result entry for this SELECT item
	result := base.QuerySpanResult{
		Name:          resultName,
		SourceColumns: sourceColumns,
		IsPlainField:  isPlainField,
	}

	l.results = append(l.results, result)
}
