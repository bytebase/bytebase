package trino

import (
	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/trino"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// extractPredicateColumnFromBooleanExpression extracts predicate columns from boolean expressions.
// This follows the TSQL pattern for consistent predicate extraction.
func (q *querySpanExtractor) extractPredicateColumnFromBooleanExpression(
	ctx parser.IBooleanExpressionContext,
) error {
	if ctx == nil {
		return nil
	}

	// Use generic tree walking to find all expressions
	return q.extractPredicateColumnFromExpression(ctx)
}

// extractPredicateColumnFromExpression recursively walks the expression tree to find column references.
// This is the core method following TSQL's recursive pattern but adapted for Trino.
func (q *querySpanExtractor) extractPredicateColumnFromExpression(
	ctx antlr.ParserRuleContext,
) error {
	if ctx == nil {
		return nil
	}

	switch typedCtx := ctx.(type) {
	case parser.IQualifiedNameContext:
		// Qualified name that might be a column reference
		col := q.extractColumnFromQualifiedName(typedCtx)
		if col != nil {
			q.predicateColumns[*col] = true
		}

	case parser.IQueryContext:
		// Subquery in expression
		if err := q.extractPredicateColumnFromSubquery(typedCtx); err != nil {
			return err
		}
	}

	// Recursively process all child nodes following TSQL pattern
	for _, child := range ctx.GetChildren() {
		if childCtx, ok := child.(antlr.ParserRuleContext); ok {
			if err := q.extractPredicateColumnFromExpression(childCtx); err != nil {
				return err
			}
		}
	}

	return nil
}

// extractPredicateColumnFromSubquery processes subqueries within predicates.
// This follows the TSQL pattern for subquery handling.
func (q *querySpanExtractor) extractPredicateColumnFromSubquery(
	ctx parser.IQueryContext,
) error {
	if ctx == nil {
		return nil
	}

	// Get subquery text for processing
	subquery := ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx)

	// Create new extractor for subquery, following TSQL pattern
	newExtractor := newQuerySpanExtractor(q.defaultDatabase, q.defaultSchema, q.gCtx, q.ignoreCaseSensitive)
	newExtractor.ctes = q.ctes // Preserve CTE context

	span, err := newExtractor.getQuerySpan(q.ctx, subquery)
	if err != nil {
		return errors.Wrapf(err, "failed to get query span for subquery: %s", subquery)
	}

	// Merge predicate columns from subquery
	q.predicateColumns, _ = base.MergeSourceColumnSet(q.predicateColumns, span.PredicateColumns)

	// Add result columns as predicates (they're used in the outer predicate context)
	for _, result := range span.Results {
		q.predicateColumns, _ = base.MergeSourceColumnSet(q.predicateColumns, result.SourceColumns)
	}

	return nil
}

// extractColumnFromQualifiedName extracts a column resource from a qualified name context.
func (q *querySpanExtractor) extractColumnFromQualifiedName(ctx parser.IQualifiedNameContext) *base.ColumnResource {
	if ctx == nil {
		return nil
	}

	db, schema, table := ExtractDatabaseSchemaName(ctx, q.defaultDatabase, q.defaultSchema)

	// If we have table qualification, it's likely a column reference
	if table != "" {
		parts := ctx.AllIdentifier()
		if len(parts) > 0 {
			columnName := NormalizeTrinoIdentifier(parts[len(parts)-1].GetText())

			// Try to match with source columns
			for col := range q.sourceColumns {
				if col.Column == columnName {
					// Check table match if specified
					if col.Table == "" || col.Table == table {
						return &col
					}
				}
			}

			// Create column resource if not found in source columns
			return &base.ColumnResource{
				Database: db,
				Schema:   schema,
				Table:    table,
				Column:   columnName,
			}
		}
	}

	return nil
}

// processJoinPredicate extracts predicate columns from JOIN conditions.
// This follows the TSQL pattern for join predicate handling.
func (q *querySpanExtractor) processJoinPredicate(ctx parser.IJoinCriteriaContext) error {
	if ctx == nil {
		return nil
	}

	// Process ON clause predicates
	if ctx.ON_() != nil && ctx.BooleanExpression() != nil {
		if err := q.extractPredicateColumnFromBooleanExpression(ctx.BooleanExpression()); err != nil {
			return err
		}
	}

	// Process USING clause - columns in USING are predicates by definition
	if ctx.USING_() != nil {
		for _, ident := range ctx.AllIdentifier() {
			columnName := NormalizeTrinoIdentifier(ident.GetText())
			// Find matching source columns and add as predicates
			for col := range q.sourceColumns {
				if col.Column == columnName {
					q.predicateColumns[col] = true
				}
			}
		}
	}

	return nil
}
