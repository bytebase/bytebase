package cosmosdb

import (
	"log/slog"
	"strconv"

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
	// Extracting predicate fields from where clause.
	whereClause := ctx.Where_clause()
	if whereClause == nil {
		return
	}
	fromClause := ctx.From_clause()
	if fromClause == nil {
		return
	}

	var originalContainerName string
	var fromIdentifier string
	if i := fromClause.From_specification().From_source().Container_expression().Container_name().Identifier(); i != nil {
		originalContainerName = i.GetText()
	}
	// Alias in the from source will shadow the original identifier.
	if i := fromClause.From_specification().From_source().Container_expression().Identifier(); i != nil {
		fromIdentifier = i.GetText()
	}

	predicateFields := extractPredicateFieldsFromWhereClause(whereClause, originalContainerName, fromIdentifier)
	l.predicatePaths = predicateFields
}

func extractPredicateFieldsFromWhereClause(ctx parser.IWhere_clauseContext, originalContainerName string, fromAlias string) map[string]*base.PathAST {
	scalarExpression := ctx.Scalar_expression()

	paths := extractPredicateFieldsFromScalarExpression(scalarExpression, originalContainerName, fromAlias)

	r := make(map[string]*base.PathAST)
	for _, path := range paths {
		if len(path) == 0 {
			continue
		}

		ast := base.NewPathAST(path[0])
		current := ast.Root
		for i := 1; i < len(path); i++ {
			current.SetNext(path[i])
			current = current.GetNext()
		}

		str, err := ast.String()
		if err != nil {
			slog.Warn("failed to convert path ast to string", log.BBError(err))
		}
		r[str] = ast
	}

	return r
}

