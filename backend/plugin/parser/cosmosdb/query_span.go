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

func GetQuerySpan(_ context.Context, _ base.GetQuerySpanContext, statement, _, _ string, _ bool) (*base.QuerySpan, error) {
	return getQuerySpanImpl(statement)
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
				SourceFieldPaths: make(map[string]*base.PathAST),
				SelectAsterisk:   true,
			},
		}
		return
	}

	var originalContainerName string
	var fromIdentifier string
	fromClause := ctx.From_clause()
	if i := fromClause.From_specification().From_source().Container_expression().Container_name().IDENTIFIER(); i != nil {
		originalContainerName = i.GetText()
	}
	// Alias in the from source will shadow the original identifier.
	if i := fromClause.From_specification().From_source().Container_expression().IDENTIFIER(); i != nil {
		fromIdentifier = i.GetText()
	}

	sourceFieldPath := make(map[string]*base.PathAST)
	objectProperties := ctx.Select_clause().Select_specification().Object_property_list().AllObject_property()
	for _, property := range objectProperties {
		path, name := extractPathFromObjectProperty(property, originalContainerName, fromIdentifier)
		if len(path) == 0 {
			continue
		}
		ast := base.NewPathAST(path[0])
		next := ast.Root
		for i := 1; i < len(path); i++ {
			next.SetNext(path[i])
			next = next.GetNext()
		}
		sourceFieldPath[name] = ast
	}
	l.result = []base.QuerySpanResult{
		{
			Name:             "",
			SourceFieldPaths: sourceFieldPath,
			SelectAsterisk:   false,
		},
	}
}

func extractPathFromObjectProperty(ctx parser.IObject_propertyContext, originalContainerName string, fromAlias string) ([]base.SelectorNode, string) {
	if ctx == nil {
		return nil, ""
	}

	path := extractPathFromScalarExpression(ctx.Scalar_expression(), originalContainerName, fromAlias)
	var propertyName string
	if ctx.Property_alias() != nil {
		propertyName = ctx.Property_alias().IDENTIFIER().GetText()
	}

	if propertyName == "" {
		// If the property alias is not specified, we will use the last path element as the property name.
		if len(path) > 0 {
			last := path[len(path)-1]
			propertyName = last.GetIdentifier()
		}
	}

	return path, propertyName
}

func extractPathFromScalarExpression(ctx parser.IScalar_expressionContext, originalContainerName string, fromAlias string) []base.SelectorNode {
	if ctx == nil {
		return nil
	}

	switch {
	case ctx.Input_alias() != nil:
		name := ctx.Input_alias().IDENTIFIER().GetText()
		if fromAlias != "" && name == fromAlias {
			name = originalContainerName
		}
		return []base.SelectorNode{
			base.NewItemSelector(name),
		}
	case ctx.DOT_SYMBOL() != nil:
		// Most usual case like a.b.c.d.
		path := extractPathFromScalarExpression(ctx.Scalar_expression(), originalContainerName, fromAlias)
		path = append(path, base.NewItemSelector(ctx.Property_name().IDENTIFIER().GetText()))

		return path
	case ctx.LS_BRACKET_SYMBOL() != nil:
		path := extractPathFromScalarExpression(ctx.Scalar_expression(), originalContainerName, fromAlias)
		switch {
		case ctx.DOUBLE_QUOTE_STRING_LITERAL() != nil:
			text := ctx.DOUBLE_QUOTE_STRING_LITERAL().GetText()
			if len(text) > 1 {
				text = text[1 : len(text)-1]
			}
			path = append(path, base.NewItemSelector(text))
		case ctx.Array_index() != nil:
			if len(path) == 0 {
				break
			}
			index, err := strconv.Atoi(ctx.Array_index().GetText())
			if err != nil {
				slog.Warn("cannot convert array index to int", slog.String("index", ctx.Array_index().GetText()))
				break
			}
			// Rebuild the ast because of the different level of array index and array name.
			last := path[len(path)-1]
			path[len(path)-1] = base.NewArraySelector(last.GetIdentifier(), index)
		default:
			// Unsupported bracket expression type
		}

		return path
	case ctx.Unary_operator() != nil:
		return extractPathFromScalarExpression(ctx.Scalar_expression(), originalContainerName, fromAlias)
	default:
		// Unsupported scalar expression type
	}

	return nil
}
