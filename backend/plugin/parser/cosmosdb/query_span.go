package cosmosdb

import (
	"context"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/cosmosdb"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterGetQuerySpan(storepb.Engine_COSMOSDB, GetQuerySpan)
}

func GetQuerySpan(_ context.Context, _ base.GetQuerySpanContext, stmt base.Statement, _, _ string, _ bool) (*base.QuerySpan, error) {
	return getQuerySpanImpl(stmt.Text)
}

func getQuerySpanImpl(statement string) (*base.QuerySpan, error) {
	parseResults, err := ParseCosmosDBQuery(statement)
	if err != nil {
		return nil, err
	}

	if len(parseResults) == 0 {
		return &base.QuerySpan{
			Results:        []base.QuerySpanResult{},
			PredicatePaths: nil,
		}, nil
	}

	if len(parseResults) != 1 {
		return nil, errors.Errorf("expecting only one statement to get query span, but got %d", len(parseResults))
	}

	ast := parseResults[0].Tree
	querySpanResultListener := &querySpanResultListener{}
	antlr.ParseTreeWalkerDefault.Walk(querySpanResultListener, ast)
	if querySpanResultListener.err != nil {
		return nil, querySpanResultListener.err
	}

	querySpanPredicatePathsListener := &querySpanPredicatePathsListener{}
	antlr.ParseTreeWalkerDefault.Walk(querySpanPredicatePathsListener, ast)
	if querySpanPredicatePathsListener.err != nil {
		return nil, querySpanPredicatePathsListener.err
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
	if ctx.Select_clause().Select_specification().MULTIPLY_OPERATOR() != nil || ctx.From_clause() == nil {
		l.result = []base.QuerySpanResult{
			{
				Name:             "",
				SourceFieldPaths: make(map[string][]*base.PathAST),
				SelectAsterisk:   true,
			},
		}
		return
	}

	containerName, fromAlias := extractFromNames(ctx.From_clause())
	sourceFieldPaths := make(map[string][]*base.PathAST)
	for _, property := range ctx.Select_clause().Select_specification().Object_property_list().AllObject_property() {
		paths, name := extractPathsFromObjectProperty(property, containerName, fromAlias)
		if name == "" {
			continue
		}
		for _, path := range paths {
			if len(path) == 0 {
				continue
			}
			sourceFieldPaths[name] = append(sourceFieldPaths[name], buildPathAST(path))
		}
	}
	l.result = []base.QuerySpanResult{
		{
			Name:             "",
			SourceFieldPaths: sourceFieldPaths,
			SelectAsterisk:   false,
		},
	}
}

// extractPathsFromObjectProperty extracts all source field paths and the output name
// from a SELECT object_property.
func extractPathsFromObjectProperty(ctx parser.IObject_propertyContext, containerName, fromAlias string) ([][]base.SelectorNode, string) {
	if ctx == nil {
		return nil, ""
	}
	paths := extractAllFieldPaths(ctx.Scalar_expression(), containerName, fromAlias)
	var propertyName string
	if ctx.Property_alias() != nil {
		propertyName = ctx.Property_alias().Identifier().GetText()
	}
	if propertyName == "" && len(paths) == 1 && len(paths[0]) > 0 {
		propertyName = paths[0][len(paths[0])-1].GetIdentifier()
	}
	return paths, propertyName
}

// extractAllFieldPaths recursively collects ALL field paths referenced in a scalar expression.
func extractAllFieldPaths(ctx parser.IScalar_expressionContext, containerName, fromAlias string) [][]base.SelectorNode {
	if ctx == nil {
		return nil
	}

	// Input alias (leaf).
	if ctx.Input_alias() != nil {
		return resolveInputAlias(ctx, containerName, fromAlias)
	}
	// Property access: a.b
	if ctx.DOT_SYMBOL() != nil {
		return extractFieldPathsDot(ctx, containerName, fromAlias)
	}
	// Bracket access: a["b"] or a[0]
	if ctx.LS_BRACKET_SYMBOL() != nil && ctx.IN_SYMBOL() == nil {
		return extractFieldPathsBracket(ctx, containerName, fromAlias)
	}
	// Unary / NOT
	if ctx.Unary_operator() != nil || (ctx.NOT_SYMBOL() != nil && len(ctx.AllScalar_expression()) == 1) {
		return extractAllFieldPaths(ctx.Scalar_expression(0), containerName, fromAlias)
	}
	// Function calls
	if ctx.Scalar_function_expression() != nil {
		return extractFieldPathsFromFunction(ctx.Scalar_function_expression(), containerName, fromAlias)
	}
	// Binary/multi-child expressions: collect from all children.
	if allExprs := ctx.AllScalar_expression(); len(allExprs) >= 2 {
		return collectFieldPathsFromAll(allExprs, containerName, fromAlias)
	}
	return nil
}

func extractFieldPathsDot(ctx parser.IScalar_expressionContext, containerName, fromAlias string) [][]base.SelectorNode {
	paths := extractAllFieldPaths(ctx.Scalar_expression(0), containerName, fromAlias)
	propName := ctx.Property_name().Identifier().GetText()
	for i := range paths {
		paths[i] = append(paths[i], base.NewItemSelector(propName))
	}
	return paths
}

func extractFieldPathsBracket(ctx parser.IScalar_expressionContext, containerName, fromAlias string) [][]base.SelectorNode {
	paths := extractAllFieldPaths(ctx.Scalar_expression(0), containerName, fromAlias)
	for i := range paths {
		appendBracketSelector(ctx, paths, i)
	}
	return paths
}

func extractFieldPathsFromFunction(ctx parser.IScalar_function_expressionContext, containerName, fromAlias string) [][]base.SelectorNode {
	if ctx == nil {
		return nil
	}
	var exprs []parser.IScalar_expressionContext
	switch {
	case ctx.Udf_scalar_function_expression() != nil:
		exprs = ctx.Udf_scalar_function_expression().AllScalar_expression()
	case ctx.Builtin_function_expression() != nil:
		exprs = ctx.Builtin_function_expression().AllScalar_expression()
	default:
		return nil
	}
	return collectFieldPathsFromAll(exprs, containerName, fromAlias)
}

func collectFieldPathsFromAll(exprs []parser.IScalar_expressionContext, containerName, fromAlias string) [][]base.SelectorNode {
	var all [][]base.SelectorNode
	for _, expr := range exprs {
		all = append(all, extractAllFieldPaths(expr, containerName, fromAlias)...)
	}
	return all
}
