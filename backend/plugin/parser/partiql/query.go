package partiql

import (
	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/partiql-parser"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func init() {
	base.RegisterQueryValidator(storepb.Engine_DYNAMODB, validateQuery)
}

func validateQuery(statement string) (bool, error) {
	parseResult, err := ParsePartiQL(statement)
	if err != nil {
		return false, err
	}
	if parseResult == nil {
		return false, nil
	}

	l := &queryValidateListener{
		valid: true,
	}
	antlr.ParseTreeWalkerDefault.Walk(l, parseResult.Tree)
	return l.valid, nil
}

type queryValidateListener struct {
	*parser.BasePartiQLParserListener

	valid bool
}

func (q *queryValidateListener) EnterRoot(ctx *parser.RootContext) {
	if !q.valid {
		return
	}
	if ctx.EXPLAIN() != nil {
		return
	}

	child := ctx.GetChild(0)
	if child == nil {
		return
	}
	switch child.(type) {
	case *parser.QueryDqlContext:
		return
	case *parser.QueryDdlContext:
		q.valid = false
		return
	case *parser.QueryDmlContext:
		q.valid = false
		return
	case *parser.QueryExecContext:
		q.valid = false
		return
	}
}
