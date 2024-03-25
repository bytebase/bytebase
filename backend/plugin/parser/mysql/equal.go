package mysql

import (
	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/mysql-parser"
	"github.com/pkg/errors"
)

// IsViewTailEqual returns true if the two view tails are equal.
func IsViewTailEqual(a, b parser.IViewTailContext) bool {
	return a.GetText() == b.GetText()
}

type isViewTailEqualViewStmtListener struct {
	*parser.BaseMySQLParserListener

	target parser.IViewTailContext
	equal  bool
	err    error
}

func (l *isViewTailEqualViewStmtListener) EnterCreateView(ctx *parser.CreateViewContext) {
	if l.err != nil {
		return
	}
	p := ctx.GetParent()
	if _, ok := p.(*parser.CreateStatementContext); !ok {
		l.err = errors.New("Expecting CreateStatementContext as parent")
		return
	}
	pp := p.GetParent()
	if _, ok := pp.(*parser.SimpleStatementContext); !ok {
		l.err = errors.New("Expecting SimpleStatementContext as parent")
		return
	}
	ppp := pp.GetParent()
	if _, ok := ppp.(*parser.QueryContext); !ok {
		l.err = errors.New("Expecting QueryContext as parent")
		return
	}

	if IsViewTailEqual(l.target, ctx.ViewTail()) {
		l.equal = true
	}
}

// IsViewTailEqualViewStmt returns true if the view tail is equal to the view statement.
func IsViewTailEqualViewStmt(a parser.IViewTailContext, b string) (bool, error) {
	parseResults, err := ParseMySQL(b)
	if err != nil {
		return false, err
	}
	if len(parseResults) != 1 {
		return false, errors.Errorf("Expecting one statement, but got %d", len(parseResults))
	}

	stmt := parseResults[0]
	listener := &isViewTailEqualViewStmtListener{
		target: a,
	}
	antlr.ParseTreeWalkerDefault.Walk(listener, stmt.Tree)
	if listener.err != nil {
		return false, listener.err
	}

	return listener.equal, nil
}
