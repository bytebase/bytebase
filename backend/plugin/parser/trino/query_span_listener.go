// Package trino provides SQL parser for Trino.
package trino

import (
	"fmt"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/trino"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// trinoQuerySpanListener walks the parse tree to extract query span information.
type trinoQuerySpanListener struct {
	parser.BaseTrinoParserListener

	extractor *querySpanExtractor
	results   []base.QuerySpanResult
	err       error

	// Current CTE being processed
	currentCTE *base.PseudoTable
}

// newTrinoQuerySpanListener creates a new listener with the given extractor.
func newTrinoQuerySpanListener(extractor *querySpanExtractor) *trinoQuerySpanListener {
	return &trinoQuerySpanListener{
		extractor: extractor,
		results:   []base.QuerySpanResult{},
	}
}

// EnterQuery processes the top level query and WITH clauses if present.
func (l *trinoQuerySpanListener) EnterQuery(ctx *parser.QueryContext) {
	if l.err != nil {
		return
	}

	if ctx.With() != nil {
		// Process WITH clause
		withCtx := ctx.With()

		// Process each named query in the WITH clause
		for _, namedQueryCtx := range withCtx.AllNamedQuery() {
			if namedQueryCtx.Identifier() == nil || namedQueryCtx.Query() == nil {
				continue
			}

			// Get the CTE name
			cteName := NormalizeTrinoIdentifier(namedQueryCtx.Identifier().GetText())

			// Create a new pseudo table for this CTE
			l.currentCTE = base.NewPseudoTable(cteName, nil)

			// Process the CTE query
			queryTree := namedQueryCtx.Query()
			if queryTree != nil {
				// Process subquery to extract columns
				if err := l.extractor.extractPredicateColumnFromSubquery(queryTree); err != nil {
					l.err = err
					return
				}
			}

			// Add the CTE to our list
			l.extractor.ctes = append(l.extractor.ctes, l.currentCTE)
			l.currentCTE = nil
		}
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

	// Check if this is a CTE reference
	for _, cte := range l.extractor.ctes {
		if strings.EqualFold(cte.Name, table) {
			// This is a CTE reference
			l.extractor.tableSourcesFrom = append(l.extractor.tableSourcesFrom, cte)
			return
		}
	}

	// Add table source to track where columns come from
	// Attempt to find the table in the schema first
	tableMeta, err := l.extractor.findTableSchema(db, schema, table)
	if err != nil {
		// We don't set l.err here because table references might be CTEs or subqueries

		// Create physical table without columns as fallback
		physicalTable := &base.PhysicalTable{
			Database: db,
			Schema:   schema,
			Name:     table,
			Columns:  []string{}, // Empty columns
		}
		l.extractor.tableSourcesFrom = append(l.extractor.tableSourcesFrom, physicalTable)
		return
	}

	// Create physical table with column names from metadata
	var columnNames []string
	for _, col := range tableMeta.GetProto().GetColumns() {
		columnNames = append(columnNames, col.Name)
	}

	physicalTable := &base.PhysicalTable{
		Database: db,
		Schema:   schema,
		Name:     tableMeta.GetProto().Name,
		Columns:  columnNames, // Populate with actual column names
	}
	l.extractor.tableSourcesFrom = append(l.extractor.tableSourcesFrom, physicalTable)

	// Add each column from the table as a source column
	for _, col := range tableMeta.GetProto().GetColumns() {
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

	// Mark that we encountered a SELECT * - we'll expand it later after all tables are processed
	result := base.QuerySpanResult{
		Name:           "*",
		SourceColumns:  make(base.SourceColumnSet), // Will be populated later
		IsPlainField:   false,
		SelectAsterisk: true,
	}

	l.results = append(l.results, result)
}

// EnterSelectSingle processes individual SELECT items.
func (l *trinoQuerySpanListener) EnterSelectSingle(ctx *parser.SelectSingleContext) {
	if l.err != nil {
		return
	}

	var resultName string
	var sourceColumns = make(base.SourceColumnSet)
	isPlainField := false

	if ctx.Expression() != nil {
		expr := ctx.Expression()

		// Check for simple column references
		columnName := l.extractColumnName(expr)
		if columnName != "" {
			isPlainField = true
			// Find matching source columns
			for col := range l.extractor.sourceColumns {
				if col.Column == columnName {
					sourceColumns[col] = true
				}
			}
		} else {
			// For complex expressions, extract all column references
			l.extractAllColumnReferences(expr, sourceColumns)
		}

		resultName = expr.GetText()
	}

	// Override with alias if provided
	if ctx.As_column_alias() != nil && ctx.As_column_alias().Column_alias() != nil && ctx.As_column_alias().Column_alias().Identifier() != nil {
		resultName = NormalizeTrinoIdentifier(ctx.As_column_alias().Column_alias().Identifier().GetText())
	}

	// Generate name if none found
	if resultName == "" {
		resultName = fmt.Sprintf("_col%d", len(l.results))
	}

	result := base.QuerySpanResult{
		Name:          resultName,
		SourceColumns: sourceColumns,
		IsPlainField:  isPlainField,
	}

	l.results = append(l.results, result)
}

// extractAllColumnReferences gathers all column references from a complex expression.
// This uses ANTLR tree traversal instead of regex patterns for better accuracy.
func (l *trinoQuerySpanListener) extractAllColumnReferences(expr parser.IExpressionContext, sourceColumns base.SourceColumnSet) {
	if expr == nil {
		return
	}

	// Use ANTLR tree traversal to find column references
	if err := l.extractor.extractPredicateColumnFromExpression(expr); err != nil {
		return // Best effort - don't fail on complex expressions
	}

	// Copy predicate columns found to result source columns
	for col := range l.extractor.predicateColumns {
		sourceColumns[col] = true
	}
}

// EnterQuerySpecification processes query specifications, including WHERE clauses
func (l *trinoQuerySpanListener) EnterQuerySpecification(ctx *parser.QuerySpecificationContext) {
	if l.err != nil {
		return
	}

	// Process WHERE clause using consistent predicate extraction
	if ctx.GetWhere() != nil {
		if err := l.extractor.extractPredicateColumnFromBooleanExpression(ctx.GetWhere()); err != nil {
			l.err = err
		}
	}
}

// EnterJoinCriteria processes JOIN conditions to extract predicate columns.
func (l *trinoQuerySpanListener) EnterJoinCriteria(ctx *parser.JoinCriteriaContext) {
	if l.err != nil {
		return
	}

	// Use consistent predicate extraction following TSQL pattern
	if err := l.extractor.processJoinPredicate(ctx); err != nil {
		l.err = err
	}
}

// EnterUnnest processes UNNEST expressions, a Trino-specific feature.
func (l *trinoQuerySpanListener) EnterUnnest(ctx *parser.UnnestContext) {
	if l.err != nil {
		return
	}

	// Process the expressions being unnested
	for _, expr := range ctx.AllExpression() {
		// Extract column references from the UNNEST expression
		columnName := l.extractColumnName(expr)
		if columnName != "" {
			// Add as source column since UNNEST creates a derived table
			for col := range l.extractor.sourceColumns {
				if col.Column == columnName {
					l.extractor.sourceColumns[col] = true
				}
			}
		}
	}

	// Create a derived table for the UNNEST
	results := []base.QuerySpanResult{}
	for i, expr := range ctx.AllExpression() {
		// In UNNEST, each column would have a name based on its position
		name := fmt.Sprintf("_unnest%d", i)

		// Try to extract source column information
		colSourceColumns := make(base.SourceColumnSet)
		columnName := l.extractColumnName(expr)
		if columnName != "" {
			for col := range l.extractor.sourceColumns {
				if col.Column == columnName {
					colSourceColumns[col] = true
				}
			}
		}

		results = append(results, base.QuerySpanResult{
			Name:          name,
			SourceColumns: colSourceColumns,
			IsPlainField:  true,
		})
	}

	// Register as a derived table source
	unnestTable := base.NewPseudoTable("unnest", results)
	l.extractor.tableSourcesFrom = append(l.extractor.tableSourcesFrom, unnestTable)
}

// EnterLateral processes LATERAL subqueries, a Trino-specific feature.
func (l *trinoQuerySpanListener) EnterLateral(ctx *parser.LateralContext) {
	if l.err != nil {
		return
	}

	// Save outer table sources to allow resolving columns in lateral query
	originalOuterTables := l.extractor.outerTableSources

	// For a LATERAL subquery, columns from tables to the left are visible inside
	l.extractor.outerTableSources = append(l.extractor.outerTableSources, l.extractor.tableSourcesFrom...)

	// Process the subquery
	if ctx.Query() != nil {
		// Create a new extractor for the subquery
		subExtractor := newQuerySpanExtractor(
			l.extractor.defaultDatabase,
			l.extractor.defaultSchema,
			l.extractor.gCtx,
			l.extractor.ignoreCaseSensitive,
		)

		// Copy outer table sources
		subExtractor.outerTableSources = append(subExtractor.outerTableSources, l.extractor.outerTableSources...)

		// Copy existing CTEs
		subExtractor.ctes = append(subExtractor.ctes, l.extractor.ctes...)

		// Process the subquery with a new listener
		subListener := newTrinoQuerySpanListener(subExtractor)
		antlr.ParseTreeWalkerDefault.Walk(subListener, ctx.Query())

		// Create a derived table for the lateral subquery
		lateralTable := base.NewPseudoTable("lateral", subListener.results)

		// Add to our table sources
		l.extractor.tableSourcesFrom = append(l.extractor.tableSourcesFrom, lateralTable)

		// Add columns from the lateral subquery to our source columns
		for _, result := range subListener.results {
			for col := range result.SourceColumns {
				l.extractor.sourceColumns[col] = true
			}
		}

		// Merge predicates from subquery
		for col := range subListener.extractor.predicateColumns {
			l.extractor.predicateColumns[col] = true
		}
	}

	// Restore outer table sources
	l.extractor.outerTableSources = originalOuterTables
}

// EnterQueryNoWith processes queries, including those with set operations (UNION, INTERSECT, EXCEPT)
func (l *trinoQuerySpanListener) EnterQueryNoWith(_ *parser.QueryNoWithContext) {
	if l.err != nil {
		return
	}

	// This method is used to extract information from a query
	// The actual parsing work is done by ANTLR, which will call the
	// appropriate Enter* methods for each node in the parse tree

	// We don't need special handling here - the listener will automatically
	// process all parts of the query, including both sides of set operations

	// The main logic for predicate column extraction is in processPredicateExpressions
	// and other visitor methods
}

// addPredicateColumn adds a column to the predicate columns list.
func (l *trinoQuerySpanListener) addPredicateColumn(columnName string) {
	if columnName == "" {
		return
	}

	// Find matching source columns and add them to predicates
	for col := range l.extractor.sourceColumns {
		if col.Column == columnName {
			l.extractor.predicateColumns[col] = true
		}
	}
}

// extractColumnName attempts to extract a column reference from an expression.
// Returns the column name if the expression is a simple column reference, or empty string otherwise.
func (l *trinoQuerySpanListener) extractColumnName(expr parser.IExpressionContext) string {
	if expr == nil {
		return ""
	}

	exprText := expr.GetText()
	if exprText == "" {
		return ""
	}

	// Handle qualified column names (e.g., table.column)
	parts := strings.Split(exprText, ".")
	if len(parts) > 1 {
		columnName := parts[len(parts)-1]
		// Check if this matches a known column
		for col := range l.extractor.sourceColumns {
			if col.Column == columnName {
				return columnName
			}
		}
	}

	// Check if the entire expression is a column name
	for col := range l.extractor.sourceColumns {
		if col.Column == exprText {
			return exprText
		}
	}

	return ""
}
