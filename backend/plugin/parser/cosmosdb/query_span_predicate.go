package cosmosdb

import (
	"log/slog"

	parser "github.com/bytebase/parser/cosmosdb"

	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

type querySpanPredicatePathsListener struct {
	*parser.BaseCosmosDBParserListener

	predicatePaths map[string]*base.PathAST
	err            error
}

func (l *querySpanPredicatePathsListener) EnterSelect(ctx *parser.SelectContext) {
	whereClause := ctx.Where_clause()
	if whereClause == nil {
		return
	}
	fromClause := ctx.From_clause()
	if fromClause == nil {
		return
	}

	originalContainerName, fromIdentifier := extractFromNames(fromClause)
	l.predicatePaths = buildPredicatePaths(whereClause.Scalar_expression(), originalContainerName, fromIdentifier)
}

func buildPredicatePaths(expr parser.IScalar_expressionContext, containerName, fromAlias string) map[string]*base.PathAST {
	paths := extractPredicateFields(expr, containerName, fromAlias)
	r := make(map[string]*base.PathAST)
	for _, path := range paths {
		if len(path) == 0 {
			continue
		}
		ast := buildPathAST(path)
		str, err := ast.String()
		if err != nil {
			slog.Warn("failed to convert path ast to string", log.BBError(err))
		}
		r[str] = ast
	}
	return r
}

// extractPredicateFields collects all field paths referenced in a scalar expression used as a predicate.
func extractPredicateFields(ctx parser.IScalar_expressionContext, containerName, fromAlias string) [][]base.SelectorNode {
	if ctx == nil {
		return nil
	}

	// Input alias (leaf field reference).
	if ctx.Input_alias() != nil {
		return resolveInputAlias(ctx, containerName, fromAlias)
	}
	// Property access: a.b.c
	if ctx.DOT_SYMBOL() != nil {
		return extractPredicateDotAccess(ctx, containerName, fromAlias)
	}
	// Bracket access: a["b"] or a[0] (but not IN expression which also uses brackets).
	if ctx.LS_BRACKET_SYMBOL() != nil && ctx.IN_SYMBOL() == nil {
		return extractPredicateBracketAccess(ctx, containerName, fromAlias)
	}
	// Unary operator or standalone NOT.
	if ctx.Unary_operator() != nil || (ctx.NOT_SYMBOL() != nil && len(ctx.AllScalar_expression()) == 1) {
		return extractPredicateFields(ctx.Scalar_expression(0), containerName, fromAlias)
	}
	// Parenthesized expression (not subquery, not EXISTS).
	if ctx.LR_BRACKET_SYMBOL() != nil && ctx.Select_() == nil && ctx.EXISTS_SYMBOL() == nil {
		return extractPredicateFields(ctx.Scalar_expression(0), containerName, fromAlias)
	}
	// Function calls.
	if ctx.Scalar_function_expression() != nil {
		return extractPredicateFieldsFromFunction(ctx.Scalar_function_expression(), containerName, fromAlias)
	}
	// Ternary operator: cond ? a : b
	if ctx.QUESTION_MARK_SYMBOL() != nil {
		return collectPredicateFieldsFromAll(ctx.AllScalar_expression(), containerName, fromAlias)
	}
	// Binary operators (comparison, arithmetic, bitwise, logical, IN, BETWEEN, LIKE, concat).
	// All share the same logic: collect fields from all child expressions.
	if isBinaryOrMultiChildExpression(ctx) {
		return collectPredicateFieldsFromAll(ctx.AllScalar_expression(), containerName, fromAlias)
	}
	return nil
}

func extractPredicateDotAccess(ctx parser.IScalar_expressionContext, containerName, fromAlias string) [][]base.SelectorNode {
	paths := extractPredicateFields(ctx.Scalar_expression(0), containerName, fromAlias)
	propName := ctx.Property_name().Identifier().GetText()
	for i := range paths {
		paths[i] = append(paths[i], base.NewItemSelector(propName))
	}
	return paths
}

func extractPredicateBracketAccess(ctx parser.IScalar_expressionContext, containerName, fromAlias string) [][]base.SelectorNode {
	paths := extractPredicateFields(ctx.Scalar_expression(0), containerName, fromAlias)
	for i := range paths {
		appendBracketSelector(ctx, paths, i)
	}
	return paths
}

func extractPredicateFieldsFromFunction(ctx parser.IScalar_function_expressionContext, containerName, fromAlias string) [][]base.SelectorNode {
	var exprs []parser.IScalar_expressionContext
	switch {
	case ctx.Udf_scalar_function_expression() != nil:
		exprs = ctx.Udf_scalar_function_expression().AllScalar_expression()
	case ctx.Builtin_function_expression() != nil:
		exprs = ctx.Builtin_function_expression().AllScalar_expression()
	default:
		return nil
	}
	return collectPredicateFieldsFromAll(exprs, containerName, fromAlias)
}

// isBinaryOrMultiChildExpression returns true for binary operators and multi-child expressions
// that all share the same extraction logic: collect fields from all child scalar expressions.
func isBinaryOrMultiChildExpression(ctx parser.IScalar_expressionContext) bool {
	return ctx.Comparison_operator() != nil ||
		ctx.Multiplicative_operator() != nil ||
		ctx.Additive_operator() != nil ||
		ctx.Shift_operator() != nil ||
		ctx.BIT_AND_SYMBOL() != nil ||
		ctx.BIT_XOR_SYMBOL() != nil ||
		ctx.BIT_OR_SYMBOL() != nil ||
		ctx.DOUBLE_BAR_SYMBOL() != nil ||
		ctx.LIKE_SYMBOL() != nil ||
		ctx.IN_SYMBOL() != nil ||
		ctx.BETWEEN_SYMBOL() != nil ||
		// AND/OR logical operators. BETWEEN also sets AND_SYMBOL, but BETWEEN is checked above.
		ctx.AND_SYMBOL() != nil ||
		ctx.OR_SYMBOL() != nil
}

func collectPredicateFieldsFromAll(exprs []parser.IScalar_expressionContext, containerName, fromAlias string) [][]base.SelectorNode {
	var all [][]base.SelectorNode
	for _, expr := range exprs {
		all = append(all, extractPredicateFields(expr, containerName, fromAlias)...)
	}
	return all
}
