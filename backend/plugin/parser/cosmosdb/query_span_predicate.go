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
	if i := fromClause.From_specification().From_source().Container_expression().Container_name().IDENTIFIER(); i != nil {
		originalContainerName = i.GetText()
	}
	// Alias in the from source will shadow the original identifier.
	if i := fromClause.From_specification().From_source().Container_expression().IDENTIFIER(); i != nil {
		fromIdentifier = i.GetText()
	}

	predicateFields := extractPredicateFieldsFromWhereClause(whereClause, originalContainerName, fromIdentifier)
	l.predicatePaths = predicateFields
}

func extractPredicateFieldsFromWhereClause(ctx parser.IWhere_clauseContext, originalContainerName string, fromAlias string) map[string]*base.PathAST {
	scalarExpression := ctx.Scalar_expression_in_where()

	paths := extractPredicateFieldsFromScalarExpressionInWhere(scalarExpression, originalContainerName, fromAlias)

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

func extractPredicateFieldsFromScalarExpressionInWhere(ctx parser.IScalar_expression_in_whereContext, originalContainerName string, fromAlias string) [][]base.SelectorNode {
	if ctx == nil {
		return nil
	}

	switch {
	case ctx.Input_alias() != nil:
		name := ctx.Input_alias().IDENTIFIER().GetText()
		if fromAlias != "" && name == fromAlias {
			name = originalContainerName
		}
		return [][]base.SelectorNode{
			{
				base.NewItemSelector(name),
			},
		}
	case ctx.AND_SYMBOL() != nil, ctx.OR_SYMBOL() != nil:
		allScalarExpressionInWheres := ctx.AllScalar_expression_in_where()
		var allPaths [][]base.SelectorNode
		for _, expr := range allScalarExpressionInWheres {
			paths := extractPredicateFieldsFromScalarExpressionInWhere(expr, originalContainerName, fromAlias)
			allPaths = append(allPaths, paths...)
		}
		return allPaths
	case ctx.DOT_SYMBOL() != nil:
		// Most usual case like a.b.c.d.
		paths := extractPredicateFieldsFromScalarExpressionInWhere(ctx.Scalar_expression_in_where(0), originalContainerName, fromAlias)
		for i := range paths {
			paths[i] = append(paths[i], base.NewItemSelector(ctx.Property_name().IDENTIFIER().GetText()))
		}
		return paths
	case ctx.LS_BRACKET_SYMBOL() != nil:
		paths := extractPredicateFieldsFromScalarExpressionInWhere(ctx.Scalar_expression_in_where(0), originalContainerName, fromAlias)
		for i := range paths {
			switch {
			case ctx.Property_name() != nil:
				paths[i] = append(paths[i], base.NewItemSelector(ctx.Property_name().IDENTIFIER().GetText()))
				paths[i] = append(paths[i], base.NewItemSelector(ctx.Array_index().GetText()))
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
		return extractPredicateFieldsFromScalarExpressionInWhere(ctx.Scalar_expression_in_where(0), originalContainerName, fromAlias)
	case ctx.Binary_operator() != nil:
		left := extractPredicateFieldsFromScalarExpressionInWhere(ctx.Scalar_expression_in_where(0), originalContainerName, fromAlias)
		right := extractPredicateFieldsFromScalarExpressionInWhere(ctx.Scalar_expression_in_where(1), originalContainerName, fromAlias)
		return append(left, right...)
	case ctx.QUESTION_MARK_SYMBOL() != nil:
		left := extractPredicateFieldsFromScalarExpressionInWhere(ctx.Scalar_expression_in_where(0), originalContainerName, fromAlias)
		mid := extractPredicateFieldsFromScalarExpressionInWhere(ctx.Scalar_expression_in_where(1), originalContainerName, fromAlias)
		right := extractPredicateFieldsFromScalarExpressionInWhere(ctx.Scalar_expression_in_where(2), originalContainerName, fromAlias)
		return append(append(left, mid...), right...)
	case ctx.Scalar_function_expression() != nil:
		switch {
		case ctx.Scalar_function_expression().Udf_scalar_function_expression() != nil:
			allScalarExpressionInWheres := ctx.Scalar_function_expression().Udf_scalar_function_expression().AllScalar_expression_in_where()
			var paths [][]base.SelectorNode
			for _, expr := range allScalarExpressionInWheres {
				path := extractPredicateFieldsFromScalarExpressionInWhere(expr, originalContainerName, fromAlias)
				paths = append(paths, path...)
			}
			return paths
		case ctx.Scalar_function_expression().Builtin_function_expression() != nil:
			allScalarExpressionInWheres := ctx.Scalar_function_expression().Builtin_function_expression().AllScalar_expression_in_where()
			var paths [][]base.SelectorNode
			for _, expr := range allScalarExpressionInWheres {
				path := extractPredicateFieldsFromScalarExpressionInWhere(expr, originalContainerName, fromAlias)
				paths = append(paths, path...)
			}
			return paths
		default:
			// Other scalar function expression types
			return nil
		}
	case ctx.LR_BRACKET_SYMBOL() != nil:
		return extractPredicateFieldsFromScalarExpressionInWhere(ctx.Scalar_expression_in_where(0), originalContainerName, fromAlias)
	default:
		// Return nil for unhandled cases
	}

	return nil
}
