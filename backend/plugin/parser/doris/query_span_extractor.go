package doris

import (
	"context"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/doris-parser"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

type querySpanExtractor struct {
	defaultDatabase string
}

func newQuerySpanExtractor(database string, _ base.GetQuerySpanContext, _ bool) *querySpanExtractor {
	return &querySpanExtractor{
		defaultDatabase: database,
	}
}

func (q *querySpanExtractor) getQuerySpan(_ context.Context, statement string) (*base.QuerySpan, error) {
	parseResult, err := ParseDorisSQL(statement)
	if err != nil {
		return nil, err
	}

	if parseResult == nil {
		return &base.QuerySpan{
			SourceColumns: base.SourceColumnSet{},
			Results:       []base.QuerySpanResult{},
		}, nil
	}

	accessTables := getAccessTables(q.defaultDatabase, parseResult)

	queryTypeListener := &queryTypeListener{
		result: base.QueryTypeUnknown,
	}
	antlr.ParseTreeWalkerDefault.Walk(queryTypeListener, parseResult.Tree)

	return &base.QuerySpan{
		Type:          queryTypeListener.result,
		SourceColumns: accessTables,
		Results:       []base.QuerySpanResult{},
	}, nil
}

func getAccessTables(database string, parseResult *ParseResult) base.SourceColumnSet {
	accessTableListener := newAccessTableListener(database)
	antlr.ParseTreeWalkerDefault.Walk(accessTableListener, parseResult.Tree)

	return accessTableListener.sourceColumnSet
}

type accessTableListener struct {
	*parser.BaseDorisSQLListener

	defaultDatabase string
	sourceColumnSet base.SourceColumnSet
}

func newAccessTableListener(database string) *accessTableListener {
	return &accessTableListener{
		defaultDatabase: database,
		sourceColumnSet: base.SourceColumnSet{},
	}
}

func (l *accessTableListener) EnterTableAtom(ctx *parser.TableAtomContext) {
	if ctx == nil {
		return
	}

	list := NormalizeQualifiedName(ctx.QualifiedName())
	switch len(list) {
	case 1:
		l.sourceColumnSet[base.ColumnResource{
			Database: l.defaultDatabase,
			Table:    list[0],
		}] = true
	case 2:
		l.sourceColumnSet[base.ColumnResource{
			Database: list[0],
			Table:    list[1],
		}] = true
	}
}
