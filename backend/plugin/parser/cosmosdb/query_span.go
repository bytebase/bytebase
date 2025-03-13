package cosmosdb

import (
	"context"
	"github.com/antlr4-go/antlr/v4"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	parser "github.com/bytebase/cosmosdb-parser"
	"github.com/pkg/errors"
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
	listener := &walkListener{}
	antlr.ParseTreeWalkerDefault.Walk(listener, ast)
	if listener.err != nil {
		return nil, err
	}
	return &base.QuerySpan{
		Results: listener.result,
	}, nil
}

type walkListener struct {
	*parser.BaseCosmosDBParserListener

	result []base.QuerySpanResult
	err    error
}

func (l *walkListener) EnterSelect(ctx *parser.SelectContext) {
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