func extractPredicateFieldsFromScalarExpression(ctx parser.IScalar_expressionContext, originalContainerName string, fromAlias string) [][]base.SelectorNode {
	if ctx == nil {
		return nil
	}

	switch {
	case ctx.Input_alias() != nil:
		name := ctx.Input_alias().Identifier().GetText()
		if fromAlias != "" && name == fromAlias {
			name = originalContainerName
		}
		return [][]base.SelectorNode{
			{
				base.NewItemSelector(name),
			},
		}
	case ctx.AND_SYMBOL() != nil && ctx.BETWEEN_SYMBOL() == nil:
		// AND logical operator (not BETWEEN...AND)
		allScalarExpressions := ctx.AllScalar_expression()
		var allPaths [][]base.SelectorNode
		for _, expr := range allScalarExpressions {
			paths := extractPredicateFieldsFromScalarExpression(expr, originalContainerName, fromAlias)
			allPaths = append(allPaths, paths...)
		}
		return allPaths
	case ctx.OR_SYMBOL() != nil:
		allScalarExpressions := ctx.AllScalar_expression()
		var allPaths [][]base.SelectorNode
		for _, expr := range allScalarExpressions {
			paths := extractPredicateFieldsFromScalarExpression(expr, originalContainerName, fromAlias)
			allPaths = append(allPaths, paths...)
		}
		return allPaths
	case ctx.DOT_SYMBOL() != nil:
		// Most usual case like a.b.c.d.
		paths := extractPredicateFieldsFromScalarExpression(ctx.Scalar_expression(0), originalContainerName, fromAlias)
		for i := range paths {
			paths[i] = append(paths[i], base.NewItemSelector(ctx.Property_name().Identifier().GetText()))
		}
		return paths
	case ctx.LS_BRACKET_SYMBOL() != nil && ctx.IN_SYMBOL() == nil:
		// Bracket access like a["b"] or a[0], but NOT IN expression
		paths := extractPredicateFieldsFromScalarExpression(ctx.Scalar_expression(0), originalContainerName, fromAlias)
		for i := range paths {
			switch {
			case ctx.DOUBLE_QUOTE_STRING_LITERAL() != nil:
				text := ctx.DOUBLE_QUOTE_STRING_LITERAL().GetText()
				if len(text) > 1 {
					text = text[1 : len(text)-1]
				}
				paths[i] = append(paths[i], base.NewItemSelector(text))
			case ctx.SINGLE_QUOTE_STRING_LITERAL() != nil:
				text := ctx.SINGLE_QUOTE_STRING_LITERAL().GetText()
				if len(text) > 1 {
					text = text[1 : len(text)-1]
				}
				paths[i] = append(paths[i], base.NewItemSelector(text))
			case ctx.Array_index() != nil:
				if len(paths[i]) == 0 {
					break
				}
				index, err := strconv.Atoi(ctx.Array_index().GetText())
				if err != nil {
					slog.Warn("cannot convert array index to int", slog.String("index", ctx.Array_index().GetText()))
					break
				}
				// Rebuild the ast because of the different level of array index and array name.
				last := paths[i][len(paths[i])-1]
				paths[i][len(paths[i])-1] = base.NewArraySelector(last.GetIdentifier(), index)
			default:
				// Do nothing for other cases
			}
		}
		return paths
	case ctx.Unary_operator() != nil:
		return extractPredicateFieldsFromScalarExpression(ctx.Scalar_expression(0), originalContainerName, fromAlias)
	case ctx.NOT_SYMBOL() != nil && len(ctx.AllScalar_expression()) == 1:
		// NOT expression
		return extractPredicateFieldsFromScalarExpression(ctx.Scalar_expression(0), originalContainerName, fromAlias)
	case ctx.Comparison_operator() != nil:
		left := extractPredicateFieldsFromScalarExpression(ctx.Scalar_expression(0), originalContainerName, fromAlias)
		right := extractPredicateFieldsFromScalarExpression(ctx.Scalar_expression(1), originalContainerName, fromAlias)
		return append(left, right...)
	case ctx.Multiplicative_operator() != nil:
		left := extractPredicateFieldsFromScalarExpression(ctx.Scalar_expression(0), originalContainerName, fromAlias)
		right := extractPredicateFieldsFromScalarExpression(ctx.Scalar_expression(1), originalContainerName, fromAlias)
		return append(left, right...)
	case ctx.Additive_operator() != nil:
		left := extractPredicateFieldsFromScalarExpression(ctx.Scalar_expression(0), originalContainerName, fromAlias)
		right := extractPredicateFieldsFromScalarExpression(ctx.Scalar_expression(1), originalContainerName, fromAlias)
		return append(left, right...)
	case ctx.Shift_operator() != nil:
		left := extractPredicateFieldsFromScalarExpression(ctx.Scalar_expression(0), originalContainerName, fromAlias)
		right := extractPredicateFieldsFromScalarExpression(ctx.Scalar_expression(1), originalContainerName, fromAlias)
		return append(left, right...)
	case ctx.BIT_AND_SYMBOL() != nil:
		left := extractPredicateFieldsFromScalarExpression(ctx.Scalar_expression(0), originalContainerName, fromAlias)
		right := extractPredicateFieldsFromScalarExpression(ctx.Scalar_expression(1), originalContainerName, fromAlias)
		return append(left, right...)
	case ctx.BIT_XOR_SYMBOL() != nil:
		left := extractPredicateFieldsFromScalarExpression(ctx.Scalar_expression(0), originalContainerName, fromAlias)
		right := extractPredicateFieldsFromScalarExpression(ctx.Scalar_expression(1), originalContainerName, fromAlias)
		return append(left, right...)
	case ctx.BIT_OR_SYMBOL() != nil:
		left := extractPredicateFieldsFromScalarExpression(ctx.Scalar_expression(0), originalContainerName, fromAlias)
		right := extractPredicateFieldsFromScalarExpression(ctx.Scalar_expression(1), originalContainerName, fromAlias)
		return append(left, right...)
	case ctx.DOUBLE_BAR_SYMBOL() != nil:
		left := extractPredicateFieldsFromScalarExpression(ctx.Scalar_expression(0), originalContainerName, fromAlias)
		right := extractPredicateFieldsFromScalarExpression(ctx.Scalar_expression(1), originalContainerName, fromAlias)
		return append(left, right...)
	case ctx.IN_SYMBOL() != nil:
		// IN expression: scalar_expression [NOT] IN (scalar_expression, ...)
		var allPaths [][]base.SelectorNode
		for _, expr := range ctx.AllScalar_expression() {
			paths := extractPredicateFieldsFromScalarExpression(expr, originalContainerName, fromAlias)
			allPaths = append(allPaths, paths...)
		}
		return allPaths
	case ctx.BETWEEN_SYMBOL() != nil:
		// BETWEEN expression: scalar_expression [NOT] BETWEEN scalar_expression AND scalar_expression
		var allPaths [][]base.SelectorNode
		for _, expr := range ctx.AllScalar_expression() {
			paths := extractPredicateFieldsFromScalarExpression(expr, originalContainerName, fromAlias)
			allPaths = append(allPaths, paths...)
		}
		return allPaths
	case ctx.LIKE_SYMBOL() != nil:
		left := extractPredicateFieldsFromScalarExpression(ctx.Scalar_expression(0), originalContainerName, fromAlias)
		right := extractPredicateFieldsFromScalarExpression(ctx.Scalar_expression(1), originalContainerName, fromAlias)
		return append(left, right...)
	case ctx.QUESTION_MARK_SYMBOL() != nil:
		left := extractPredicateFieldsFromScalarExpression(ctx.Scalar_expression(0), originalContainerName, fromAlias)
		mid := extractPredicateFieldsFromScalarExpression(ctx.Scalar_expression(1), originalContainerName, fromAlias)
		right := extractPredicateFieldsFromScalarExpression(ctx.Scalar_expression(2), originalContainerName, fromAlias)
		return append(append(left, mid...), right...)
	case ctx.Scalar_function_expression() != nil:
		switch {
		case ctx.Scalar_function_expression().Udf_scalar_function_expression() != nil:
			allScalarExpressions := ctx.Scalar_function_expression().Udf_scalar_function_expression().AllScalar_expression()
			var paths [][]base.SelectorNode
			for _, expr := range allScalarExpressions {
				path := extractPredicateFieldsFromScalarExpression(expr, originalContainerName, fromAlias)
				paths = append(paths, path...)
			}
			return paths
		case ctx.Scalar_function_expression().Builtin_function_expression() != nil:
			allScalarExpressions := ctx.Scalar_function_expression().Builtin_function_expression().AllScalar_expression()
			var paths [][]base.SelectorNode
			for _, expr := range allScalarExpressions {
				path := extractPredicateFieldsFromScalarExpression(expr, originalContainerName, fromAlias)
				paths = append(paths, path...)
			}
			return paths
		default:
			return nil
		}
	case ctx.LR_BRACKET_SYMBOL() != nil && ctx.Select_() == nil && ctx.EXISTS_SYMBOL() == nil:
		// Parenthesized expression
		return extractPredicateFieldsFromScalarExpression(ctx.Scalar_expression(0), originalContainerName, fromAlias)
	default:
		// Return nil for unhandled cases (constants, parameters, subqueries, etc.)
	}

	return nil
}
