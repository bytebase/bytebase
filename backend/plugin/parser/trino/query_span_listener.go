package trino

import (
	parser "github.com/bytebase/trino-parser"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// trinoQuerySpanListener walks the parse tree to extract query span information.
type trinoQuerySpanListener struct {
	parser.BaseTrinoParserListener

	extractor *querySpanExtractor
	results   []base.QuerySpanResult
	err       error
}

// newTrinoQuerySpanListener creates a new listener with the given extractor.
func newTrinoQuerySpanListener(extractor *querySpanExtractor) *trinoQuerySpanListener {
	return &trinoQuerySpanListener{
		extractor: extractor,
		results:   []base.QuerySpanResult{},
	}
}

// EnterTableName processes table references in the query.
func (l *trinoQuerySpanListener) EnterTableName(ctx *parser.TableNameContext) {
	if l.err != nil {
		return
	}

	if ctx.QualifiedName() == nil {
		return
	}

	// Extract database, schema, and table name
	db, schema, table := ExtractDatabaseSchemaName(
		ctx.QualifiedName(),
		l.extractor.defaultDatabase,
		l.extractor.defaultSchema,
	)

	// Attempt to find the table in the schema
	tableMeta, err := l.extractor.findTableSchema(db, schema, table)
	if err != nil {
		// We don't set l.err here because table references might be CTEs or subqueries
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
	if l.err != nil {
		return
	}

	// Get column name and alias
	var resultName string
	var sourceColumns base.SourceColumnSet
	isPlainField := false

	if ctx.Expression() != nil {
		// For now, just use the expression text as the result name
		resultName = ctx.Expression().GetText()
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
