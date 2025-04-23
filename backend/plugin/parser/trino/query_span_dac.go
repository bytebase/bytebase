// Package trino implements data access control features for the Trino parser.
package trino

import (
	"strings"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/trino-parser"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// processPredicateExpressions extracts predicate columns from boolean expressions.
// This function helps with identifying columns used in WHERE, JOIN, and other predicate clauses.
func (l *trinoQuerySpanListener) processPredicateExpressions(expr any) {
	if expr == nil {
		return
	}

	// Extract column names from the expression and mark them as predicate columns
	exprText := ""

	// Get text representation based on expression type
	switch typedExpr := expr.(type) {
	case parser.IBooleanExpressionContext:
		exprText = typedExpr.GetText()

	case parser.IExpressionContext:
		exprText = typedExpr.GetText()

	case parser.IPredicate_Context:
		exprText = typedExpr.GetText()

	default:
		// For other types, try to get text if available
		if textProvider, ok := expr.(interface{ GetText() string }); ok {
			exprText = textProvider.GetText()
		}
	}

	if exprText == "" {
		return
	}

	// Extract column references from the expression text
	l.extractPredicateColumnsFromText(exprText)
}

// extractPredicateColumnsFromText extracts column references from raw text
// This is a fallback method to ensure we don't miss columns in complex expressions
func (l *trinoQuerySpanListener) extractPredicateColumnsFromText(text string) {
	if text == "" {
		return
	}

	// Build a map of all known column names for efficient lookup
	columnsByName := make(map[string][]base.ColumnResource)
	for col := range l.extractor.sourceColumns {
		if col.Column != "" {
			columnsByName[col.Column] = append(columnsByName[col.Column], col)
		}
	}

	// Get predicates based on common SQL patterns
	for colName, cols := range columnsByName {
		// Skip very short column names to avoid false positives
		if len(colName) <= 1 {
			continue
		}

		// Check if the column appears in a predicate context
		if isPredicateColumn(text, colName) {
			// Mark all matching columns as predicates
			for _, col := range cols {
				l.extractor.addPredicateColumn(col)
			}
		}
	}
}

// isPredicateColumn determines if a column name appears in a predicate context
// A predicate context is one where the column is used in a filter condition
// such as WHERE, JOIN ON, or HAVING clauses.
func isPredicateColumn(text, columnName string) bool {
	// Quick check if the column name exists at all
	if !strings.Contains(text, columnName) {
		return false
	}

	// Convert text to lowercase for case-insensitive matching
	lowerText := strings.ToLower(text)
	lowerColumn := strings.ToLower(columnName)

	// 1. Check for qualified column references in JOIN/WHERE contexts
	// This handles table.column patterns in predicates
	qualifiedPatterns := []string{
		"." + lowerColumn + " =",
		"." + lowerColumn + "=",
		"." + lowerColumn + " <",
		"." + lowerColumn + "<",
		"." + lowerColumn + " >",
		"." + lowerColumn + ">",
		"." + lowerColumn + " in",
		"." + lowerColumn + " is ",
		"." + lowerColumn + " like",
		"." + lowerColumn + " between",
	}

	for _, pattern := range qualifiedPatterns {
		if strings.Contains(lowerText, pattern) {
			return true
		}
	}

	// 2. Check for column followed by comparison operators
	// This identifies patterns like "column = value"
	postfixPatterns := []string{
		lowerColumn + " =",
		lowerColumn + "=",
		lowerColumn + " <",
		lowerColumn + "<",
		lowerColumn + " >",
		lowerColumn + ">",
		lowerColumn + " <=",
		lowerColumn + "<=",
		lowerColumn + " >=",
		lowerColumn + ">=",
		lowerColumn + " <>",
		lowerColumn + "<>",
		lowerColumn + " !=",
		lowerColumn + "!=",
		lowerColumn + " in ",
		lowerColumn + " in(",
		lowerColumn + " like ",
		lowerColumn + " is ",
		lowerColumn + " between",
	}

	for _, pattern := range postfixPatterns {
		if strings.Contains(lowerText, pattern) {
			return true
		}
	}

	// 3. Check for column preceded by WHERE, AND, OR, etc.
	// This identifies patterns like "WHERE column" or "AND column"
	prefixPatterns := []string{
		"where " + lowerColumn,
		"and " + lowerColumn,
		"or " + lowerColumn,
		"on " + lowerColumn,
		"having " + lowerColumn,
		"join " + lowerColumn,
		"where(" + lowerColumn,
		"and(" + lowerColumn,
		"or(" + lowerColumn,
	}

	for _, pattern := range prefixPatterns {
		if strings.Contains(lowerText, pattern) {
			return true
		}
	}

	// 4. Check for column in list contexts typical in predicates
	// Such as IN clauses or multi-condition JOINs
	listPatterns := []string{
		"(" + lowerColumn + ")",
		"(" + lowerColumn + ",",
		"," + lowerColumn + ")",
		"in(" + lowerColumn,
		"," + lowerColumn + " ",
	}

	for _, pattern := range listPatterns {
		if strings.Contains(lowerText, pattern) {
			return true
		}
	}

	// 5. Check if the text is just the column name (handle simple expressions)
	if lowerText == lowerColumn {
		return true
	}

	// For very complex expressions that don't match our patterns,
	// we need deeper parsing. For now, return false.
	return false
}

// processSubquery extracts columns and predicates from subqueries.
// Used for WITH clauses, LATERAL, and subqueries in general.
func (l *trinoQuerySpanListener) processSubquery(query parser.IQueryContext) {
	if query == nil {
		return
	}

	// Create a new extractor for the subquery
	subExtractor := newQuerySpanExtractor(
		l.extractor.defaultDatabase,
		l.extractor.defaultSchema,
		l.extractor.gCtx,
		l.extractor.ignoreCaseSensitive,
	)

	// Copy existing CTEs to maintain context
	subExtractor.ctes = append(subExtractor.ctes, l.extractor.ctes...)

	// Process the subquery with a new listener
	subListener := newTrinoQuerySpanListener(subExtractor)

	// Use the ANTLR tree walker to process the query
	antlr.ParseTreeWalkerDefault.Walk(subListener, query)

	// Merge source columns from subquery
	for col := range subListener.extractor.sourceColumns {
		l.extractor.sourceColumns[col] = true
	}

	// Merge predicate columns from subquery
	for col := range subListener.extractor.predicateColumns {
		l.extractor.predicateColumns[col] = true
	}
}

// processPredicateJoin extracts predicate columns from JOIN conditions.
func (l *trinoQuerySpanListener) processPredicateJoin(ctx parser.IJoinCriteriaContext) {
	if ctx == nil {
		return
	}

	// Process ON clause predicates
	if ctx.ON_() != nil && ctx.BooleanExpression() != nil {
		l.processPredicateExpressions(ctx.BooleanExpression())
	}

	// Process USING clause
	if ctx.USING_() != nil {
		for _, ident := range ctx.AllIdentifier() {
			columnName := NormalizeTrinoIdentifier(ident.GetText())
			// Find matching source columns
			for col := range l.extractor.sourceColumns {
				if col.Column == columnName {
					l.extractor.addPredicateColumn(col)
				}
			}
		}
	}
}
