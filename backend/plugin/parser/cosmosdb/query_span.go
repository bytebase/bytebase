package cosmosdb

import (
	"context"
	"log/slog"
	"strconv"

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
	// TODO(zp): Considering the case of multiple from sources once we support it.
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

	var originalContainerName string
	var fromIdentifier string
	fromClause := ctx.From_clause()
	if i := fromClause.From_specification().From_source().Container_expression().Container_name().Identifier(); i != nil {
		originalContainerName = i.GetText()
	}
	// Alias in the from source will shadow the original identifier.
	if i := fromClause.From_specification().From_source().Container_expression().Identifier(); i != nil {
		fromIdentifier = i.GetText()
	}

	sourceFieldPaths := make(map[string][]*base.PathAST)
	objectProperties := ctx.Select_clause().Select_specification().Object_property_list().AllObject_property()
	for _, property := range objectProperties {
		paths, name := extractPathsFromObjectProperty(property, originalContainerName, fromIdentifier)
		if name == "" {
			continue
		}
		for _, path := range paths {
			if len(path) == 0 {
				continue
			}
			ast := base.NewPathAST(path[0])
			next := ast.Root
			for i := 1; i < len(path); i++ {
				next.SetNext(path[i])
				next = next.GetNext()
			}
			sourceFieldPaths[name] = append(sourceFieldPaths[name], ast)
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
// from a SELECT object_property. For direct field references (c.name), this returns
// one path. For expressions (UPPER(c.email), c.a || c.b), this returns all referenced
// field paths so masking can check if any source field is sensitive.
func extractPathsFromObjectProperty(ctx parser.IObject_propertyContext, originalContainerName string, fromAlias string) ([][]base.SelectorNode, string) {
	if ctx == nil {
		return nil, ""
	}

	paths := extractAllFieldPaths(ctx.Scalar_expression(), originalContainerName, fromAlias)
	var propertyName string
	if ctx.Property_alias() != nil {
		propertyName = ctx.Property_alias().Identifier().GetText()
	}

	if propertyName == "" {
		// If the property alias is not specified, we will use the last path element as the property name.
		// This only works for direct field references (single path).
		if len(paths) == 1 && len(paths[0]) > 0 {
			last := paths[0][len(paths[0])-1]
			propertyName = last.GetIdentifier()
		}
	}

	return paths, propertyName
}

// extractAllFieldPaths recursively collects ALL field paths referenced in a scalar expression.
// For direct field references (c.name), returns one path.
// For expressions (functions, operators), recurses into all children to find field references.
func extractAllFieldPaths(ctx parser.IScalar_expressionContext, originalContainerName string, fromAlias string) [][]base.SelectorNode {
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
			{base.NewItemSelector(name)},
		}
	case ctx.DOT_SYMBOL() != nil:
		paths := extractAllFieldPaths(ctx.Scalar_expression(0), originalContainerName, fromAlias)
		propName := ctx.Property_name().Identifier().GetText()
		for i := range paths {
			paths[i] = append(paths[i], base.NewItemSelector(propName))
		}
		return paths
	case ctx.LS_BRACKET_SYMBOL() != nil && ctx.IN_SYMBOL() == nil:
		paths := extractAllFieldPaths(ctx.Scalar_expression(0), originalContainerName, fromAlias)
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
				last := paths[i][len(paths[i])-1]
				paths[i][len(paths[i])-1] = base.NewArraySelector(last.GetIdentifier(), index)
			default:
			}
		}
		return paths
	case ctx.Unary_operator() != nil:
		return extractAllFieldPaths(ctx.Scalar_expression(0), originalContainerName, fromAlias)
	case ctx.NOT_SYMBOL() != nil && len(ctx.AllScalar_expression()) == 1:
		return extractAllFieldPaths(ctx.Scalar_expression(0), originalContainerName, fromAlias)
	case ctx.Scalar_function_expression() != nil:
		return extractFieldPathsFromFunctionExpression(ctx.Scalar_function_expression(), originalContainerName, fromAlias)
	}

	// Binary operators and other multi-child expressions: collect from all children.
	allExprs := ctx.AllScalar_expression()
	if len(allExprs) >= 2 {
		var all [][]base.SelectorNode
		for _, child := range allExprs {
			all = append(all, extractAllFieldPaths(child, originalContainerName, fromAlias)...)
		}
		return all
	}

	return nil
}

func extractFieldPathsFromFunctionExpression(ctx parser.IScalar_function_expressionContext, originalContainerName string, fromAlias string) [][]base.SelectorNode {
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
	var all [][]base.SelectorNode
	for _, expr := range exprs {
		all = append(all, extractAllFieldPaths(expr, originalContainerName, fromAlias)...)
	}
	return all
}
