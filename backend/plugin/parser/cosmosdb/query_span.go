package cosmosdb

import (
	"context"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/cosmosdb-parser"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func init() {
	base.RegisterGetQuerySpan(storepb.Engine_COSMOSDB, GetQuerySpan)
}

func GetQuerySpan(_ context.Context, _ base.GetQuerySpanContext, statement, _, _ string, _ bool) (*base.QuerySpan, error) {
	return getQuerySpan(statement)
}

func getQuerySpan(statement string) (*base.QuerySpan, error) {
	parseResults, err := ParseCosmosDBQuery(statement)
	if err != nil {
		return nil, err
	}

	if len(parseResults) == 0 {
		return nil, errors.New("no parse result")
	}

	ast := parseResults[0].Tree
	querySpanResultListener := &querySpanResultListener{}
	antlr.ParseTreeWalkerDefault.Walk(querySpanResultListener, ast)
	if querySpanResultListener.err != nil {
		return nil, err
	}

	querySpanPredicatePathsListener := &querySpanPredicatePathsListener{}
	antlr.ParseTreeWalkerDefault.Walk(querySpanPredicatePathsListener, ast)
	if querySpanPredicatePathsListener.err != nil {
		return nil, err
	}
	return &base.QuerySpan{
		Results:        querySpanResultListener.result,
		PredicatePaths: querySpanPredicatePathsListener.predicatePaths,
	}, nil
}

type querySpanResultListener struct {
	*parser.BaseCosmosDBParserListener

	result []base.QuerySpanResult

	err error
}

func (l *querySpanResultListener) EnterSelect(ctx *parser.SelectContext) {
	// TODO(zp): Considering the case of multiple from sources once we support it.
	if ctx.Select_clause().Select_specification().MULTIPLY_OPERATOR() != nil {
		l.result = []base.QuerySpanResult{
			{
				Name:             "",
				SourceFieldPaths: map[string]bool{},
			},
		}
		return
	}
}

type querySpanPredicatePathsListener struct {
	*parser.BaseCosmosDBParserListener

	predicatePaths map[string]bool
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

	predicateFields, err := extractPredicateFieldsFromWhereClause(whereClause, originalContainerName, fromIdentifier)
	if err != nil {
		l.err = err
		return
	}
	l.predicatePaths = predicateFields
}

func extractPredicateFieldsFromWhereClause(ctx parser.IWhere_clauseContext, originalContainerName string, fromAlias string) (map[string]bool, error) {
	scalarExpression := ctx.Scalar_expression_in_where()

	paths := extractPredicateFieldsFromScalarExpressionInWhere(scalarExpression, originalContainerName, fromAlias)

	r := make(map[string]bool)
	for _, path := range paths {
		r[path] = true
	}
	return r, nil
}

func extractPredicateFieldsFromScalarExpressionInWhere(ctx parser.IScalar_expression_in_whereContext, originalContainerName string, fromAlias string) []string {
	if ctx == nil {
		return nil
	}

	switch {
	case ctx.Input_alias() != nil:
		name := ctx.Input_alias().IDENTIFIER().GetText()
		if fromAlias != "" && name == fromAlias {
			name = originalContainerName
		}
		return []string{name}
	case ctx.AND_SYMBOL() != nil, ctx.OR_SYMBOL() != nil:
		allScalarExpressionInWheres := ctx.AllScalar_expression_in_where()
		var paths []string
		for _, expr := range allScalarExpressionInWheres {
			path := extractPredicateFieldsFromScalarExpressionInWhere(expr, originalContainerName, fromAlias)
			paths = append(paths, path...)
		}
		return paths
	case ctx.DOT_SYMBOL() != nil:
		// Most usual case like a.b.c.d.
		paths := extractPredicateFieldsFromScalarExpressionInWhere(ctx.Scalar_expression_in_where(0), originalContainerName, fromAlias)
		for i, path := range paths {
			paths[i] = path + "." + ctx.Property_name().IDENTIFIER().GetText()
		}
		return paths
	case ctx.LS_BRACKET_SYMBOL() != nil:
		paths := extractPredicateFieldsFromScalarExpressionInWhere(ctx.Scalar_expression_in_where(0), originalContainerName, fromAlias)
		for i, path := range paths {
			switch {
			case ctx.Property_name() != nil:
				paths[i] = path + "." + ctx.Property_name().IDENTIFIER().GetText()
			case ctx.Array_index() != nil:
				paths[i] = path + "[" + ctx.Array_index().GetText() + "]"
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
			var paths []string
			for _, expr := range allScalarExpressionInWheres {
				path := extractPredicateFieldsFromScalarExpressionInWhere(expr, originalContainerName, fromAlias)
				paths = append(paths, path...)
			}
			return paths
		case ctx.Scalar_function_expression().Builtin_function_expression() != nil:
			allScalarExpressionInWheres := ctx.Scalar_function_expression().Builtin_function_expression().AllScalar_expression_in_where()
			var paths []string
			for _, expr := range allScalarExpressionInWheres {
				path := extractPredicateFieldsFromScalarExpressionInWhere(expr, originalContainerName, fromAlias)
				paths = append(paths, path...)
			}
			return paths
		}
	case ctx.LR_BRACKET_SYMBOL() != nil:
		return extractPredicateFieldsFromScalarExpressionInWhere(ctx.Scalar_expression_in_where(0), originalContainerName, fromAlias)
	}

	return nil
}
